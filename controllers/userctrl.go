package controllers

import (
	"goappuser"
	"goappuser/database"
	"goappuser/middlewares"
	"goappuser/user"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

//LoginController manage login logout
type UserController struct {
	DomainBase  string
	db          dbm.DatabaseQuerier
	userManager user.Manager
}

//BasePath base path used for the routes of the controller
func (l *UserController) BasePath() string {
	return "/api/user"
}

//GetName Name of the controller
func (l *UserController) GetName() string {
	return "UserController"
}

//LoadController Middleware of the controller
func (l *UserController) LoadController(r *mux.Router, db dbm.DatabaseQuerier, userManager user.Manager) {
	l.db = db
	l.userManager = userManager
	sub := r.PathPrefix(l.BasePath()).Subrouter()
	sub.Handle("/login", middlewares.NewMiddlewaresFunc(l.Login)).Methods("POST")
	sub.Handle("/logout", middlewares.NewMiddlewaresFunc(l.Logout)).Methods("GET")
	sub.Handle("/", middlewares.NewMiddlewaresFunc(l.Register)).Methods("POST")
}

//Register a new user
func (l *UserController) Register(w http.ResponseWriter, r *http.Request, next func()) {

	email := r.FormValue("email")
	if len(email) <= 0 {
		app.JSONResp(w, app.RequestError{"Register", "Email missing", 0})
		next()
		return
	}
	password := r.FormValue("password")
	if len(password) <= 0 {
		app.JSONResp(w, app.RequestError{"Register", "Password missing", 1})
		next()
		return
	}
	log.Println("email ", email)
	user := user.NewUser(email, password)
	if err := l.userManager.Register(user); err != nil {
		app.JSONResp(w, app.RequestError{"Register", err.Error(), 2})
	} else {
		app.JSONResp(w, user)
	}
	next()
	return
}

//Login log in
func (l *UserController) Login(w http.ResponseWriter, r *http.Request, next func()) {
	if user, err := l.userManager.Authenticate(r); err != nil {
		app.JSONResp(w, app.RequestError{Title: "login error", Description: err.Error(), Code: 0})
	} else {
		//Session
		app.JSONResp(w, user)
	}
}

//Logout log out
func (l *UserController) Logout(w http.ResponseWriter, r *http.Request, next func()) {

}
