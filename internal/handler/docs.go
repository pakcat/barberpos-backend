package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type DocsHandler struct {
	OpenAPIPath string
}

func (h DocsHandler) RegisterRoutes(r chi.Router) {
	r.Get("/openapi.yaml", h.serveSpec)
	r.Get("/docs", h.serveUI)
}

func (h DocsHandler) serveSpec(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, h.OpenAPIPath)
}

func (h DocsHandler) serveUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	specURL := "/openapi.yaml"
	html := `<!doctype html>
<html>
  <head>
    <title>BarberPOS API Docs</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
      window.onload = () => {
        SwaggerUIBundle({
          url: '` + specURL + `',
          dom_id: '#swagger-ui'
        });
      };
    </script>
  </body>
</html>`
	_, _ = w.Write([]byte(html))
}
