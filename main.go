package main

import (
	"context"
	"embed"
	"log/slog"

	"github.com/korosuke613/polybuckets/internal"
	"github.com/korosuke613/polybuckets/internal/env"
	"github.com/korosuke613/polybuckets/internal/server"
)

//go:embed templates/*.html
var templates embed.FS

// main is the entry point of the application. It sets up the server and routes.
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up the default logger
	slog.SetDefault(internal.NewJsonLogger())

	// Initialize Echo server
	e := server.NewEchoServer(templates)

	// Set up routes and middleware
	server.SetupMiddleware(e, templates)
	server.SetupRoutes(e, ctx)

	// Load configuration and start server
	server.StartServer(e, env.PBConfig)
}
