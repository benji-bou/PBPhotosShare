package model

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/nfnt/resize"
)

const (
	//MinFile minimum File Size
	MinFile = 1 // bytes
	// MaxFileSize file size is memcache limit (1MB) minus key size minus overhead:
	MaxFileSize = 244999000 // bytes
	//ImageType regxp accepted type
	ImageType = "((gif|p?jpeg|(x-)?png))"
	//AcceptFileTypes accepted files
	AcceptFileTypes = ImageType
	//ThumbMaxWidth max size
	ThumbMaxWidth = 200

	//ThumbMaxHeight max height
	ThumbMaxHeight = 0
)

var (
	acceptFileTypes = regexp.MustCompile(AcceptFileTypes)
	imageTypes      = regexp.MustCompile(ImageType)
)

//Media Represent a media file
type Media struct {
	Name          string `bson:"name" json:"name"`
	Path          string `bson:"path" json:"-"`
	ThumbnailPath string `bson:"thumbnailPath" json:"-"`
	MimeType      string `bson:"mimetype" json:"mimetype"`
	URL           string `bson:"url" json:"url,omitempty"`
	ThumbnailURL  string `bson:"thumbnailUrl" json:"thumbnailUrl,omitempty"`
	Size          int64  `bson:"size" json:"size"`
	pathToWrite   string
}

//NewMedia Create a new Media ptr
func NewMedia(inputFile io.Reader, filename, filePath, url string) (*Media, error) {
	media := &Media{
		Name:          fmt.Sprint(time.Now().Unix()) + "_" + filename,
		Path:          filePath,
		ThumbnailPath: filePath + "/thumbnail",
		URL:           url,
		ThumbnailURL:  url + "/thumbnail",
	}

	media.WriteImages(inputFile)
	return media, nil
}

func init() {
	image.RegisterFormat("jpeg", "jpeg", jpeg.Decode, jpeg.DecodeConfig)
	image.RegisterFormat("png", "png", png.Decode, png.DecodeConfig)
	image.RegisterFormat("gif", "gif", gif.Decode, gif.DecodeConfig)
}

func encodeImage(writeInto io.Writer, image image.Image, mimeType string) error {
	switch mimeType {
	case "jpeg", "pjpeg":
		return jpeg.Encode(writeInto, image, nil)
	case "gif":
		return gif.Encode(writeInto, image, nil)
	default:
		return png.Encode(writeInto, image)
	}
}

//WriteImages writes Images to filesystem normal and thumbnail
func (media *Media) WriteImages(input io.Reader) error {
	image, mimeType, err := image.Decode(input)
	if err != nil {
		return err
	}
	if isOk, err := media.ValidateType(mimeType); isOk == true {
		media.pathToWrite = media.Path + "/" + media.Name
		encodeImage(media, image, mimeType)
		thumb := resize.Resize(ThumbMaxWidth, ThumbMaxHeight, image, resize.Lanczos3)
		media.pathToWrite = media.ThumbnailPath + "/" + media.Name
		encodeImage(media, thumb, mimeType)
	} else {
		return err
	}
	return nil
}

//ValidateType check type of media
func (media *Media) ValidateType(typeMime string) (bool, error) {
	var err error

	// typeMime := http.DetectContentType(p)
	if acceptFileTypes.MatchString(typeMime) {
		media.MimeType = typeMime
		return true, err
	}
	err = errors.New("Filetype " + media.MimeType + " not allowed")
	return false, err
}

// ValidateSize check the size
func (media *Media) ValidateSize(n int64) (bool, error) {
	var err error
	if n < MinFile {
		err = errors.New("File is too small")
	} else if n > MaxFileSize {
		err = errors.New("File is too big")
	} else {
		media.Size = n
		return true, err
	}
	return false, err
}

func (media *Media) Write(p []byte) (int, error) {
	if media.Path == "" {
		return 0, errors.New("No file path")
	}
	return media.writeForPath(p, media.pathToWrite)
}

func (media *Media) writeForPath(p []byte, path string) (int, error) {
	if media.Path == "" {
		return 0, errors.New("No file path")
	}

	var fileOut *os.File
	var err error
	fileOut, err = os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0766)
	if err != nil {
		log.Println("open file error", err)
		return 0, errors.New("Unable to Open file")
	}
	defer fileOut.Close()
	reader := bytes.NewReader(p)
	n, errorCopy := io.Copy(fileOut, reader)
	return int(n), errorCopy
}
