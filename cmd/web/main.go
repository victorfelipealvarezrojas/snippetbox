package main

import (
	"crypto/tls"
	"database/sql"
	"flag"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"time"

	// no usa directamente nada del paquete mysql. habla con la base a través de database/sql, no del driver.
	//Pero el driver sí tiene que cargarse, porque su función init() es la que lo registra en database/sql.
	// Sin esa registración, sql.Open("mysql", ...) no sabría qué driver usar.
	// New import

	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/v2"

	"github.com/go-playground/form/v4" //form.decoder

	_ "github.com/go-sql-driver/mysql"
	"github.com/valvarez/snippetbox/internal/models"
)

type config struct {
	addr      string
	staticDir string
}

type application struct {
	logger         *slog.Logger
	snippets       *models.SnippetModel
	users          *models.UserModel
	templateCache  map[string]*template.Template
	formDecoder    *form.Decoder
	sessionManager *scs.SessionManager
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

	// Usamos la función scs.New() para inicializar un nuevo gestor de sesiones. Luego,
	// lo configuramos para que use nuestra base de datos MySQL como almacén de sesiones y establecemos una
	// duración de 12 horas (para que las sesiones caduquen automáticamente 12 horas
	// después de su creación).
	sessionManager := scs.New()
	sessionManager.Store = mysqlstore.New(db)
	sessionManager.Lifetime = 12 * time.Hour

	formDecoder := form.NewDecoder()

	app := &application{
		logger:         logger,
		snippets:       &models.SnippetModel{DB: db},
		users:          &models.UserModel{DB: db},
		templateCache:  templateCache,
		formDecoder:    formDecoder,
		sessionManager: sessionManager,
	}

	// Inicializa una estructura tls.Config para almacenar la configuración TLS no predeterminada que
	// queremos que use el servidor. En este caso, lo único que estamos cambiando
	// es el valor de las preferencias de curva, para que solo se utilicen curvas elípticas con
	// implementaciones de ensamblador.
	tlsConfig := &tls.Config{
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
	}

	srv := &http.Server{
		Addr:           cfg.addr,
		Handler:        app.routes(),
		ErrorLog:       slog.NewLogLogger(logger.Handler(), slog.LevelError),
		MaxHeaderBytes: 524288, // controlar el número máximo de bytes que el servidor leerá al analizar los encabezados de la solicitud
		TLSConfig:      tlsConfig,
		IdleTimeout:    time.Minute,      //  IdleTimeout en 1 minuto, lo que significa que todas las conexiones persistentes se cerrarán automáticamente después de 1 minuto de inactividad.
		ReadTimeout:    5 * time.Second,  // ReadTimeout a 5 segundos. Esto significa que si los encabezados o el cuerpo de la solicitud aún se están leyendo 5 segundos después de que se haya aceptado la solicitud por primera vez,cerrará la conexión subyacente.
		WriteTimeout:   10 * time.Second, // WriteTimeout cerrará la conexión subyacente si nuestro servidor intenta escribir en ella después de un período determinado (en nuestro código, 10 segundos)
	}

	logger.Info("starting server", "addr", srv.Addr)

	err = srv.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")
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
