package controllers

import (
	"PBPhotosShare/models"
	"app"
	"app/database"
	"app/middlewares"
	"errors"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gorilla/mux"
)

func check(err error, next func()) {
	if err != nil {
		log.Println("error uploading file", err)
		next()
	}
}

//MediaController represent media management controller
type MediaController struct {
	DomainBase string
	db         dbm.DatabaseQuerier
}

//BasePath base path used for the routes of the controller
func (m *MediaController) BasePath() string {
	return "/api/media"
}

//GetName Name of the controller
func (m *MediaController) GetName() string {
	return "MediaController"
}

//LoadController Middleware of the controller
func (m *MediaController) LoadController(r *mux.Router, db dbm.DatabaseQuerier) {
	m.db = db
	sub := r.PathPrefix(m.BasePath()).Subrouter()
	sub.Handle("/updateDB", middlewares.NewMiddlewaresFunc(m.CreatesMediasFromDirectory)).Methods("GET")
	sub.Handle("/random", middlewares.NewMiddlewaresFunc(m.GetRandomMedia)).Methods("GET")
	sub.Handle("/{page}", middlewares.NewMiddlewaresFunc(m.GetMedias)).Methods("GET")
	sub.Handle("/", middlewares.NewMiddlewaresFunc(m.UploadPrivateImage)).Methods("POST")
	sub.Handle("/", middlewares.NewMiddlewaresFunc(m.GetMedias)).Methods("GET")
}

//CreatesMediasFromDirectory generate Media model in DB from file in  "./static/images/"
func (m *MediaController) CreatesMediasFromDirectory(w http.ResponseWriter, r *http.Request, next func()) {
	filePath := "./static/images/"
	url := "/images/"
	filepath.Walk(filePath, func(path string, f os.FileInfo, err error) error {
		media := model.Media{}
		m.db.GetOneModel(dbm.M{"Name": f.Name()}, media)
		if media != (model.Media{}) {
			log.Println(media)
			return nil
		}
		media = model.Media{
			Name:          f.Name(),
			Path:          filePath,
			ThumbnailPath: filePath + "/thumbnail",
			URL:           url,
			ThumbnailURL:  url + "/thumbnail",
		}
		m.db.InsertModel(media)
		return nil
	})
	app.JSONResp(w, struct{ Response string }{"Ok"})
}

//GetMedias used to retriev 20 medias from the database, the page must be add in parameter
func (m *MediaController) GetMedias(w http.ResponseWriter, r *http.Request, next func()) {
	vars := mux.Vars(r)
	if pagestr := vars["page"]; pagestr == "" {
		pagestr = "0"
	} else if pagestr == "random" {
		next()
	} else {
		if page, err := strconv.Atoi(pagestr); err == nil {
			medias := make([]model.Media, 0, 40)
			if mediasInterface, err := m.db.GetModels(nil, medias, 40, page*40); err == nil {
				medias := mediasInterface.([]model.Media)
				for index, element := range medias {
					medias[index].URL = m.DomainBase + element.URL
					medias[index].ThumbnailURL = m.DomainBase + element.ThumbnailURL
				}
				app.JSONResp(w, medias)
			} else {
				log.Println(err)
				next()
			}
		} else {
			log.Println(err)
			next()
		}
	}
}

//DownloadMedias get all files name to DownloadMedias
// It will search in ./static.images the files
// It will zip them
// Open a dialog box to download
func (m *MediaController) DownloadMedias(w http.ResponseWriter, r *http.Request, next func()) {
	//Open Dialog box
	w.Header().Set("Content-Disposition", "attachment; filename=WHATEVER_YOU_WANT")
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
}

//GetRandomMedia return an image choosed randomly in the Database
func (m *MediaController) GetRandomMedia(w http.ResponseWriter, r *http.Request, next func()) {
	media := &model.Media{}
	if dbMongo, ok := m.db.(*dbm.MongoDatabaseSession); ok == true {
		err := dbMongo.GetRandomOneModel(media)
		media.URL = m.DomainBase + media.URL
		media.ThumbnailURL = m.DomainBase + media.ThumbnailURL
		if err != nil {
			log.Println(err)
			app.JSONResp(w, app.RequestError{"Error random Image", "error while retrieve a random image", 1})
		} else {
			app.JSONResp(w, media)
		}
	}
}

//UploadPrivateImage Upload image to the filesystem
func (m *MediaController) UploadPrivateImage(w http.ResponseWriter, r *http.Request, next func()) {
	files, err := m.handleUploads(r)
	if err != nil {
		log.Println("error handleUploads", err)
		app.JSONResp(w, app.RequestError{"Error Upload", "Didn't Upload", 0})
	} else {
		var genralMedia []interface{}
		for _, elem := range files {
			genralMedia = append(genralMedia, elem)
		}
		errorsInsert := m.db.InsertModel(genralMedia...)
		for err := range errorsInsert {
			log.Println(err)
		}
		app.JSONResp(w, files)
	}
}

func (m *MediaController) handleUpload(r *http.Request, p *multipart.Part) (*model.Media, error) {
	filenameDecoded, _ := url.QueryUnescape(p.FileName())
	return model.NewMedia(p, filenameDecoded, "./static/images", "/images")
}

func (m *MediaController) handleUploads(r *http.Request) ([]*model.Media, error) {
	var filesInfos []*model.Media
	mr, err := r.MultipartReader()
	if err != nil {
		log.Println("error Getting MultipartReader", err)
		return nil, errors.New("Cannot read Multipart")
	}
	part, err := mr.NextPart()
	for err == nil {
		if name := part.FormName(); name != "" {
			if part.FileName() != "" {
				if mediaInfos, err := m.handleUpload(r, part); err == nil {
					filesInfos = append(filesInfos, mediaInfos)
				} else {
					log.Println("error handle file ", part.FileName(), "-->", err)
				}
			}
		}
		part, err = mr.NextPart()
	}
	return filesInfos, nil
}

func (m *MediaController) handleMultiPartForm(r *http.Request, formFields []string) ([]*model.Media, error) {
	var filesInfos []*model.Media
	r.ParseMultipartForm(int64(len(formFields) * model.MaxFileSize))
	for _, formField := range formFields {
		file, handler, err := r.FormFile(formField)
		if err != nil {
			log.Println("FormFile error ", err)
			continue
		}
		defer file.Close()
		filenameDecoded, _ := url.QueryUnescape(handler.Filename)
		fi, errorMedia := model.NewMedia(file, filenameDecoded, "./static/images", "/images")
		if errorMedia != nil {
			continue
		}
		filesInfos = append(filesInfos, fi)
	}
	return filesInfos, nil
}
