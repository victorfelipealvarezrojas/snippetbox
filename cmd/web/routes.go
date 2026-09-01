package main

import (
	"net/http"

	"github.com/justinas/alice"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("GET /static/", http.StripPrefix("/static", fileServer))

	// Crea una nueva cadena de middleware que contenga el middleware específico para nuestras
	// rutas de aplicación dinámicas. Por ahora, esta cadena solo contendrá el
	// middleware de sesión LoadAndSave.
	dynamic := alice.New(app.sessionManager.LoadAndSave)

	// HandleFunc espera algo que tenga la firma func(w http.ResponseWriter, r *http.Request). Una función suelta.
	// Handle espera algo que implemente la interfaz http.Handler, o sea que tenga el método ServeHTTP(w, r). Un valor de un tipo con ese método.
	mux.Handle("GET /{$}", dynamic.ThenFunc(app.home))
	mux.Handle("GET /snippet/view/{id}", dynamic.ThenFunc(app.snippetView))
	mux.Handle("GET /snippet/create", dynamic.ThenFunc(app.snippetCreate))
	mux.Handle("POST /snippet/create", dynamic.ThenFunc(app.snippetCreatePost))

	standard := alice.New(app.recoverPanic, app.logRequest, commonHeaders)

	return standard.Then(mux)
}
