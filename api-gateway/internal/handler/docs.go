package handler

import (
	"io/fs"
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	"github.com/labstack/echo/v4"
	"github.com/ride-sharing/api-gateway/docs"
)

// serviceDocs defines a service's documentation metadata.
type serviceDocs struct {
	Name    string
	Title   string
	SpecURL string
}

var services = []serviceDocs{
	{Name: "user-service", Title: "User Service API", SpecURL: "/docs/specs/user-service.yaml"},
	{Name: "match-service", Title: "Match Service API", SpecURL: "/docs/specs/match-service.yaml"},
	{Name: "chat-service", Title: "Chat Service API", SpecURL: "/docs/specs/chat-service.yaml"},
	{Name: "location-service", Title: "Location Service API", SpecURL: "/docs/specs/location-service.yaml"},
}

// RegisterDocsRoutes sets up Swagger UI documentation routes.
func RegisterDocsRoutes(e *echo.Echo) {
	// Serve spec YAML files as static assets
	specsFS, err := fs.Sub(docs.SpecFS, "specs")
	if err != nil {
		panic(err)
	}
	e.GET("/docs/specs/*", func(c echo.Context) error {
		name := c.Param("*")
		data, err := fs.ReadFile(specsFS, name)
		if err != nil {
			return c.String(http.StatusNotFound, "spec not found")
		}
		c.Response().Header().Set(echo.HeaderContentType, "application/yaml")
		return c.Blob(http.StatusOK, "application/yaml", data)
	})

	// Serve Swagger UI for each service via go-swagger middleware
	for _, svc := range services {
		opts := middleware.SwaggerUIOpts{
			BasePath: "/",
			Path:     svc.Name,
			SpecURL:  svc.SpecURL,
			Title:    svc.Title,
		}
		uiHandler := middleware.SwaggerUI(opts, nil)
		e.GET("/"+svc.Name+"/*", echo.WrapHandler(uiHandler))
	}

	// Landing page listing all services
	e.GET("/docs", docsLandingPage)
	e.GET("/docs/", docsLandingPage)
}

// docsLandingPage serves a simple HTML page linking to each service's Swagger UI.
func docsLandingPage(c echo.Context) error {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>API Documentation</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0f172a; color: #e2e8f0; min-height: 100vh; display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 2rem; }
    h1 { font-size: 2rem; margin-bottom: 0.5rem; color: #f8fafc; }
    p { color: #94a3b8; margin-bottom: 2rem; }
    .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 1.5rem; max-width: 900px; width: 100%; }
    .card { background: #1e293b; border: 1px solid #334155; border-radius: 12px; padding: 1.5rem; text-decoration: none; color: inherit; transition: border-color 0.2s, transform 0.2s; }
    .card:hover { border-color: #3b82f6; transform: translateY(-2px); }
    .card h2 { font-size: 1.1rem; color: #f8fafc; margin-bottom: 0.25rem; }
    .card span { font-size: 0.85rem; color: #64748b; }
  </style>
</head>
<body>
  <h1>🚀 API Documentation</h1>
  <p>Interactive Swagger UI for all microservices</p>
  <div class="grid">
    <a class="card" href="/user-service/"><h2>User Service</h2><span>Auth, profiles, CRUD — port 8081</span></a>
    <a class="card" href="/match-service/"><h2>Match Service</h2><span>Swipes, matches, discover — port 8082</span></a>
    <a class="card" href="/chat-service/"><h2>Chat Service</h2><span>Conversations, messages — port 8083</span></a>
    <a class="card" href="/location-service/"><h2>Location Service</h2><span>Location, nearby, presence — port 8084</span></a>
  </div>
</body>
</html>`
	return c.HTML(http.StatusOK, html)
}
