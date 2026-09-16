package main

import (
	"html/template"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/valvarez/snippetbox/internal/models"
	"github.com/valvarez/snippetbox/ui"
)

type templateData struct {
	CurrentYear     int
	Snippet         models.Snippet
	Snippets        []models.Snippet
	Form            any
	Flash           string
	IsAuthenticated bool
	CSRFToken       string
}

// Crea una función humanDate que devuelva una cadena bien formateada
func humanDate(t time.Time) string {
	return t.Format("02 Jan 2006 at 15:04")
}

// Inicializa una plantilla. FuncMap y almacenarlo en una variable global. Esto es
// Esencialmente un string-keyed map que actúa como una búsqueda entre los nombres de nuestros
// funciones personalizadas y las funciones en sí.
var functions = template.FuncMap{
	"humanDate": humanDate,
}

func newTemplateCache() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	// Usamos fs.Glob() para obtener una porción de todas las rutas de archivo en el sistema de archivos integrado ui.Files que coincidan con el patrón 'html/pages/*.tmpl'. Esto, esencialmente,
	// nos da una porción de todas las plantillas de página para la aplicación, igual que antes.
	// pages, err := filepath.Glob("./ui/html/pages/*.tmpl.html") (remplazo)
	pages, err := fs.Glob(ui.Files, "html/pages/*.tmpl.html")

	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)
		// Create a slice containing the filepath patterns for the templates we
		// want to parse.
		patterns := []string{
			"html/base.tmpl.html",
			"html/partials/*.tmpl.html",
			page,
		}

		// Utilice ParseFS() en lugar de ParseFiles() para analizar los archivos de plantilla
		// del sistema de archivos integrado ui.Files.
		ts, err := template.New(name).Funcs(functions).ParseFS(ui.Files, patterns...)
		if err != nil {
			return nil, err
		}
		cache[name] = ts
	}

	return cache, nil
}
