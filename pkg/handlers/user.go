package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/sessions"
	"github.com/jake-t-dev/user-mgt-system.git/pkg/models"
	"github.com/jake-t-dev/user-mgt-system.git/pkg/repository"
	"golang.org/x/crypto/bcrypt"
)

func RegisterPage(db *sql.DB, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		tmpl.ExecuteTemplate(w, "register", nil)

	}
}

func RegisterHandler(db *sql.DB, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user models.User
		var errorMessages []string

		r.ParseForm()

		user.Name = r.FormValue("name")
		user.Email = r.FormValue("email")
		user.Password = r.FormValue("password")
		user.Category, _ = strconv.Atoi(r.FormValue("category"))

		if user.Name == "" {
			errorMessages = append(errorMessages, "Name is required.")
		}
		if user.Email == "" {
			errorMessages = append(errorMessages, "Email is required.")
		}
		if user.Password == "" {
			errorMessages = append(errorMessages, "Password is required.")
		}

		if len(errorMessages) > 0 {
			tmpl.ExecuteTemplate(w, "autherrors", errorMessages)
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			errorMessages = append(errorMessages, "Failed to hash password.")
			tmpl.ExecuteTemplate(w, "autherrors", errorMessages)
			return
		}
		user.Password = string(hashedPassword)

		user.DOB = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
		user.Bio = "Bio goes here"
		user.Avatar = ""

		err = repository.CreateUser(db, user)
		if err != nil {
			errorMessages = append(errorMessages, "Failed to create user: "+err.Error())
			tmpl.ExecuteTemplate(w, "autherrors", errorMessages)
			return
		}

		w.Header().Set("HX-Location", "/login")
		w.WriteHeader(http.StatusNoContent)
	}
}

func LoginPage(db *sql.DB, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		tmpl.ExecuteTemplate(w, "login", nil)

	}
}

func LoginHandler(db *sql.DB, tmpl *template.Template, store *sessions.CookieStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		email := r.FormValue("email")
		password := r.FormValue("password")

		var errorMessages []string

		if email == "" {
			errorMessages = append(errorMessages, "Email is required.")
		}
		if password == "" {
			errorMessages = append(errorMessages, "Password is required.")
		}

		if len(errorMessages) > 0 {
			tmpl.ExecuteTemplate(w, "autherrors", errorMessages)
			return
		}

		user, err := repository.GetUserByEmail(db, email)
		if err != nil {
			if err == sql.ErrNoRows {
				errorMessages = append(errorMessages, "Invalid email or password")
				tmpl.ExecuteTemplate(w, "autherrors", errorMessages)
				return
			}

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
		if err != nil {
			errorMessages = append(errorMessages, "Invalid email or password")
			tmpl.ExecuteTemplate(w, "autherrors", errorMessages)

			return
		}

		session, err := store.Get(r, "logged-in-user")
		if err != nil {
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
		session.Values["user_id"] = user.Id
		if err := session.Save(r, w); err != nil {
			http.Error(w, "Error saving session", http.StatusInternalServerError)
			return
		}

		w.Header().Set("HX-Location", "/")
		w.WriteHeader(http.StatusNoContent)
	}

}

func Homepage(db *sql.DB, tmpl *template.Template, store *sessions.CookieStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		user, _ := CheckLoggedIn(w, r, store, db)

		if err := tmpl.ExecuteTemplate(w, "home.html", user); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

func CheckLoggedIn(w http.ResponseWriter, r *http.Request, store *sessions.CookieStore, db *sql.DB) (models.User, string) {

	session, err := store.Get(r, "logged-in-user")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return models.User{}, ""
	}

	userID, ok := session.Values["user_id"]
	if !ok {

		fmt.Println("Redirecting to /login")
		http.Redirect(w, r, "/login", http.StatusSeeOther)

		return models.User{}, ""
	}

	user, err := repository.GetUserById(db, userID.(string))
	if err != nil {
		if err == sql.ErrNoRows {
			session.Options.MaxAge = -1
			session.Save(r, w)

			fmt.Println("Redirecting to /login")
			http.Redirect(w, r, "/login", http.StatusSeeOther)

			return models.User{}, ""
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return models.User{}, ""
	}

	return user, userID.(string)
}
