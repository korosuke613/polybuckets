package main

import (
	"context"
	"embed"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/korosuke613/polybuckets/internal"
	"github.com/korosuke613/polybuckets/internal/s3client"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

//go:embed templates/*.html
var templates embed.FS

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Filesystem: http.FS(templates),
	}))

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

	// ファイルダウンロード用のルート
	e.GET("/download/:bucket/*", func(c echo.Context) error {
		bucket := c.Param("bucket")
		key := c.Param("*")

		// key を unescape
		key, err := url.QueryUnescape(key)
		if err != nil {
			return c.Render(http.StatusInternalServerError, "error.html", map[string]interface{}{
				"Error": err.Error(),
			})
		}

		// S3からオブジェクトを取得
		result, err := client.GetObject(c.Request().Context(), bucket, key)
		if err != nil {
			return c.Render(http.StatusInternalServerError, "error.html", map[string]interface{}{
				"Error": err.Error(),
			})
		}
		defer result.Body.Close()

		// ダウンロード用ヘッダーを設定
		c.Response().Header().Set(echo.HeaderContentType, "application/octet-stream")
		c.Response().Header().Set("Content-Disposition", "attachment; filename=\""+key[strings.LastIndex(key, "/")+1:]+"\"")

		// コンテンツをストリーミング
		_, err = io.Copy(c.Response().Writer, result.Body)
		return err
	})

	e.GET("/*", func(c echo.Context) error {
		path := c.Request().URL.Path
		return handleRequest(c.Request().Context(), c, client, path)
	})

	port := os.Getenv("POLIBACKETSU_PORT")
	if port == "" {
		port = "1323"
	}
	ip := os.Getenv("POLIBACKETSU_IP_ADDRESS")
	if ip == "" {
		ip = "localhost"
	}
	e.Logger.Fatal(e.Start(ip + ":" + port))
}

func handleRequest(ctx context.Context, c echo.Context, client *s3client.Client, path string) error {
	switch {
	case path == "/":
		buckets, err := client.ListBuckets(ctx)
		if err != nil {
			return c.Render(http.StatusInternalServerError, "error.html", map[string]interface{}{
				"Error": err.Error(),
				"Path":  "/",
			})
		}
		return c.Render(http.StatusOK, "buckets.html", buckets)

	default:
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

// TemplateRenderer implements echo.Renderer interface
type TemplateRenderer struct {
	templates *template.Template
}

// Render renders template
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

// Helper functions for path processing...
