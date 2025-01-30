package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/jake-t-dev/user-mgt-system.git/pkg/handlers"
)

var tmpl *template.Template
var db *sql.DB
var Store = sessions.NewCookieStore([]byte("usermanagementsecret"))

func init() {
	tmpl, _ = template.ParseGlob("templates/*.html")

	Store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   3600 * 3,
		HttpOnly: true,
	}

}

func initDB() {
	var err error
	db, err = sql.Open("mysql", "root:root@(127.0.0.1:3333)/user_management?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}
}

func main() {

	router := mux.NewRouter()

	initDB()
	defer db.Close()

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		tmpl.ExecuteTemplate(w, "home.html", nil)

	})

	router.HandleFunc("/register", handlers.RegisterPage(db, tmpl)).Methods("GET")

	router.HandleFunc("/register", handlers.RegisterHandler(db, tmpl)).Methods("POST")

	http.ListenAndServe(":4000", router)

}
