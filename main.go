package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

var tmpl *template.Template
var db *sql.DB

func init() {
	tmpl, _ = template.ParseGlob("templates/*.html")

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

	http.ListenAndServe(":4000", router)

}
