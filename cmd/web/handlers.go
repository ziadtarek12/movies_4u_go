package main

import (
	"encoding/json"
	"errors"

	// "html/template"
	"net/http"
	"strconv"

	"movies4u.net/internals/models"
	"movies4u.net/internals/validator"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type userCreateForm struct {
	UserName string
	Email    string
	validator.Validator
}

func (app *application) setCSPHeader(w http.ResponseWriter) {
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; img-src *; style-src 'self' 'unsafe-inline';")
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	app.setCSPHeader(w)
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data := app.newTemplateData(r)
	app.render(w, http.StatusOK, "index.html", data)
}

func (app *application) search(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) searchPost(w http.ResponseWriter, r *http.Request) {
	var filmRequest struct {
		Film string
	}

	err := json.NewDecoder(r.Body).Decode(&filmRequest)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	var films []models.Film
	result := app.DB.Where("name LIKE ?", "%"+filmRequest.Film+"%").Find(&films)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			app.notFound(w)
		} else {
			app.serverError(w, err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(films); err != nil {
		app.serverError(w, err)
	}
}

func (app *application) getFilm(w http.ResponseWriter, r *http.Request) {
	app.infoLog.Printf("filmView: Method=%s, URL=%s", r.Method, r.URL)

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 0 {
		app.notFound(w)
		return
	}

	if id > 9999 {
		id = 1
	}

	var film models.Film
	result := app.DB.Preload("Genres").Preload("Directors").Preload("Stars").First(&film, id)

	if result.Error != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	filmJson, err := film.Json(app.DB)
	if err != nil {
		app.serverError(w, err)
		return
	}
	w.Write(filmJson)
}

func (app *application) getFilms(w http.ResponseWriter, r *http.Request) {
	app.infoLog.Printf("filmView: Method=%s, URL=%s", r.Method, r.URL)
	start := r.URL.Query().Get("start")
	finish := r.URL.Query().Get("end")
	var startId, finishId int

	if start == "" {
		startId = 1
	} else {
		var err error
		startId, err = strconv.Atoi(start)
		if err != nil {
			app.serverError(w, err)
			return
		}
	}

	if finish == "" {
		finishId = 1
	} else {
		var err error
		finishId, err = strconv.Atoi(finish)
		if err != nil {
			app.serverError(w, err)
			return
		}
	}

	var films []models.Film
	result := app.DB.Preload("Genres").Preload("Directors").Preload("Stars").Where("id BETWEEN ? AND ?", startId, finishId).Find(&films)

	if result.Error != nil {
		app.serverError(w, result.Error)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	var filmJsons []json.RawMessage
	for _, film := range films {
		filmJson, err := film.Json(app.DB)
		if err != nil {
			app.serverError(w, err)
			return
		}
		filmJsons = append(filmJsons, filmJson)
	}

	if err := json.NewEncoder(w).Encode(filmJsons); err != nil {
		app.serverError(w, err)
	}
}

func (app *application) methodNotAllowed(method string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		app.infoLog.Printf("methodNotAllowed: Method=%s, URL=%s", r.Method, r.URL)
		w.Header().Set("Allow", method)
		app.clientError(w, http.StatusMethodNotAllowed)
	}
}

func (app *application) userLogin(w http.ResponseWriter, r *http.Request) {
	app.infoLog.Printf("userLogin: Method=%s, URL=%s", r.Method, r.URL)
	data := app.newTemplateData(r)
	data.Form = userCreateForm{}
	app.render(w, http.StatusOK, "login.html", data)
}

func (app *application) userSignin(w http.ResponseWriter, r *http.Request) {
	app.infoLog.Printf("userSignin: Method=%s, URL=%s", r.Method, r.URL)
	data := app.newTemplateData(r)
	data.Form = userCreateForm{}
	app.render(w, http.StatusOK, "signin.html", data)
}

func (app *application) userSigninPost(w http.ResponseWriter, r *http.Request) {
	app.infoLog.Printf("userSigninPost: Method=%s, URL=%s", r.Method, r.URL)

	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	username := r.Form.Get("username")
	email := r.Form.Get("email")
	password := r.Form.Get("password")
	confirmPassword := r.Form.Get("confirm_password")

	form := userCreateForm{
		UserName: username,
		Email:    email,
	}

	// Validate form fields
	form.Validator.CheckField(validator.NotBlank(username), "username", "This field can't be blank")
	form.Validator.CheckField(validator.MinChars(username, 3), "username", "Username must be at least 3 characters")
	form.Validator.CheckField(validator.NotBlank(email), "email", "This field can't be blank")
	form.Validator.CheckField(validator.Matches(email, validator.EmailRX), "email", "This is not an email")
	form.Validator.CheckField(validator.NotBlank(password), "password", "This field can't be blank")
	form.Validator.CheckField(validator.MinChars(password, 8), "password", "Password must be at least 8 characters")
	form.Validator.CheckField(validator.NotBlank(confirmPassword), "confirm_password", "This field can't be blank")
	form.Validator.CheckField(validator.PasswordsMatch(password, confirmPassword), "password", "Passwords do not match")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "signin.html", data)
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		app.serverError(w, err)
		return
	}

	// Create a new user record
	user := models.User{
		UserName: username,
		Email:    email,
		Password: string(hashedPassword),
	}

	// Insert the user into the database
	result := app.DB.Create(&user)
	if result.Error != nil {
		app.serverError(w, result.Error)
		return
	}

	// Redirect to the login page
	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

func (app *application) userLoginPost(w http.ResponseWriter, r *http.Request) {
	app.infoLog.Printf("userLoginPost: Method=%s, URL=%s", r.Method, r.URL)
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	email, password := r.PostForm.Get("email"), r.PostForm.Get("password")

	form := userCreateForm{
		Email: email,
	}

	form.Validator.CheckField(validator.NotBlank(email), "email", "This field can't be blank")
	form.Validator.CheckField(validator.NotBlank(password), "password", "This field can't be blank")

	form.Validator.CheckField(validator.Matches(email, validator.EmailRX), "email", "This is not an email")
	form.Validator.CheckField(validator.MinChars(password, 8), "password", "Password must be at least 8 characters")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "login.html", data)
		return
	}
	app.infoLog.Println("I amhere")
	app.infoLog.Println(email + password)
	id, err := app.Authenticate(email, password)
	app.infoLog.Println("I amhere")
	if err != nil {
		if errors.Is(err, models.ErrInvalidCredentials) {
			form.AddNonFieldError("Email or password is incorrect")
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, http.StatusUnprocessableEntity, "login.html", data)
		} else {
			app.serverError(w, err)
		}
		return
	}

	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}
	app.sessionManager.Put(r.Context(), "userID", id)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) userLogoutPost(w http.ResponseWriter, r *http.Request) {

	err := app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}
	app.sessionManager.Remove(r.Context(), "userID")

	app.sessionManager.Put(r.Context(), "flash", "You've been logged ouu successfully!")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) putWatchlist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		app.clientError(w, http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		ID        uint `json:"id"`
		Watchlist bool `json:"watchlist"`
	}

	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	if body.ID > 9999 {
		app.clientError(w, http.StatusUnprocessableEntity)
		return
	}

	userID := app.sessionManager.GetInt(r.Context(), "userID")
	if userID == 0 {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	var user models.User
	result := app.DB.First(&user, userID)
	if result.Error != nil {
		app.serverError(w, result.Error)
		return
	}

	if body.Watchlist {
		app.DB.Model(&user).Association("WatchList").Delete(&models.Film{ID: body.ID})
	} else {
		app.DB.Model(&user).Association("WatchList").Append(&models.Film{ID: body.ID})
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(map[string]bool{"watchlist": body.Watchlist}); err != nil {
		app.serverError(w, err)
	}
}

func (app *application) putWatchedlist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		app.clientError(w, http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		ID        uint `json:"id"`
		Watchlist bool `json:"watchedlist"`
	}

	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	if body.ID > 9999 {
		app.clientError(w, http.StatusUnprocessableEntity)
		return
	}

	userID := app.sessionManager.GetInt(r.Context(), "userID")
	if userID == 0 {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	var user models.User
	result := app.DB.First(&user, userID)
	if result.Error != nil {
		app.serverError(w, result.Error)
		return
	}

	if body.Watchlist {
		app.DB.Model(&user).Association("WatchedList").Delete(&models.Film{ID: body.ID})
	} else {
		app.DB.Model(&user).Association("WatchedList").Append(&models.Film{ID: body.ID})
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(map[string]bool{"watchedlist": body.Watchlist}); err != nil {
		app.serverError(w, err)
	}
}

func (app *application) getWatchlist(w http.ResponseWriter, r *http.Request) {
	userID := app.sessionManager.GetInt(r.Context(), "userID")
	if userID == 0 {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	var user models.User
	result := app.DB.Preload("WatchList").First(&user, userID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			app.notFound(w)
		} else {
			app.serverError(w, result.Error)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(user.WatchList); err != nil {
		app.serverError(w, err)
	}
}

func (app *application) getWatchedlist(w http.ResponseWriter, r *http.Request) {
	userID := app.sessionManager.GetInt(r.Context(), "userID")
	if userID == 0 {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	var user models.User
	result := app.DB.Preload("WatchedList").First(&user, userID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			app.notFound(w)
		} else {
			app.serverError(w, result.Error)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(user.WatchedList); err != nil {
		app.serverError(w, err)
	}
}
