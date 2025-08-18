package main

import (
	"context"
	"demo-service/internal/storage"
	"os"
	"os/signal"

	"github.com/rs/zerolog/log"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	dsn := "postgres://demo:demo@localhost:5432/demo"
	storage, err := storage.New(ctx, dsn)

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Data Base")
	}
	defer storage.Close()

	log.Info().Msg("Service started successfully")
	<-ctx.Done()
	log.Info().Msg("Service stopped")
}
