package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/v2"
	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"movies4u.net/internals/dataloader"
	"movies4u.net/internals/models"
)

type application struct {
	errorLog       *log.Logger
	infoLog        *log.Logger
	DB             *gorm.DB
	templateCache  map[string]*template.Template
	sessionManager *scs.SessionManager
}

func main() {
	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	addr := flag.String("addr", ":4000", "Http Server Listening Port")
	dsn := flag.String("dsn", "web:pass@/goapi?parseTime=true", "My sql datasource name")
	jsonFilePath := flag.String("json", "./data/films.json", "Path to the JSON file containing film data")

	flag.Parse()

	db, err := gorm.Open(mysql.Open(*dsn), &gorm.Config{})
	if err != nil {
		errorLog.Fatal(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		errorLog.Fatal(err)
	}
	defer sqlDB.Close()

	templateCache, err := newTemplateCache()
	if err != nil {
		errorLog.Fatal(err)
	}

	sessionManager := scs.New()
	sessionManager.Store = mysqlstore.New(sqlDB)
	sessionManager.Lifetime = 12 * time.Hour
	sessionManager.Cookie.Secure = true // Set to false if not using HTTPS

	app := application{
		errorLog:       errorLog,
		infoLog:        infoLog,
		DB:             db,
		templateCache:  templateCache,
		sessionManager: sessionManager,
	}

	// Ensure tables are created before checking their contents
	err = db.AutoMigrate(&models.User{}, &models.Genre{}, &models.Star{}, &models.Director{}, &models.Film{})
	if err != nil {
		errorLog.Fatal(err)
	}

	
	
	dataLoader := dataloader.DataLoader{DB: db}
	err = dataLoader.LoadFilmsFromFile(*jsonFilePath)
	if err != nil {
		if errors.Is(err, dataloader.ErrDataLoaded){
			infoLog.Println("Loaded Database")
		}else{
			errorLog.Fatal(err)
		}
		
	}
	
	

	tlsConfig := &tls.Config{
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
		MinVersion:       tls.VersionTLS12,
		MaxVersion:       tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}

	infoLog.Printf("Starting Server %s", *addr)

	srv := &http.Server{
		Addr:         *addr,
		ErrorLog:     errorLog,
		Handler:      app.routes(),
		TLSConfig:    tlsConfig,
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	err = srv.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")
	errorLog.Fatal(err)
}
