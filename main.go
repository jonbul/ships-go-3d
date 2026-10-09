package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"ships3d/config"
	"ships3d/controllers"
	dataaccess "ships3d/dataAccess"
)

func main() {
	// .env is optional: in a container the variables come from the
	// environment itself.
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file loaded:", err)
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	store, err := dataaccess.Connect(cfg.MongoURI, cfg.MongoDatabase)
	if err != nil {
		log.Fatal("could not connect to MongoDB: ", err)
	}
	defer func() { _ = store.Close() }()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server := controllers.NewServer(cfg, store)
	go server.Hub().Run(ctx)

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           server.Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		var err error
		if cfg.TLS() {
			log.Printf("ships-go-3d listening on https://:%s", cfg.Port)
			err = httpServer.ListenAndServeTLS(cfg.CertPath, cfg.KeyPath)
		} else {
			log.Printf("ships-go-3d listening on http://:%s (no SSL_CERT_PATH/SSL_KEY_PATH)", cfg.Port)
			err = httpServer.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
}
