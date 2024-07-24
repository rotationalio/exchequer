package exchequer

import (
	"io/fs"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rotationalio/exchequer/pkg/logger"
	"github.com/rotationalio/exchequer/pkg/metrics"
)

// Sets up the server's middleware and routes.
func (s *Server) setupRoutes() (err error) {
	// Setup HTML template renderer
	templateFiles, _ := fs.Sub(content, "templates")
	includes := []string{"layouts/*.html", "components/*.html"}
	if s.router.HTMLRender, err = NewRender(templateFiles, "*.html", includes...); err != nil {
		return err
	}

	// NOTE: partials can't have the same names as top-level pages
	s.router.HTMLRender.(*Render).AddPattern(templateFiles, "partials/*/*.html")

	// Create CORS configuration
	corsConf := cors.Config{
		AllowMethods:     []string{"GET", "HEAD"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-CSRF-TOKEN"},
		AllowOrigins:     []string{s.conf.Origin},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	// Application Middleware
	// NOTE: ordering is important to how middleware is handled
	middlewares := []gin.HandlerFunc{
		// Logging should be on the outside so we can record the correct latency of requests
		// NOTE: logging panics will not recover
		logger.GinLogger("web"),

		// Panic recovery middleware
		gin.Recovery(),

		// CORS configuration allows the front-end to make cross-origin requests
		cors.New(corsConf),

		// Maintenance mode middleware to return unavailable
		s.Maintenance(),
	}

	// Kubernetes liveness probes added before middleware.
	s.router.GET("/healthz", s.Healthz)
	s.router.GET("/livez", s.Healthz)
	s.router.GET("/readyz", s.Readyz)

	// Prometheus metrics handler added before middleware.
	// Note metrics will be served at /metrics
	metrics.Routes(s.router)

	// Add the middleware to the router
	for _, middleware := range middlewares {
		if middleware != nil {
			s.router.Use(middleware)
		}
	}

	// NotFound and NotAllowed routes
	s.router.NoRoute(s.NotFound)
	s.router.NoMethod(s.NotAllowed)

	// Static Files
	staticFiles, _ := fs.Sub(content, "static")
	s.router.StaticFS("/static", http.FS(staticFiles))

	// Pages
	s.router.GET("/", s.Index)

	// API Routes (Including Content Negotiated Partials)
	v1 := s.router.Group("/v1")
	{
		// Status/Heartbeat endpoint
		v1.GET("/status", s.Status)

		// Adyen JSON webhooks and integration
		adyen := v1.Group("/adyen", s.AdyenWebhookAuth())
		{
			adyen.GET("/payments", s.AdyenPaymentsWebhook)
		}
	}

	return nil
}
