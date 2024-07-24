package exchequer

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/adyen/adyen-go-api-library/v11/src/adyen"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/rotationalio/exchequer/pkg/config"
	"github.com/rotationalio/exchequer/pkg/logger"
	"github.com/rotationalio/exchequer/pkg/metrics"
)

func init() {
	// Initializes zerolog with our default logging requirements
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.TimestampFieldName = logger.GCPFieldKeyTime
	zerolog.MessageFieldName = logger.GCPFieldKeyMsg

	// Add the severity hook for GCP logging
	var gcpHook logger.SeverityHook
	log.Logger = zerolog.New(os.Stdout).Hook(gcpHook).With().Timestamp().Logger()
}

func New(conf config.Config) (svc *Server, err error) {
	// Load the default configuration from the environment if config is empty.
	if conf.IsZero() {
		if conf, err = config.New(); err != nil {
			return nil, err
		}
	}

	if err = conf.Validate(); err != nil {
		return nil, err
	}

	// Setup our logging config first thing
	zerolog.SetGlobalLevel(conf.GetLogLevel())
	if conf.ConsoleLog {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	// Set the gin mode for all gin servers
	gin.SetMode(conf.Mode)

	// Register the prometheus metrics
	if err = metrics.Setup(); err != nil {
		return nil, err
	}

	svc = &Server{
		conf:  conf,
		errc:  make(chan error, 1),
		adyen: CreateAdyenClient(conf.Adyen),
	}

	// Configure the gin router if enabled
	svc.router = gin.New()
	svc.router.RedirectTrailingSlash = true
	svc.router.RedirectFixedPath = false
	svc.router.HandleMethodNotAllowed = true
	svc.router.ForwardedByClientIP = true
	svc.router.UseRawPath = false
	svc.router.UnescapePathValues = true
	if err = svc.setupRoutes(); err != nil {
		return nil, err
	}

	// Create the http server if enabled
	svc.srv = &http.Server{
		Addr:              svc.conf.BindAddr,
		Handler:           svc.router,
		ErrorLog:          nil,
		ReadHeaderTimeout: 20 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	return svc, nil
}

type Server struct {
	sync.RWMutex
	conf    config.Config
	srv     *http.Server
	router  *gin.Engine
	adyen   *adyen.APIClient
	url     *url.URL
	started time.Time
	healthy bool
	ready   bool
	errc    chan error
}

// Serve the compliance and administrative user interfaces in its own go routine.
func (s *Server) Serve() (err error) {
	// Handle OS Signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		s.errc <- s.Shutdown()
	}()

	// Create a socket to listen on and infer the final URL.
	// NOTE: if the bindaddr is 127.0.0.1:0 for testing, a random port will be assigned,
	// manually creating the listener will allow us to determine which port.
	// When we start listening all incoming requests will be buffered until the server
	// actually starts up in its own go routine below.
	var sock net.Listener
	if sock, err = net.Listen("tcp", s.srv.Addr); err != nil {
		return fmt.Errorf("could not listen on bind addr %s: %s", s.srv.Addr, err)
	}

	s.setURL(sock.Addr())
	s.SetStatus(true, true)
	s.started = time.Now()

	// Listen for HTTP requests and handle them.
	go func(errc chan<- error) {
		// Make sure we don't use the external err to avoid data races.
		if serr := s.serve(sock); !errors.Is(serr, http.ErrServerClosed) {
			errc <- serr
			return
		}
		errc <- nil
	}(s.errc)

	log.Info().Str("url", s.URL()).Msg("exchequer billing service started")
	return <-s.errc
}

// ServeTLS if a tls configuration is provided, otherwise Serve.
func (s *Server) serve(sock net.Listener) error {
	if s.srv.TLSConfig != nil {
		return s.srv.ServeTLS(sock, "", "")
	}
	return s.srv.Serve(sock)
}

// Shutdown the web server gracefully.
func (s *Server) Shutdown() (err error) {
	log.Info().Msg("gracefully shutting down exchequer billing service")
	s.SetStatus(false, false)

	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	s.srv.SetKeepAlivesEnabled(false)
	if serr := s.srv.Shutdown(ctx); serr != nil {
		err = errors.Join(err, serr)
	}

	log.Debug().Err(err).Msg("exchequer billing service has shut down")
	return err
}

// SetStatus sets the health and ready status on the server, modifying the behavior of
// the kubernetes probe responses.
func (s *Server) SetStatus(health, ready bool) {
	s.Lock()
	s.healthy = health
	s.ready = ready
	s.Unlock()
	log.Debug().Bool("health", health).Bool("ready", ready).Msg("server status set")
}

// URL returns the endpoint of the server as determined by the configuration and the
// socket address and port (if specified).
func (s *Server) URL() string {
	s.RLock()
	defer s.RUnlock()
	return s.url.String()
}

func (s *Server) setURL(addr net.Addr) {
	s.Lock()
	defer s.Unlock()

	s.url = &url.URL{
		Scheme: "http",
		Host:   addr.String(),
	}

	if s.srv.TLSConfig != nil {
		s.url.Scheme = "https"
	}

	if tcp, ok := addr.(*net.TCPAddr); ok && tcp.IP.IsUnspecified() {
		s.url.Host = fmt.Sprintf("127.0.0.1:%d", tcp.Port)
	}
}

// Debug returns a server that uses the specified http server instead of creating one.
// This function is primarily used to create test servers easily.
func Debug(conf config.Config, srv *http.Server) (s *Server, err error) {
	if s, err = New(conf); err != nil {
		return nil, err
	}

	// Replace the http server with the one specified
	s.srv = nil
	s.srv = srv
	s.srv.Handler = s.router
	return s, nil
}
