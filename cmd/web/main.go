package main

import (
	"database/sql"
	"flag"
	"html/template"
	"log/slog"
	"net/http"
	"os"

	// no usa directamente nada del paquete mysql. habla con la base a través de database/sql, no del driver.
	//Pero el driver sí tiene que cargarse, porque su función init() es la que lo registra en database/sql.
	// Sin esa registración, sql.Open("mysql", ...) no sabría qué driver usar.
	_ "github.com/go-sql-driver/mysql"
	"github.com/valvarez/snippetbox/internal/models"
)

type config struct {
	addr      string
	staticDir string
}

type application struct {
	logger        *slog.Logger
	snippets      *models.SnippetModel
	templateCache map[string]*template.Template
}

func main() {
	var cfg config

	flag.StringVar(&cfg.addr, "addr", ":4000", "HTTP network address")
	flag.StringVar(&cfg.staticDir, "static-dir", "./ui/static", "Path to static assets")

	dsn := flag.String("dsn", "web:pass@/snippetbox?parseTime=true", "MySQL data source name")

	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	db, err := openDB(*dsn)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	defer db.Close()

	templateCache, err := newTemplateCache()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	app := &application{
		logger:        logger,
		snippets:      &models.SnippetModel{DB: db},
		templateCache: templateCache,
	}

	logger.Info("starting server", "addr", cfg.addr)

	// http.Handler es una interface: type Handler interface { ServeHTTP(ResponseWriter, *Request) }
	// mux la satisface porque tiene ese método(ServeHTTP); es lo que ListenAndServe espera como 2º argumento
	err = http.ListenAndServe(cfg.addr, app.routes())

	logger.Error(err.Error())
	os.Exit(1)

}

// Lsql.Open() no crea ninguna conexión, todo lo que hace es inicializar
// para uso futuro. Las conexiones reales a la base de datos se establecen de forma perezosa, a medida que
// cuando sea necesario por primera vez. para verificar que todo esté configurado correctamente se usa el método db.Ping()
func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err

	}
	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
