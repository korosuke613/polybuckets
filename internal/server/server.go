package server

import (
	"bytes"
	"context"
	"embed"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/korosuke613/polybuckets/internal"
	"github.com/korosuke613/polybuckets/internal/env"
	"github.com/korosuke613/polybuckets/internal/s3client"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// TemplateRenderer implements echo.Renderer interface for rendering HTML templates.
type TemplateRenderer struct {
	templates *template.Template
}

// Render renders the specified template with the provided data.
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

// NewEchoServer creates a new Echo server instance.
func NewEchoServer(templates embed.FS) *echo.Echo {
	e := echo.New()
	e.Renderer = &TemplateRenderer{
		templates: template.Must(template.ParseFS(templates, "templates/*.html", "templates/partials/*.html")),
	}
	return e
}

// SetupMiddleware sets up the middleware for the Echo instance.
func SetupMiddleware(e *echo.Echo, templates embed.FS) {
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `{"time":"${time_rfc3339_nano}","level":"INFO","msg":"access log","value":` +
			`{"remote_ip":"${remote_ip}",` +
			`"host":"${host}","method":"${method}","uri":"${uri}","user_agent":"${user_agent}",` +
			`"status":${status},"error":"${error}","latency":${latency},"latency_human":"${latency_human}"` +
			`,"bytes_in":${bytes_in},"bytes_out":${bytes_out}${custom}}}` + "\n",
		CustomTagFunc: func(c echo.Context, buf *bytes.Buffer) (int, error) {
			writeString := ""

			// if ListObject cache hit, output to log
			hitCache := c.Get("hitCache")
			if hitCache != nil {
				writeString += `,"hit_cache":` + strconv.FormatBool(hitCache.(bool))

				cacheExpire := c.Get("cacheExpire")
				if hitCache == true && cacheExpire != nil {
					writeString += `,"cache_expire":"` + cacheExpire.(string) + `"`
					c.Set("cacheExpire", nil)
				}
			}

			return buf.WriteString(writeString)
		},
	}))
	e.Use(middleware.Recover())
	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Filesystem: http.FS(templates),
	}))
	e.HideBanner = true
	e.HidePort = true
}

// SetupRoutes sets up the routes for the Echo instance.
func SetupRoutes(e *echo.Echo, ctx context.Context) {
	// Initialize S3 client
	siteName := env.PBConfig.SiteName
	client, err := s3client.NewClient(ctx)
	client.CacheDuration = env.PBConfig.CacheDuration
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
				"SiteName": siteName,
				"Error":    err.Error(),
			})
		}

		// Get the object from S3
		result, err := client.GetObject(c.Request().Context(), bucket, key)
		if err != nil {
			return c.Render(http.StatusInternalServerError, "error.html", map[string]interface{}{
				"SiteName": siteName,
				"Error":    err.Error(),
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
}

// handleRequest handles incoming HTTP requests and routes them to the appropriate S3 operations.
func handleRequest(ctx context.Context, c echo.Context, client *s3client.Client, path string) error {
	siteName := env.PBConfig.SiteName
	switch {
	case path == "/":
		// List all buckets

		// ListBuckets を継承
		type BucketsInfo struct {
			Buckets  []s3client.BucketInfo
			SiteName string
		}

		buckets, err := client.ListBuckets(ctx)
		bucketsInfo := BucketsInfo{
			Buckets:  buckets,
			SiteName: siteName,
		}
		if err != nil {
			return c.Render(http.StatusInternalServerError, "error.html", map[string]interface{}{
				"SiteName": siteName,
				"Error":    err.Error(),
				"Path":     "/",
			})
		}
		return c.Render(http.StatusOK, "buckets.html", bucketsInfo)

	default:
		// List objects in a bucket
		bucket, parentPrefix, prefix := internal.ParsePath(path)
		// if the query parameter `refresh` is set to `true`, clear the cache
		if c.QueryParam("refresh") == "true" {
			client.ClearListObjectsCache(ctx, bucket, prefix)
		}

		objects, hitCache, err := client.ListObjects(ctx, bucket, prefix)

		c.Set("hitCache", hitCache)
		var cacheExpire time.Time
		if hitCache {
			cacheEntry := client.GetListObjectsCacheEntry(ctx, bucket, prefix)
			if cacheEntry != nil {
				cacheExpire = cacheEntry.Expiry
				c.Set("cacheExpire", cacheExpire.Format(time.RFC3339))
			}
		}

		// Clear old cache entries
		go client.ClearOldListObjectsCache(ctx)

		if err != nil {
			return c.Render(http.StatusInternalServerError, "error.html", map[string]interface{}{
				"SiteName":     siteName,
				"Error":        err.Error(),
				"Bucket":       bucket,
				"ParentPrefix": parentPrefix,
				"Prefix":       prefix,
			})
		}

		return c.Render(http.StatusOK, "objects.html", map[string]interface{}{
			"SiteName":     siteName,
			"Bucket":       bucket,
			"ParentPrefix": parentPrefix,
			"Prefix":       prefix,
			"Objects":      objects,
			"HitCache":     hitCache,
			"LastCached":   cacheExpire.Add(-client.CacheDuration).UTC(),
		})
	}
}

// StartServer starts the Echo server with the provided configuration.
func StartServer(e *echo.Echo, pbConfig *env.PBConfigType) {
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
