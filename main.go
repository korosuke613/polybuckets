package main

import (
	"context"
	"embed"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/korosuke613/polybuckets/internal"
	"github.com/korosuke613/polybuckets/internal/env"
	"github.com/korosuke613/polybuckets/internal/s3client"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

//go:embed templates/*.html
var templates embed.FS

// main is the entry point of the application. It sets up the server and routes.
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up the default logger
	slog.SetDefault(internal.NewJsonLogger())

	e := echo.New()
	// Middleware setup
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `{"time":"${time_rfc3339_nano}","level":"INFO","msg":"access log","value":` +
			`{"remote_ip":"${remote_ip}",` +
			`"host":"${host}","method":"${method}","uri":"${uri}","user_agent":"${user_agent}",` +
			`"status":${status},"error":"${error}","latency":${latency},"latency_human":"${latency_human}"` +
			`,"bytes_in":${bytes_in},"bytes_out":${bytes_out}}}` + "\n",
	}))
	e.Use(middleware.Recover())
	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Filesystem: http.FS(templates),
	}))
	e.HideBanner = true
	e.HidePort = true

	// Template renderer setup
	e.Renderer = &TemplateRenderer{
		templates: template.Must(template.ParseFS(templates, "templates/*.html")),
	}

	// Initialize S3 client
	client, err := s3client.NewClient(ctx)
	if err != nil {
		e.Logger.Fatal("Failed to initialize S3 client: ", err)
	}

	// Serve static files (favicon.ico)
	e.Static("/static", "static")

	// Route handlers
	// Favicon handler
	e.GET("/favicon.ico", func(c echo.Context) error {
		return c.NoContent(http.StatusNotFound)
	})

	// Route for file download
	e.GET("/download/:bucket/*", func(c echo.Context) error {
		bucket := c.Param("bucket")
		key := c.Param("*")

		// Unescape the key
		key, err := url.QueryUnescape(key)
		if err != nil {
			return c.Render(http.StatusInternalServerError, "error.html", map[string]interface{}{
				"Error": err.Error(),
			})
		}

		// Get the object from S3
		result, err := client.GetObject(c.Request().Context(), bucket, key)
		if err != nil {
			return c.Render(http.StatusInternalServerError, "error.html", map[string]interface{}{
				"Error": err.Error(),
			})
		}
		defer result.Body.Close()

		return c.Stream(http.StatusOK, "application/octet-stream", result.Body)
	})

	// Catch-all route handler
	e.GET("/*", func(c echo.Context) error {
		path := c.Request().URL.Path
		return handleRequest(c.Request().Context(), c, client, path)
	})

	// Load configuration
	pbConfig := env.LoadPBConfig()
	slog.Info("loaded config", "config", pbConfig)
	port := pbConfig.Port
	if port == "" {
		port = "1323"
	}
	ip := pbConfig.IPAddress
	if ip == "" {
		ip = "0.0.0.0"
	}
	slog.Info("starting server", "ip", ip, "port", port)
	e.Logger.Fatal(e.Start(ip + ":" + port))
}

// handleRequest handles incoming HTTP requests and routes them to the appropriate S3 operations.
func handleRequest(ctx context.Context, c echo.Context, client *s3client.Client, path string) error {
	switch {
	case path == "/":
		// List all buckets
		buckets, err := client.ListBuckets(ctx)
		if err != nil {
			return c.Render(http.StatusInternalServerError, "error.html", map[string]interface{}{
				"Error": err.Error(),
				"Path":  "/",
			})
		}
		return c.Render(http.StatusOK, "buckets.html", buckets)

	default:
		// List objects in a bucket
		bucket, parentPrefix, prefix := internal.ParsePath(path)
		objects, err := client.ListObjects(ctx, bucket, prefix)
		if err != nil {
			return c.Render(http.StatusInternalServerError, "error.html", map[string]interface{}{
				"Error":        err.Error(),
				"Bucket":       bucket,
				"ParentPrefix": parentPrefix,
				"Prefix":       prefix,
			})
		}

		return c.Render(http.StatusOK, "objects.html", map[string]interface{}{
			"Bucket":       bucket,
			"ParentPrefix": parentPrefix,
			"Prefix":       prefix,
			"Objects":      objects,
		})
	}
}

// TemplateRenderer implements echo.Renderer interface for rendering HTML templates.
type TemplateRenderer struct {
	templates *template.Template
}

// Render renders the specified template with the provided data.
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

// Helper functions for path processing...
