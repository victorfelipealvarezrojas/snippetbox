package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/justinas/nosurf"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Crea una función diferida (que siempre se ejecutará en caso de
		// un pánico cuando Go desenrolle la pila).
		defer func() {
			// Utilice la función de recuperación integrada para comprobar si se ha producido un
			// pánico o no. Si se ha producido...
			if err := recover(); err != nil {
				// Establece un encabezado "Connection: close" en la respuesta.
				w.Header().Set("Connection", "close")
				// Llama al método auxiliar app.serverError para devolver un error 500.
				// Respuesta interna del servidor.
				app.serverError(w, r, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (app *application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			ip     = r.RemoteAddr
			proto  = r.Proto
			method = r.Method
			uri    = r.URL.RequestURI()
		)
		app.logger.Info("received request", "ip", ip, "proto", proto, "method", method, "uri", uri)
		next.ServeHTTP(w, r)
	})
}

func commonHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Content-Security-Policy: restringe el origen de los recursos que el
		// navegador puede cargar; principal defensa contra XSS e inyección.
		//   default-src 'self'                  -> por defecto, solo mismo origen
		//   style-src 'self' fonts.googleapis.com -> CSS propio + Google Fonts
		//   font-src fonts.gstatic.com          -> fuentes desde el CDN de Google
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' fonts.googleapis.com; font-src fonts.gstatic.com")

		// Referrer-Policy: controla cuánto del header Referer se filtra al navegar.
		// origin-when-cross-origin -> cross-origin envía solo el origen (sin ruta
		// ni query); same-origin envía la URL completa.
		w.Header().Set("Referrer-Policy", "origin-when-cross-origin")

		// X-Content-Type-Options: nosniff -> obliga al navegador a respetar el
		// Content-Type declarado en vez de adivinarlo (MIME sniffing). Evita que
		// un recurso sea reinterpretado como otro tipo ejecutable.
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// X-Frame-Options: deny -> prohíbe embeber la página en iframe/frame/
		// embed/object. Defensa contra clickjacking. deny es total: ni el mismo
		// origen puede embeberla.
		w.Header().Set("X-Frame-Options", "deny")

		// X-XSS-Protection: 0 -> desactiva el filtro XSS heurístico legacy del
		// navegador. Está deprecado y su heurística introducía vulnerabilidades
		// propias; la protección real la da la CSP. Se apaga explícitamente.
		w.Header().Set("X-XSS-Protection", "0")

		// Server: Go -> normaliza el header Server. net/http no lo emite en
		// respuestas normales pero sí en algunos errores internos; fijarlo unifica
		// lo expuesto. Cosmético en seguridad (igual revela el runtime).
		w.Header().Set("Server", "Go")

		// Cede el control al siguiente handler de la cadena.
		next.ServeHTTP(w, r)
	})
}

func (app *application) requireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !app.isAuthenticated(r) {
			http.Redirect(w, r, "/user/login", http.StatusSeeOther)
			return
		}

		// configure el encabezado "Cache-Control: no-store" para que las páginas
		// que requieren autenticación no se almacenen en la caché del navegador del usuario (o
		// en otra caché intermedia).
		w.Header().Add("Cache-Control", "no-store")
		// And call the next handler in the chain.
		next.ServeHTTP(w, r)
	})
}

// noSurf es un middleware que protege contra ataques CSRF (Cross-Site Request
// Forgery) usando el paquete justinas/nosurf. Se coloca en la cadena de
// middleware dinámico para que todas las rutas que procesan formularios
// queden cubiertas.
func noSurf(next http.Handler) http.Handler {
	// nosurf.New envuelve el siguiente handler y devuelve un *CSRFHandler
	// (que a su vez implementa http.Handler). En cada petición no-segura
	// (POST, PUT, PATCH, DELETE) verifica que el token CSRF del formulario
	// coincida con el de la cookie antes de llamar a next; si no coinciden,
	// responde 400 y no ejecuta el handler. Las peticiones seguras
	// (GET, HEAD, OPTIONS, TRACE) pasan sin verificación.
	csrfHandler := nosurf.New(next)

	// SetBaseCookie configura los atributos de la cookie donde nosurf guarda
	// su token CSRF (cookie "csrf_token"), no la cookie de sesión.
	csrfHandler.SetBaseCookie(http.Cookie{
		HttpOnly: true, // impide que JavaScript lea la cookie (mitiga robo de token vía XSS)
		Path:     "/",  // la cookie aplica a todas las rutas del sitio
		Secure:   true, // la cookie solo se envía sobre HTTPS
	})

	return csrfHandler
}

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Recupera el valor de authenticatedUserID de la sesión usando el
		// método GetInt(). Este devolverá el valor cero para un entero (0) si no
		// hay ningún valor de "authenticatedUserID" en la sesión; en ese caso,
		// llamamos al siguiente controlador en la cadena como de costumbre y regresamos.
		id := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")
		if id == 0 {
			next.ServeHTTP(w, r)
			return
		}

		// De lo contrario, comprobamos si existe un usuario con ese ID en nuestra
		// base de datos.
		exists, err := app.users.Exists(id)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		// Si se encuentra un usuario coincidente, sabemos que la solicitud
		// proviene de un usuario autenticado que existe en nuestra base de datos.
		// Creamos una nueva copia de la solicitud (con un valor isAuthenticatedContextKey
		// de verdadero en el contexto de la solicitud) y la asignamos a r.
		if exists {
			ctx := context.WithValue(r.Context(), isAuthenticatedContextKey, true)
			r = r.WithContext(ctx)
		}
		// Call the next handler in the chain.
		next.ServeHTTP(w, r)
	})
}
