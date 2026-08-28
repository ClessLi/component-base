package server

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ClessLi/component-base/pkg/core"
	"github.com/ClessLi/component-base/pkg/errors"
	logV1 "github.com/ClessLi/component-base/pkg/log/v1"
	"github.com/marmotedu/component-base/pkg/version"
	"github.com/marmotedu/iam/pkg/shutdown/shutdownmanagers/posixsignal"

	"github.com/gin-gonic/gin"
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
	PingTimeout     time.Duration
	ShutdownTimeout time.Duration
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
	// middlewareNames is the list of middleware names to install.
	middlewareNames []string

	// SecureServingInfo holds configuration of the TLS server.
	SecureServingInfo *SecureServingInfo

	// InsecureServingInfo holds configuration of the insecure http server.
	InsecureServingInfo *InsecureServingInfo

	// healthz indicates whether the healthz endpoint is enabled.
	healthz bool

	// HealthCheckPath is the path for the health check endpoint.
	HealthCheckPath string

	// PingTimeout is the timeout duration for health check pings.
	PingTimeout time.Duration

	// EnableMetrics indicates whether the metrics endpoint is enabled.
	EnableMetrics bool

	// enableProfiling indicates whether the profiling endpoint is enabled.
	enableProfiling bool

	// Router is the gin engine.
	Router *gin.Engine

	// mu protects insecureServer and secureServer from concurrent access.
	mu sync.Mutex

	// insecureServer is the insecure http server.
	insecureServer *http.Server

	// secureServer is the secure https server.
	secureServer *http.Server

	// bizHooks provides lifecycle hooks for installation.
	bizHooks APIServerBizHooks[Svc, Store]

	// svcPtr holds the initialized service factory instance.
	svcPtr *Svc

	// gs manages graceful shutdown.
	gs *shutdown.GracefulShutdown

	// ShutdownTimeout is the timeout used for server shutdown.
	ShutdownTimeout time.Duration

	// parentCtx is the parent context passed from the caller.
	parentCtx context.Context
	// ctx is the internal context used for the server lifecycle.
	ctx context.Context
	// cancel is the function to cancel the internal server context.
	cancel context.CancelFunc

	// state tracks the server lifecycle state using atomic operations.
	state atomic.Int32

	// shutdownBeforeRun indicates if Close() was called before Run().
	// When true, Run() should be rejected to prevent starting after shutdown signal.
	shutdownBeforeRun atomic.Bool
}

// PreparedGenericAPIServer is the interface for a prepared server ready to run.
type PreparedGenericAPIServer interface {
	Run() error
	Close() error
}

type preparedGenericAPIServer[Svc any, Store any] struct {
	*GenericAPIServer[Svc, Store]
}

// initRouter initializes the gin router.
func (s *GenericAPIServer[Svc, Store]) initRouter() {
	if s.Router == nil {
		logV1.Debug("Initializing gin router")
		s.Router = gin.Default()
	}
}

// initMiddlewares installs necessary and custom middlewares.
func (s *GenericAPIServer[Svc, Store]) initMiddlewares() error {
	if s.svcPtr == nil {
		return errors.New("service pointer not initialized, initBiz must be called first")
	}

	// Install built-in recovery middleware
	s.Router.Use(gin.Recovery())
	logV1.Debug("Built-in recovery middleware installed")

	// Get custom middleware registry
	mws, err := s.bizHooks.RegisteredMiddlewares()
	if err != nil {
		return errors.Wrap(err, "failed to get middleware registry")
	}

	// Install custom middlewares in configured order
	for _, name := range s.middlewareNames {
		mw, exists := mws[name]
		if !exists {
			return errors.Errorf("middleware not found: %s", name)
		}

		*s.svcPtr = mw(*s.svcPtr)
		logV1.Infof("Middleware installed: %s", name)
	}

	return nil
}

// initBiz initializes store, service, middlewares, and controllers.
func (s *GenericAPIServer[Svc, Store]) initBiz() error {
	// Defer cleanup to handle partial initialization failures.
	// Only triggers if installed remains false (i.e., an error occurred).
	installed := false
	defer func() {
		if !installed {
			s.cleanupBiz()
		}
	}()

	// Initialize storage layer
	logV1.Debug("Initializing store layer")
	store, err := s.bizHooks.InitStore()
	if err != nil {
		return errors.Wrap(err, "failed to initialize store")
	}
	logV1.Debug("Store layer initialized")

	// Initialize service layer
	logV1.Debug("Initializing service layer")
	svc, err := s.bizHooks.InitService(store)
	if err != nil {
		return errors.Wrap(err, "failed to initialize service")
	}
	s.svcPtr = &svc
	logV1.Debug("Service layer initialized")

	// Install middlewares
	logV1.Debug("Installing middlewares")
	if err := s.initMiddlewares(); err != nil {
		return errors.Wrap(err, "failed to install middlewares")
	}
	logV1.Debug("Middlewares installed")

	// Install metrics handler
	if s.EnableMetrics {
		logV1.Debug("Installing metrics handler")
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
		logV1.Debug("Metrics handler installed")
	}

	// Install healthz handler
	if s.healthz {
		logV1.Debug("Installing healthz handler")
		s.Router.GET(s.HealthCheckPath, func(c *gin.Context) {
			core.WriteResponse(c, nil, map[string]string{"status": "ok"})
		})
	}

	// Install profiling handler
	if s.enableProfiling {
		// TODO: Add profiling handlers (pprof, trace, etc.)
		logV1.Debug("Profiling handlers not yet implemented")
	}

	// Install version handler
	logV1.Debug("Installing version handler")
	s.Router.GET("/version", func(c *gin.Context) {
		core.WriteResponse(c, nil, version.Get())
	})

	// Initialize controller layer
	logV1.Debug("Initializing controller layer")
	if err := s.bizHooks.InitController(s.Router, *s.svcPtr); err != nil {
		return errors.Wrap(err, "failed to initialize controller")
	}
	logV1.Debug("Controller layer initialized")

	// TODO: Provide lifecycle stage extension points for Pre/Post hooks.
	// Current: Installation flow is linear and hardcoded: Store → Service → Middleware → Controller.
	// Future: Allow users to inject custom logic before/after each stage,
	// e.g., PreInitStore, PostInitStore, PreInitController, PostInitController.

	installed = true
	return nil
}

// initLifecycle initializes and starts the graceful shutdown manager.
// It registers Close() as the unified shutdown callback and starts listening for system signals.
func (s *GenericAPIServer[Svc, Store]) initLifecycle() error {
	// Register Close() as the unified shutdown callback.
	// This ensures both HTTP server shutdown and business resource cleanup
	// are executed in a coordinated manner, regardless of the trigger source.
	s.gs.AddShutdownCallback(shutdown.ShutdownFunc(func(shutdownManager string) error {
		logV1.Infof("Shutdown triggered by manager [%s], initiating graceful shutdown", shutdownManager)
		if err := s.Close(); err != nil {
			logV1.Errorf("Failed to close Generic APIServer: %+v", err)
			return err
		}
		return nil
	}))
	logV1.Debug("Shutdown callback registered")

	// Start the shutdown manager to listen for system signals (e.g., SIGTERM, SIGINT).
	// Starting here ensures gs is initialized only once and avoids duplicate signal listeners.
	if err := s.gs.Start(); err != nil {
		logV1.Errorf("Failed to start shutdown manager: %+v", err)
		return err
	}
	logV1.Debug("Shutdown manager started")

	// TODO: Handle shutdown signal arriving before Run() is called.
	// Current: If a shutdown signal arrives between PrepareRun() and Run(),
	//   Close() will be called but return immediately (state == Created),
	//   leaving the server able to start on subsequent Run() calls.
	//   This may not match user expectations (signal received = should not start).
	// Future: Consider adding a shutdownRequested flag to prevent Run() after signal.
	//   Alternatively, delay gs.Start() to Run() with duplicate-start protection.

	return nil
}

// cleanupBiz cleans up all resources initialized during initBiz.
// It returns an error if cleanup fails, allowing callers to handle it appropriately.
// In defer scenarios, the error can be ignored or logged.
// Safe to call multiple times (idempotent via Shutdown implementation).
func (s *GenericAPIServer[Svc, Store]) cleanupBiz() error {
	if s.bizHooks != nil {
		if err := s.bizHooks.Shutdown(); err != nil {
			logV1.Errorf("Failed to cleanup business resources: %+v", err)
			return err
		}
	}
	return nil
}

// cleanupRunState cleans up resources initialized during run.
// It cancels the internal context and shuts down business resources.
// Errors from cleanupBiz are logged but not returned, as this is called in defer.
func (s *GenericAPIServer[Svc, Store]) cleanupRunState() {
	if s.cancel != nil {
		s.cancel()
	}
	_ = s.cleanupBiz()
}

// NewAPIServer returns a new GenericAPIServer instance from the given config.
func (c CompletedAPIServerConfig[Svc, Store]) NewAPIServer(ctx context.Context, bizHooks APIServerBizHooks[Svc, Store]) *GenericAPIServer[Svc, Store] {
	if bizHooks == nil {
		logV1.Panicf("bizHooks must not be nil")
	}
	if ctx == nil {
		logV1.Panicf("ctx must not be nil")
	}

	// Set gin mode
	if c.Mode != "" {
		gin.SetMode(c.Mode)
		logV1.Infof("Gin mode set to %s", c.Mode)
	}

	// Initialize graceful shutdown manager
	gs := shutdown.New()
	gs.AddShutdownManager(posixsignal.NewPosixSignalManager())

	s := &GenericAPIServer[Svc, Store]{
		SecureServingInfo:   c.SecureServing,
		InsecureServingInfo: c.InsecureServing,
		healthz:             c.Healthz,
		HealthCheckPath:     c.HealthCheckPath,
		PingTimeout:         c.PingTimeout,
		EnableMetrics:       c.EnableMetrics,
		enableProfiling:     c.EnableProfiling,
		middlewareNames:     c.Middlewares,
		bizHooks:            bizHooks,
		gs:                  gs,
		parentCtx:           ctx,
	}
	s.state.Store(ServerStateCreated)

	return s
}

// PrepareRun prepares the server for running by installing services and registering shutdown callbacks.
func (s *GenericAPIServer[Svc, Store]) PrepareRun() PreparedGenericAPIServer {
	logV1.Info("Preparing generic APIServer")

	// Initialize router
	s.initRouter()

	// Install all services
	if err := s.initBiz(); err != nil {
		logV1.Panicf("Failed to install services: %+v", err)
	}

	// Initialize and start shutdown manager
	if err := s.initLifecycle(); err != nil {
		logV1.Panicf("Failed to initialize lifecycle: %+v", err)
	}

	logV1.Info("Generic APIServer prepared")
	return &preparedGenericAPIServer[Svc, Store]{s}
}

// Run starts the HTTP server. The shutdown manager is already started in PrepareRun.
func (p *preparedGenericAPIServer[Svc, Store]) Run() error {
	logV1.Debug("Starting prepared generic APIServer")

	// Reject if shutdown signal was received before Run() was called.
	// This prevents starting the server after resources have been cleaned up.
	if p.shutdownBeforeRun.Load() {
		return errors.New("server shutdown was requested before Run(), cannot start")
	}

	// Check state: created or stopped to running
	currentState := p.state.Load()
	if currentState != ServerStateCreated && currentState != ServerStateStopped {
		return errors.Errorf("server cannot run in current state: %d", currentState)
	}

	// Check if parent context is still valid before creating a new child context
	if p.parentCtx.Err() != nil {
		return errors.Wrapf(p.parentCtx.Err(), "parent context is no longer valid")
	}

	// Rebuild context for each run cycle to ensure a fresh, uncancelled context
	p.ctx, p.cancel = context.WithCancel(p.parentCtx)

	logV1.Debugf("Server state transition: %d -> Running", currentState)
	p.state.Store(ServerStateRunning)

	return p.GenericAPIServer.run()
}

// run spawns the HTTP server. It only returns when the port cannot be listened on initially.
func (s *GenericAPIServer[Svc, Store]) run() error {
	var eg errgroup.Group

	// Defer cleanup to handle startup failure or server exit with error.
	// Only triggers if cleanupNeeded remains true (i.e., an error occurred).
	cleanupNeeded := true
	defer func() {
		if cleanupNeeded {
			s.cleanupRunState()
		}
	}()

	// Call startup hook
	if err := s.bizHooks.Startup(); err != nil {
		s.transitionToStopped("startup failure")
		return errors.Wrap(err, "failed to startup business resources")
	}

	// Start insecure server if configured
	if s.InsecureServingInfo != nil {
		if err := s.startHTTPServer(
			&eg,
			s.InsecureServingInfo.Address(),
			false,
			"",
			"",
			s.ping,
			"Insecure",
			func(server *http.Server) { s.insecureServer = server },
			func() *http.Server { return s.insecureServer },
		); err != nil {
			return err
		}
	}

	// Start secure server if configured
	if s.SecureServingInfo != nil {
		if err := s.startHTTPServer(
			&eg,
			s.SecureServingInfo.Address(),
			true,
			s.SecureServingInfo.CertKey.CertFile,
			s.SecureServingInfo.CertKey.KeyFile,
			nil, // TODO: Implement pingSecure for HTTPS health check
			"Secure",
			func(server *http.Server) { s.secureServer = server },
			func() *http.Server { return s.secureServer },
		); err != nil {
			return err
		}
	}

	if err := eg.Wait(); err != nil {
		s.transitionToStopped("error")
		return errors.Wrap(err, "server exited with error")
	}

	s.transitionToStopped("normal exit")

	// Normal exit, no cleanup needed (resources are cleaned up by Close)
	cleanupNeeded = false
	return nil
}

// startHTTPServer starts an HTTP or HTTPS server with proper lifecycle management.
// It handles old server cleanup, new server creation, goroutine startup, and health check.
func (s *GenericAPIServer[Svc, Store]) startHTTPServer(
	eg *errgroup.Group,
	address string,
	isSecure bool,
	certFile, keyFile string,
	pingFunc func(ctx context.Context) error,
	pingName string,
	setServer func(server *http.Server),
	getOldServer func() *http.Server,
) error {
	serverType := "insecure HTTP"
	if isSecure {
		serverType = "secure HTTPS"
	}
	logV1.Infof("Starting %s server on %s", serverType, address)

	// Save and clear old server reference under lock
	s.mu.Lock()
	oldServer := getOldServer()
	setServer(nil)
	s.mu.Unlock()

	// Shutdown previous server outside the lock to avoid blocking concurrent access
	if oldServer != nil {
		logV1.Warnf("Previous %s server instance exists, shutting down before overwrite", serverType)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), s.ShutdownTimeout)
		if err := oldServer.Shutdown(shutdownCtx); err != nil {
			logV1.Errorf("Failed to shutdown previous %s server: %+v", serverType, err)
		}
		shutdownCancel()
	}

	// Create new server under lock
	s.mu.Lock()
	newServer := &http.Server{
		Addr:    address,
		Handler: s.Router,
	}
	setServer(newServer)
	serverToStart := newServer
	s.mu.Unlock()

	eg.Go(func() error {
		var err error
		if isSecure {
			logV1.Debugf("Starting HTTPS server with cert=%s, key=%s", certFile, keyFile)
			err = serverToStart.ListenAndServeTLS(certFile, keyFile)
		} else {
			logV1.Debug("Starting HTTP server")
			err = serverToStart.ListenAndServe()
		}

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return errors.Wrap(err, "server failed to start")
		}
		return nil
	})

	// Health check if provided
	if pingFunc != nil {
		pingCtx, pingCancel := context.WithTimeout(s.ctx, s.PingTimeout)
		defer pingCancel()

		if err := pingFunc(pingCtx); err != nil {
			// Shutdown the server goroutine to prevent resource leak
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), s.ShutdownTimeout)
			if shutdownErr := serverToStart.Shutdown(shutdownCtx); shutdownErr != nil {
				logV1.Errorf("Failed to shutdown %s server after health check failure: %+v", pingName, shutdownErr)
			}
			shutdownCancel()
			return errors.Wrapf(err, "%s server health check failed", pingName)
		}
	}

	logV1.Infof("%s server started", serverType)
	return nil
}

// Close gracefully shuts down the API server.
// Safe to call multiple times; concurrent calls are idempotent.
func (s *GenericAPIServer[Svc, Store]) Close() error {
	// Atomic state transition: Running -> Closing
	if !s.state.CompareAndSwap(ServerStateRunning, ServerStateClosing) {
		currentState := s.state.Load()
		if currentState == ServerStateCreated {
			logV1.Debug("Server not started, cleaning up business resources")
			s.shutdownBeforeRun.Store(true)
			if err := s.cleanupBiz(); err != nil {
				return err
			}
			logV1.Debug("Business resources cleaned up")
			s.state.Store(ServerStateStopped)
			return nil
		}
		if currentState == ServerStateClosing {
			logV1.Debug("Server close already in progress")
			return nil
		}
		if currentState == ServerStateStopped {
			logV1.Debug("Server already closed")
			return nil
		}
		return errors.Errorf("server cannot be closed in current state: %d", currentState)
	}

	logV1.Debug("Server state transition: Running -> Closing")

	// Perform actual close
	err := s.close()

	s.state.Store(ServerStateStopped)
	logV1.Debug("Server state transition: Closing -> Stopped")

	return err
}

// transitionToStopped atomically transitions the server state from Running to Stopped.
func (s *GenericAPIServer[Svc, Store]) transitionToStopped(reason string) {
	if s.state.CompareAndSwap(ServerStateRunning, ServerStateStopped) {
		logV1.Debugf("Server state transition: Running -> Stopped (reason: %s)", reason)
	}
}

// stopHTTPServer adds a server shutdown task to the error group.
func (s *GenericAPIServer[Svc, Store]) stopHTTPServer(eg *errgroup.Group, server *http.Server, serverType string) {
	eg.Go(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), s.ShutdownTimeout)
		defer cancel()
		logV1.Debugf("Shutting down %s server", serverType)
		if err := server.Shutdown(ctx); err != nil {
			logV1.Errorf("Failed to shutdown %s server: %+v", serverType, err)
			return err
		}
		logV1.Debugf("%s server shutdown complete", serverType)
		return nil
	})
}

// close performs the actual shutdown logic.
func (s *GenericAPIServer[Svc, Store]) close() error {
	logV1.Info("Shutting down API server")

	var eg errgroup.Group

	s.mu.Lock()
	insecureServer := s.insecureServer
	secureServer := s.secureServer
	s.mu.Unlock()

	if insecureServer != nil {
		s.stopHTTPServer(&eg, insecureServer, "insecure")
	}

	if secureServer != nil {
		s.stopHTTPServer(&eg, secureServer, "secure")
	}

	// Wait for server shutdown, but continue to clean up business resources even if failed
	errs := make([]error, 0)
	if err := eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		logV1.Warnf("API server shutdown completed with errors: %+v", err)
		errs = append(errs, err)
	}

	// Cancel internal context to propagate shutdown signal to all dependent goroutines.
	// cancel may be nil if Close is called before Run creates the context.
	// This is done AFTER server graceful shutdown to allow in-flight requests to complete.
	if s.cancel != nil {
		s.cancel()
	}

	// Clean up business resources (Store/Service/Controller) regardless of server shutdown result.
	// Reuse cleanupBiz to centralize cleanup logic and error handling.
	if err := s.cleanupBiz(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		logV1.Error("API server shutdown completed with errors")
		return errors.NewAggregate(errs)
	}

	logV1.Info("API server shutdown completed")
	return nil
}

// ping verifies the HTTP server is responding to health checks.
func (s *GenericAPIServer[Svc, Store]) ping(ctx context.Context) error {
	addr := s.InsecureServingInfo.Address()
	if strings.HasPrefix(addr, "0.0.0.0") {
		idx := strings.LastIndex(addr, ":")
		if idx == -1 {
			return errors.Errorf("invalid address format: %s", addr)
		}
		addr = "127.0.0.1" + addr[idx:]
	}
	url := "http://" + addr + s.HealthCheckPath

	logV1.Debugf("Pinging health check endpoint: %s", url)

	// Use HTTP client with timeout to avoid hanging requests
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Exponential backoff: 1s -> 2s -> 4s -> 8s (max 8s)
	backoff := 1 * time.Second
	maxBackoff := 8 * time.Second

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return errors.Wrap(err, "failed to create health check request")
		}

		resp, err := client.Do(req)
		if err == nil {
			if resp.StatusCode == http.StatusOK {
				logV1.Debug("Health check passed")
				resp.Body.Close()
				return nil
			}
			resp.Body.Close()
		}

		// Sleep with exponential backoff before retrying
		time.Sleep(backoff)
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}

		select {
		case <-ctx.Done():
			return errors.New("health check timed out")
		default:
		}
	}
}
