package main

import (
	"net/http"
	// "github.com/gorilla/csrf"
	"movies4u.net/ui"
)

func (app *application) routes() http.Handler {

	// key := app.generateRandomKey()
	// csrf := csrf.Protect([]byte(key), csrf.Secure(false), csrf.SameSite(csrf.SameSiteLaxMode))

	router := http.NewServeMux()

	// Static file server
	fileServer := http.FileServer(http.FS(ui.Files))
	router.Handle("GET /static/", fileServer)

	// Unprotected routes
	unprotectedRoutes := map[string]http.HandlerFunc{
		"GET /user/login":   app.userLogin,
		"POST /user/login":  app.userLoginPost,
		"GET /user/signin":  app.userSignin,
		"POST /user/signin": app.userSigninPost,
	}

	// Protected routes
	protectedRoutes := map[string]http.HandlerFunc{
		"/":                 app.home,
		"GET /films/{id}":   app.getFilm,
		"POST /user/logout": app.userLogoutPost,
		"POST /film/search": app.searchPost,
		"GET /film/search":  app.search,
		"GET /films":        app.getFilms,
		"PUT /watchlist":    app.putWatchlist,
		"PUT /watchedlist":  app.putWatchedlist,
		"GET /watchlist":    app.getWatchlist,
		"GET /watchedlist":  app.getWatchedlist,
	}
	// Register unprotected routes
	for pattern, handler := range unprotectedRoutes {
		// router.Handle(pattern, csrf(http.HandlerFunc(handler)))
		router.Handle(pattern, http.HandlerFunc(handler))
	}

	// Register protected routes with authentication
	for pattern, handler := range protectedRoutes {
		// router.Handle(pattern, csrf(app.requireAuthentication(http.HandlerFunc(handler))))
		router.Handle(pattern, app.requireAuthentication(http.HandlerFunc(handler)))
	}

	// Method Not Allowed handlers
	methodNotAllowedRoutes := map[string]string{
		"/film/view/{id}": http.MethodGet,
		"/film/create":    http.MethodPost,
		"/user/login":     http.MethodGet + " " + http.MethodPost,
		"/user/signin":    http.MethodGet + " " + http.MethodPost,
		"/user/logout":    http.MethodPost,
	}

	for pattern, methods := range methodNotAllowedRoutes {
		router.HandleFunc(pattern, app.methodNotAllowed(methods))
	}

	return app.chainMiddleware(router, app.sessionManager.LoadAndSave, app.recoverPanic, app.logRequest, secureHeaders, app.authenticate)
}
