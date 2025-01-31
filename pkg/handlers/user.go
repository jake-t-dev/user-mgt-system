package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/uuid"
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

func Editpage(db *sql.DB, tmpl *template.Template, store *sessions.CookieStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		user, _ := CheckLoggedIn(w, r, store, db)

		if err := tmpl.ExecuteTemplate(w, "editProfile", user); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

func UpdateProfileHandler(db *sql.DB, tmpl *template.Template, store *sessions.CookieStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		currentUserProfile, userID := CheckLoggedIn(w, r, store, db)

		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		var errorMessages []string

		name := r.FormValue("name")
		bio := r.FormValue("bio")
		dobStr := r.FormValue("dob")

		if name == "" {
			errorMessages = append(errorMessages, "Name is required.")
		}

		if dobStr == "" {
			errorMessages = append(errorMessages, "Date of birth is required.")
		}

		dob, err := time.Parse("2006-01-02", dobStr)
		if err != nil {
			errorMessages = append(errorMessages, "Invalid date format.")
		}

		if len(errorMessages) > 0 {
			tmpl.ExecuteTemplate(w, "autherrors", errorMessages)
			return
		}

		user := models.User{
			Id:       userID,
			Name:     name,
			DOB:      dob,
			Bio:      bio,
			Category: currentUserProfile.Category,
		}

		if err := repository.UpdateUser(db, userID, user); err != nil {
			errorMessages = append(errorMessages, "Failed to update user")
			tmpl.ExecuteTemplate(w, "autherrors", errorMessages)
			log.Fatal(err)

			return
		}

		w.Header().Set("HX-Location", "/")
		w.WriteHeader(http.StatusNoContent)
	}
}

func AvatarPage(db *sql.DB, tmpl *template.Template, store *sessions.CookieStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		user, _ := CheckLoggedIn(w, r, store, db)

		if err := tmpl.ExecuteTemplate(w, "uploadAvatar", user); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

func UploadAvatarHandler(db *sql.DB, tmpl *template.Template, store *sessions.CookieStore) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		user, userID := CheckLoggedIn(w, r, store, db)

		var errorMessages []string

		r.ParseMultipartForm(10 << 20)

		file, handler, err := r.FormFile("avatar")
		if err != nil {
			if err == http.ErrMissingFile {
				errorMessages = append(errorMessages, "No file submitted")
			} else {
				errorMessages = append(errorMessages, "Error retrieving the file")
			}

			if len(errorMessages) > 0 {
				tmpl.ExecuteTemplate(w, "autherrors", errorMessages)
				return
			}

		}
		defer file.Close()

		uuid, err := uuid.NewRandom()
		if err != nil {
			errorMessages = append(errorMessages, "Error generating unique identifier")
			tmpl.ExecuteTemplate(w, "autherrors", errorMessages)

			return
		}
		filename := uuid.String() + filepath.Ext(handler.Filename)

		filePath := filepath.Join("uploads", filename)

		dst, err := os.Create(filePath)
		if err != nil {
			errorMessages = append(errorMessages, "Error saving the file")
			tmpl.ExecuteTemplate(w, "autherrors", errorMessages)

			return
		}
		defer dst.Close()
		if _, err = io.Copy(dst, file); err != nil {
			errorMessages = append(errorMessages, "Error saving the file")
			tmpl.ExecuteTemplate(w, "autherrors", errorMessages)
			return
		}

		if err := repository.UpdateUserAvatar(db, userID, filename); err != nil {
			errorMessages = append(errorMessages, "Error updating user avatar")
			tmpl.ExecuteTemplate(w, "autherrors", errorMessages)

			log.Fatal(err)
			return
		}

		if user.Avatar != "" {
			oldAvatarPath := filepath.Join("uploads", user.Avatar)

			if oldAvatarPath != filePath {
				if err := os.Remove(oldAvatarPath); err != nil {
					fmt.Printf("Warning: failed to delete old avatar file: %s\n", err)
				}
			}
		}

		w.Header().Set("HX-Location", "/")
		w.WriteHeader(http.StatusNoContent)
	}
}

func LogoutHandler(store *sessions.CookieStore) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "logged-in-user")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		delete(session.Values, "user_id")

		if err = session.Save(r, w); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		session.Options.MaxAge = -1
		session.Save(r, w)

		http.Redirect(w, r, "/login", http.StatusSeeOther)
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
