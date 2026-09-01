package main

import (
	"html/template"
	"path/filepath"
	"time"

	"github.com/valvarez/snippetbox/internal/models"
)

type templateData struct {
	CurrentYear int
	Snippet     models.Snippet
	Snippets    []models.Snippet
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
	pages, err := filepath.Glob("./ui/html/pages/*.tmpl.html")
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)

		// El objeto template.FuncMap debe registrarse en el conjunto de plantillas antes de llamar al método ParseFiles().
		// Esto significa que debemos usar template.New() para crear un conjunto de plantillas vacío,
		// usar el método Funcs() para registrar el objeto template.FuncMap y, a continuación, analizar el archivo de forma habitual.
		ts, err := template.New(name).Funcs(functions).ParseFiles("./ui/html/base.tmpl.html")
		if err != nil {
			return nil, err
		}
		// Call ParseGlob() *on this template set* to add any partials.
		ts, err = ts.ParseGlob("./ui/html/partials/*.tmpl.html")
		if err != nil {
			return nil, err
		}
		// Call ParseFiles() *on this template set* to add the page template.
		ts, err = ts.ParseFiles(page)
		if err != nil {
			return nil, err
		}

		cache[name] = ts
	}
	return cache, nil
}
