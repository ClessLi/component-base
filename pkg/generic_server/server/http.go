package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/ClessLi/component-base/pkg/core"
	"github.com/ClessLi/component-base/pkg/errors"
	logV1 "github.com/ClessLi/component-base/pkg/log/v1"
	"github.com/marmotedu/iam/pkg/shutdown/shutdownmanagers/posixsignal"

	"github.com/gin-gonic/gin"
	"github.com/marmotedu/component-base/pkg/version"
	"github.com/marmotedu/iam/pkg/shutdown"
	ginprometheus "github.com/zsais/go-gin-prometheus"
	"golang.org/x/sync/errgroup"
)

// APIServerConfig is a structure used to configure a GenericAPIServer.
// Its members are sorted roughly in order of importance for composers.
type APIServerConfig[Svc any, Store any] struct {
	SecureServing   *SecureServingInfo
	InsecureServing *InsecureServingInfo
	Middlewares     []string
	Healthz         bool
	EnableProfiling bool
	EnableMetrics   bool
	Mode            string
	HealthCheckPath string
	MaxPingCount    int
}

// Complete fills in any fields not set that are required to have valid data and can be derived
// from other fields. If you're going to `ApplyOptions`, do that first. It's mutating the receiver.
func (c *APIServerConfig[Svc, Store]) Complete() CompletedAPIServerConfig[Svc, Store] {
	return CompletedAPIServerConfig[Svc, Store]{APIServerConfig: c}
}

// NewAPIServerConfig returns a APIServerConfig struct with the default values.
func NewAPIServerConfig[Svc any, Store any]() *APIServerConfig[Svc, Store] {
	return &APIServerConfig[Svc, Store]{
		Healthz:         true,
		Middlewares:     []string{},
		EnableProfiling: true,
		EnableMetrics:   true,
	}
}

// CompletedAPIServerConfig is the completed configuration for GenericAPIServer.
type CompletedAPIServerConfig[Svc any, Store any] struct {
	*APIServerConfig[Svc, Store]
}

// GenericAPIServer contains state for a generic API server.
type GenericAPIServer[Svc any, Store any] struct {
	// middlewares is the list of middleware names to install.
	middlewares []string

	// SecureServingInfo holds configuration of the TLS server.
	SecureServingInfo *SecureServingInfo

	// InsecureServingInfo holds configuration of the insecure http server.
	InsecureServingInfo *InsecureServingInfo

	// healthz indicates whether the healthz endpoint is enabled.
	healthz bool

	// EnableMetrics indicates whether the metrics endpoint is enabled.
	EnableMetrics bool

	// enableProfiling indicates whether the profiling endpoint is enabled.
	enableProfiling bool

	// Router is the gin engine.
	Router *gin.Engine

	// insecureServer is the insecure http server.
	insecureServer *http.Server

	// secureServer is the secure https server.
	secureServer *http.Server

	// installHooks provides lifecycle hooks for installation.
	installHooks APIServerInstallHooks[Svc, Store]

	// installedSvcPointer holds the initialized service factory instance.
	installedSvcPointer *Svc

	// gs manages graceful shutdown.
	gs *shutdown.GracefulShutdown

	// ShutdownTimeout is the timeout used for server shutdown.
	ShutdownTimeout time.Duration
}

// PreparedGenericAPIServer is the interface for a prepared server ready to run.
type PreparedGenericAPIServer interface {
	Run() error
}

type preparedGenericAPIServer[Svc any, Store any] struct {
	*GenericAPIServer[Svc, Store]
}

// Run starts the shutdown manager and the HTTP server.
func (p *preparedGenericAPIServer[Svc, Store]) Run() error {
	logV1.Debug("Starting prepared generic APIServer...")

	// Start the shutdown manager
	if err := p.gs.Start(); err != nil {
		logV1.Fatalf("Failed to start shutdown manager: %v", err)
	}

	return p.GenericAPIServer.run()
}

// Setup initializes the Gin router.
func (s *GenericAPIServer[Svc, Store]) Setup() {
	if s.Router == nil {
		logV1.Debug("Initializing Gin router...")
		s.Router = gin.New()
	}
}

// installMiddlewares installs necessary and custom middlewares.
func (s *GenericAPIServer[Svc, Store]) installMiddlewares() error {
	// Install built-in recovery middleware
	s.Router.Use(gin.Recovery())
	logV1.Debug("Installed built-in recovery middleware")

	// Get custom middleware registry
	mws, err := s.installHooks.RegisteredMiddlewares()
	if err != nil {
		return errors.Wrap(err, "failed to get middleware registry")
	}

	// Install custom middlewares in configured order
	for _, name := range s.middlewares {
		mw, exists := mws[name]
		if !exists {
			return errors.Errorf("middleware not found: %s", name)
		}

		*s.installedSvcPointer = mw(*s.installedSvcPointer)
		logV1.Infof("Installed middleware: %s", name)
	}

	return nil
}

// installServices initializes store, service, middlewares, and controllers.
func (s *GenericAPIServer[Svc, Store]) installServices() error {
	// Initialize storage layer
	logV1.Debug("Initializing store layer...")
	store, err := s.installHooks.InitStore()
	if err != nil {
		return errors.Wrap(err, "failed to initialize store")
	}
	logV1.Debug("Store layer initialized successfully")

	// Initialize service layer
	logV1.Debug("Initializing service layer...")
	svc, err := s.installHooks.InitService(store)
	if err != nil {
		return errors.Wrap(err, "failed to initialize service")
	}
	s.installedSvcPointer = &svc
	logV1.Debug("Service layer initialized successfully")

	// Install middlewares
	logV1.Debug("Installing middlewares...")
	if err := s.installMiddlewares(); err != nil {
		return errors.Wrap(err, "failed to install middlewares")
	}
	logV1.Debug("Middlewares installed successfully")

	// Install metrics handler
	if s.EnableMetrics {
		logV1.Debug("Installing metrics handler...")
		prometheus := ginprometheus.NewPrometheus("gin")
		prometheus.ReqCntURLLabelMappingFn = func(c *gin.Context) string {
			url := c.Request.URL.Path
			for _, p := range c.Params {
				if p.Key == "id" || p.Key == "name" {
					url = url[:len(url)-len(p.Value)] + "{" + p.Key + "}"
				}
			}
			return url
		}
		prometheus.Use(s.Router)
		logV1.Debug("Metrics handler installed successfully")
	}

	// Install healthz handler
	if s.healthz {
		logV1.Debug("Installing healthz handler...")
		s.Router.GET("/healthz", func(c *gin.Context) {
			core.WriteResponse(c, nil, map[string]string{"status": "ok"})
		})
	}

	// Install profiling handler
	if s.enableProfiling {
		// TODO: Add profiling handlers (pprof, trace, etc.)
		logV1.Debug("Profiling handlers not yet implemented")
	}

	// Install version handler
	logV1.Debug("Installing version handler...")
	s.Router.GET("/version", func(c *gin.Context) {
		core.WriteResponse(c, nil, version.Get())
	})

	// Initialize controller layer
	logV1.Debug("Initializing controller layer...")
	if err := s.installHooks.InitController(s.Router, *s.installedSvcPointer); err != nil {
		return errors.Wrap(err, "failed to initialize controller")
	}
	logV1.Debug("Controller layer initialized successfully")

	return nil
}

// NewAPIServer returns a new instance of GenericAPIServer from the given config.
func (c CompletedAPIServerConfig[Svc, Store]) NewAPIServer(installHooks APIServerInstallHooks[Svc, Store]) *GenericAPIServer[Svc, Store] {
	if installHooks == nil {
		logV1.Panicf("installHooks must not be nil")
	}

	// Set Gin mode
	if c.Mode != "" {
		gin.SetMode(c.Mode)
		logV1.Infof("Gin mode set to: %s", c.Mode)
	}

	// Initialize graceful shutdown manager
	gs := shutdown.New()
	gs.AddShutdownManager(posixsignal.NewPosixSignalManager())

	return &GenericAPIServer[Svc, Store]{
		SecureServingInfo:   c.SecureServing,
		InsecureServingInfo: c.InsecureServing,
		healthz:             c.Healthz,
		EnableMetrics:       c.EnableMetrics,
		enableProfiling:     c.EnableProfiling,
		middlewares:         c.Middlewares,
		installHooks:        installHooks,
		gs:                  gs,
	}
}

// PrepareRun prepares the server for running by installing services and registering shutdown callbacks.
func (s *GenericAPIServer[Svc, Store]) PrepareRun() PreparedGenericAPIServer {
	logV1.Info("Preparing generic APIServer for running...")

	// Initialize router
	s.Setup()

	// Install all services
	if err := s.installServices(); err != nil {
		logV1.Panicf("Failed to install services: %+v", err)
	}

	// TODO: Decouple resource uninstall logic from ShutdownFunc into independent hooks.
	// Current: ShutdownFunc contains all cleanup logic for Store/Service/Controller.
	// Future: Add UninstallStore/UninstallMiddlewares/UninstallController hooks,
	// and register each uninstall callback separately in this method.
	s.gs.AddShutdownCallback(s.installHooks.ShutdownFunc())
	logV1.Debug("Registered shutdown callback")

	// TODO: Provide lifecycle stage extension points for Pre/Post hooks.
	// Current: Installation flow is linear and hardcoded: Store → Service → Middleware → Controller.
	// Future: Allow users to inject custom logic before/after each stage,
	// e.g., PreInitStore, PostInitStore, PreInitController, PostInitController.

	logV1.Info("Generic APIServer prepared successfully")
	return &preparedGenericAPIServer[Svc, Store]{s}
}

// run spawns the http server. It only returns when the port cannot be listened on initially.
func (s *GenericAPIServer[Svc, Store]) run() error {
	var eg errgroup.Group

	// Start insecure server if configured
	if s.InsecureServingInfo != nil {
		logV1.Infof("Starting insecure HTTP server on %s", s.InsecureServingInfo.Address())
		s.insecureServer = &http.Server{
			Addr:    s.InsecureServingInfo.Address(),
			Handler: s.Router,
		}

		eg.Go(func() error {
			return s.startServer(s.insecureServer, false, "", "")
		})

		// Ping the server to verify the router is working
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := s.ping(ctx); err != nil {
			return errors.Wrap(err, "server health check failed")
		}
		logV1.Info("Insecure HTTP server started successfully")
	}

	// Start secure server if configured
	if s.SecureServingInfo != nil {
		logV1.Infof("Starting secure HTTPS server on %s", s.SecureServingInfo.Address())
		s.secureServer = &http.Server{
			Addr:    s.SecureServingInfo.Address(),
			Handler: s.Router,
		}

		eg.Go(func() error {
			return s.startServer(s.secureServer, true, s.SecureServingInfo.CertKey.CertFile, s.SecureServingInfo.CertKey.KeyFile)
		})
		logV1.Info("Secure HTTPS server started successfully")
	}

	if err := eg.Wait(); err != nil {
		return errors.Wrap(err, "server exited with error")
	}

	return nil
}

// startServer starts the HTTP or HTTPS server based on the provided configuration.
func (s *GenericAPIServer[Svc, Store]) startServer(server *http.Server, isSecure bool, certFile, keyFile string) error {
	var err error
	if isSecure {
		logV1.Debugf("Starting HTTPS server with cert=%s, key=%s", certFile, keyFile)
		err = server.ListenAndServeTLS(certFile, keyFile)
	} else {
		logV1.Debug("Starting HTTP server")
		err = server.ListenAndServe()
	}

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return errors.Wrap(err, "server failed to start")
	}

	return nil
}

// Close gracefully shuts down the API server.
func (s *GenericAPIServer[Svc, Store]) Close() {
	logV1.Info("Shutting down API server...")

	ctx, cancel := context.WithTimeout(context.Background(), s.ShutdownTimeout)
	defer cancel()

	var eg errgroup.Group

	if s.insecureServer != nil {
		eg.Go(func() error {
			logV1.Debug("Shutting down insecure server...")
			if err := s.insecureServer.Shutdown(ctx); err != nil {
				logV1.Errorf("Failed to shutdown insecure server: %v", err)
				return err
			}
			logV1.Debug("Insecure server shutdown complete")
			return nil
		})
	}

	if s.secureServer != nil {
		eg.Go(func() error {
			logV1.Debug("Shutting down secure server...")
			if err := s.secureServer.Shutdown(ctx); err != nil {
				logV1.Errorf("Failed to shutdown secure server: %v", err)
				return err
			}
			logV1.Debug("Secure server shutdown complete")
			return nil
		})
	}

	if err := eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		logV1.Warnf("Server shutdown completed with errors: %v", err)
	} else {
		logV1.Info("API server shutdown complete")
	}
}

// ping verifies the HTTP server is responding to health checks.
func (s *GenericAPIServer[Svc, Store]) ping(ctx context.Context) error {
	url := "http://" + s.InsecureServingInfo.Address() + "/healthz"
	if strings.Contains(s.InsecureServingInfo.Address(), "0.0.0.0") {
		url = "http://127.0.0.1" + strings.Split(s.InsecureServingInfo.Address(), ":")[1] + "/healthz"
	}

	logV1.Debugf("Pinging server health check endpoint: %s", url)

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return errors.Wrap(err, "failed to create health check request")
		}

		resp, err := http.DefaultClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			logV1.Debug("Server health check passed")
			return nil
		}

		// Sleep for a second before retrying
		time.Sleep(1 * time.Second)

		select {
		case <-ctx.Done():
			return errors.New("server health check timed out")
		default:
		}
	}
}
