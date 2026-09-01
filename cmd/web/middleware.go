package main

import (
	"fmt"
	"net/http"
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
