package main // Main function

import (
	"PBPhotosShare/controllers"
	"app"
	"app/database"
	"app/middlewares"
	"app/security"
	"app/user"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kidstuff/mongostore"
)

var db *dbm.MongoDatabaseSession
var sessionStore *mongostore.MongoStore
var userManager user.Manager

const domainName = "127.0.0.1"
const domainPort = "8080"
const isSecure = true

func fullDomain() string {
	tmpPort := ""
	if len(domainPort) >= 0 {
		tmpPort = ":" + domainPort
	}

	tmpProtocol := "http"
	if isSecure == true {
		tmpProtocol += "s"
	}
	return tmpProtocol + "://" + domainName + tmpPort
}

func main() {
	// displayEnv()
	db = setupDB()
	defer db.Close()
	sessionStore = setupMongostore()
	userManager = setupUserManager()
	setupApp()
}

func setupUserManager() user.Manager {
	return user.NewDBManage(db, security.NewAuth(security.Basic))
}

func setupMongostore() *mongostore.MongoStore {
	return mongostore.NewMongoStore(db.Database.C("sessions"), 3600, true, []byte("mariageP&BKey"))
}

func setupDB() *dbm.MongoDatabaseSession {
	db := dbm.NewMongoDatabaseSession("127.0.0.1", "27017", "mariageDB", "adminMariage", "adminMariage")
	db.Connect()
	return db
}

func setupApp() {

	config := &app.ServerConfig{Host: domainName, Port: domainPort, SSL: isSecure, SessionStore: sessionStore}
	app.Start(config, func() http.Handler {
		r := mux.NewRouter().StrictSlash(true)

		mediaCtr := &controllers.MediaController{DomainBase: fullDomain()}
		mediaCtr.LoadController(r, db)

		userCtr := &controllers.UserController{DomainBase: fullDomain()}
		userCtr.LoadController(r, db, userManager)

		sub := r.PathPrefix("/images").Subrouter()
		logger := NewLogger(http.Dir("./static/"))
		sub.Handle("/{rest}", logger)
		sub.Handle("/thumbnail/{rest}", logger)
		log.Printf("Server  %s on port %s started using SSL %t", config.Host, config.Port, config.SSL)

		handler := middlewares.NewMiddlewares()
		handler.UseFunc(middlewares.NewAccessControlMiddleware(func(r *http.Request) bool { return true }))
		handler.UseHandler(r)
		return handler
	})
}

type logger struct {
	parent http.Handler
}

func NewLogger(filesys http.FileSystem) *logger {
	return &logger{parent: http.FileServer(filesys)}
}

func (u *logger) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	log.Println("request file", r.URL)
	u.parent.ServeHTTP(w, r)
}
