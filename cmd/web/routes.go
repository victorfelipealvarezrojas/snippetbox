package main

import (
	"net/http"

	"github.com/justinas/alice"
	"github.com/valvarez/snippetbox/ui"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	// fileServer := http.FileServer(http.Dir("./ui/static/"))
	// mux.Handle("GET /static/", http.StripPrefix("/static", fileServer))

	// Usa la función http.FileServerFS() para crear un manejador HTTP que
	// sirva los archivos incrustados en ui.Files. Es importante tener en cuenta que nuestros
	// archivos estáticos se encuentran en la carpeta "static" del sistema de archivos incrustado ui.Files.
	// Por ejemplo, nuestra hoja de estilos CSS se encuentra en
	// "static/css/main.css". Esto significa que ya no necesitamos eliminar el
	// prefijo de la URL de la solicitud; cualquier solicitud que comience con /static/ puede
	// pasarse directamente al servidor de archivos y se servirá el archivo estático correspondiente
	// (siempre que exista).
	mux.Handle("GET /static/", http.FileServerFS(ui.Files))

	// Crea una nueva cadena de middleware que contenga el middleware específico para nuestras
	// rutas de aplicación dinámicas. Por ahora, esta cadena solo contendrá el
	// middleware de sesión LoadAndSave.
	dynamic := alice.New(app.sessionManager.LoadAndSave, noSurf, app.authenticate) // este midd lo proporciona el paquete justinas/nosurf

	// HandleFunc espera algo que tenga la firma func(w http.ResponseWriter, r *http.Request). Una función suelta.
	// Handle espera algo que implemente la interfaz http.Handler, o sea que tenga el método ServeHTTP(w, r). Un valor de un tipo con ese método.
	mux.Handle("GET /{$}", dynamic.ThenFunc(app.home))
	mux.Handle("GET /snippet/view/{id}", dynamic.ThenFunc(app.snippetView))
	mux.Handle("GET /user/signup", dynamic.ThenFunc(app.userSignup))
	mux.Handle("POST /user/signup", dynamic.ThenFunc(app.userSignupPost))
	mux.Handle("GET /user/login", dynamic.ThenFunc(app.userLogin))
	mux.Handle("POST /user/login", dynamic.ThenFunc(app.userLoginPost))

	protected := dynamic.Append(app.requireAuthentication) // este midd lo proporciona el paquete

	mux.Handle("GET /snippet/create", protected.ThenFunc(app.snippetCreate))
	mux.Handle("POST /snippet/create", protected.ThenFunc(app.snippetCreatePost))
	mux.Handle("POST /user/logout", protected.ThenFunc(app.userLogoutPost))

	standard := alice.New(app.recoverPanic, app.logRequest, commonHeaders)

	return standard.Then(mux)
}
