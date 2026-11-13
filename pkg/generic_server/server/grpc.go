package server

import (
	"context"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	healthzclientv1 "github.com/ClessLi/component-base/pkg/client-sdk/grpc_health/v1"
	"github.com/ClessLi/component-base/pkg/errors"
	svcregisterv1 "github.com/ClessLi/component-base/pkg/generic_server/framework/grpc/service_register/v1"
	logV1 "github.com/ClessLi/component-base/pkg/log/v1"
	"github.com/ClessLi/skirnir/pkg/discover"
	"github.com/marmotedu/iam/pkg/shutdown"
	"github.com/marmotedu/iam/pkg/shutdown/shutdownmanagers/posixsignal"

	uuid "github.com/satori/go.uuid"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// RAInfo holds configuration of the Registration Authority server.
type RAInfo struct {
	Host string
	Port int
}

// GRPCServingInfo holds configuration for gRPC serving.
type GRPCServingInfo struct {
	ChunkSize    int
	RecvTimeoutM int
}

// NewGRPCServingInfo returns a GRPCServingInfo with default values.
func NewGRPCServingInfo() *GRPCServingInfo {
	return &GRPCServingInfo{
		ChunkSize:    1024,
		RecvTimeoutM: 1,
	}
}

// GRPCServerConfig is a structure used to configure a GenericGRPCServer.
// Its members are sorted roughly in order of importance for composers.
type GRPCServerConfig[Svc any, Store any] struct {
	Namespace       string
	SecureServing   *SecureServingInfo
	InsecureServing *InsecureServingInfo
	RA              *RAInfo
	GRPCServing     *GRPCServingInfo
	Middlewares     []string
	Healthz         bool
	EnableProfiling bool
	EnableMetrics   bool
	PingTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// NewGRPCServerConfig returns a GRPCServerConfig struct with the default values.
func NewGRPCServerConfig[Svc any, Store any]() *GRPCServerConfig[Svc, Store] {
	return &GRPCServerConfig[Svc, Store]{
		Healthz:         true,
		Middlewares:     []string{},
		EnableProfiling: true,
		EnableMetrics:   true,
		PingTimeout:     30 * time.Second,
		ShutdownTimeout: 30 * time.Second,
	}
}

// CompletedGRPCServerConfig is the completed configuration for GenericGRPCServer.
type CompletedGRPCServerConfig[Svc any, Store any] struct {
	*GRPCServerConfig[Svc, Store]
}

// Complete fills in any fields not set that are required to have valid data and can be derived
// from other fields. If you're going to `ApplyOptions`, do that first. It's mutating the receiver.
func (c *GRPCServerConfig[Svc, Store]) Complete() CompletedGRPCServerConfig[Svc, Store] {
	if c.GRPCServing == nil {
		c.GRPCServing = NewGRPCServingInfo()
	}
	if c.PingTimeout == 0 {
		c.PingTimeout = 30 * time.Second
	}
	if c.ShutdownTimeout == 0 {
		c.ShutdownTimeout = 30 * time.Second
	}
	return CompletedGRPCServerConfig[Svc, Store]{GRPCServerConfig: c}
}

// GenericGRPCServer contains state for a generic gRPC server.
type GenericGRPCServer[Txp, Ep, Svc, Store any] struct {
	// middlewareNames is the list of middleware names to install.
	middlewareNames []string

	// SecureServingInfo holds configuration of the TLS server.
	SecureServingInfo *SecureServingInfo

	// InsecureServingInfo holds configuration of the insecure grpc server.
	InsecureServingInfo *InsecureServingInfo

	// RAInfo holds configuration of the Registration Authority server.
	RAInfo *RAInfo

	// ChunkSize set chunk size of the grpc server.
	ChunkSize int

	// ReceiveTimeout is the timeout used for the grpc server receiving data.
	ReceiveTimeout time.Duration

	// PingTimeout is the timeout duration for health check pings.
	PingTimeout time.Duration

	// ShutdownTimeout is the timeout used for server shutdown.
	ShutdownTimeout time.Duration

	// healthz indicates whether the health check service is enabled.
	healthz bool

	// enableMetrics indicates whether the metrics interceptor is enabled.
	enableMetrics bool

	// enableProfiling indicates whether the profiling service is enabled.
	enableProfiling bool

	// mu protects server references and registration state from concurrent access.
	mu sync.Mutex

	// insecureServer is the insecure grpc server.
	insecureServer *grpc.Server

	// secureServer is the secure grpc server.
	secureServer *grpc.Server

	// insecureSvrHealthz and secureSvrHealthz are the health servers for each serving.
	insecureSvrHealthz, secureSvrHealthz *health.Server

	// instanceSuffixID is the unique suffix for RA service instance IDs.
	instanceSuffixID string

	// registryClient is the RA registration client.
	registryClient discover.RegistryClient

	// registeredService is the list of service names registered to RA.
	registeredService []string

	// serviceRegisters holds the gRPC service register map built by biz hooks.
	serviceRegisters map[string]svcregisterv1.ServiceRegister

	// bizHooks provides lifecycle hooks for installation.
	bizHooks GRPCServerBizHooks[Txp, Ep, Svc, Store]

	// svcPtr holds the initialized service factory instance.
	svcPtr *Svc

	// gs manages graceful shutdown.
	gs *shutdown.GracefulShutdown

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

// PreparedGenericGRPCServer is the interface for a prepared server ready to run.
type PreparedGenericGRPCServer interface {
	Run() error
	Close() error
}

type preparedGenericGRPCServer[Txp, Ep, Svc, Store any] struct {
	*GenericGRPCServer[Txp, Ep, Svc, Store]
}

// installRAClient initializes the RA registration client and instance suffix ID.
func (s *GenericGRPCServer[Txp, Ep, Svc, Store]) installRAClient() error {
	if s.RAInfo == nil || (s.RAInfo.Host == "" && s.RAInfo.Port == 0) {
		return nil
	}

	client, err := discover.NewKitConsulRegistryClient(s.RAInfo.Host, uint16(s.RAInfo.Port))
	if err != nil {
		return errors.Wrap(err, "failed to create RA registry client")
	}

	id, err := uuid.NewV4()
	if err != nil {
		return errors.Wrap(err, "failed to generate instance suffix ID")
	}

	s.registryClient = client
	s.instanceSuffixID = id.String()
	return nil
}

// initMiddlewares installs custom middlewares in configured order.
func (s *GenericGRPCServer[Txp, Ep, Svc, Store]) initMiddlewares() error {
	if s.svcPtr == nil {
		return errors.New("service pointer not initialized, initBiz must be called first")
	}

	// TODO: Install built-in gRPC interceptors (recovery, request size limiter, etc.).

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

// initBiz initializes store, service, middlewares, endpoints, transport, and service registers.
func (s *GenericGRPCServer[Txp, Ep, Svc, Store]) initBiz() error {
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

	// Install metrics interceptors
	if s.enableMetrics {
		// TODO: Add gRPC metrics interceptors (e.g. grpc-prometheus).
		logV1.Debug("gRPC metrics interceptors not yet implemented")
	}

	// Install profiling service
	if s.enableProfiling {
		// TODO: Add gRPC profiling service (e.g. channelz, pprof over gRPC).
		logV1.Debug("gRPC profiling service not yet implemented")
	}

	// Initialize endpoint layer
	logV1.Debug("Initializing endpoint layer")
	eps, err := s.bizHooks.InitEndpoints(*s.svcPtr)
	if err != nil {
		return errors.Wrap(err, "failed to initialize endpoints")
	}
	logV1.Debug("Endpoint layer initialized")

	// Initialize transport layer (chunkSize injected from the server config)
	logV1.Debug("Initializing transport layer")
	txp, err := s.bizHooks.InitTransport(eps, s.ChunkSize)
	if err != nil {
		return errors.Wrap(err, "failed to initialize transport")
	}
	logV1.Debug("Transport layer initialized")

	// Build gRPC service registers (counterpart of HTTP InitController).
	// Biz hooks only declare what to register; the framework owns the registration
	// control flow (health status transitions, pb register invocation).
	logV1.Debug("Initializing service registers")
	registrations, err := s.bizHooks.ServiceRegistrations(txp)
	if err != nil {
		return errors.Wrap(err, "failed to build service registrations")
	}
	s.serviceRegisters = svcregisterv1.GenerateServiceRegisters(registrations...)
	logV1.Debug("Service registers initialized")

	installed = true
	return nil
}

// initLifecycle initializes and starts the graceful shutdown manager.
// It registers Close() as the unified shutdown callback and starts listening for system signals.
func (s *GenericGRPCServer[Txp, Ep, Svc, Store]) initLifecycle() error {
	s.gs.AddShutdownCallback(shutdown.ShutdownFunc(func(shutdownManager string) error {
		logV1.Infof("Shutdown triggered by manager [%s], initiating graceful shutdown", shutdownManager)
		if err := s.Close(); err != nil {
			logV1.Errorf("Failed to close generic gRPC server: %+v", err)
			return err
		}
		return nil
	}))
	logV1.Debug("Shutdown callback registered")

	if err := s.gs.Start(); err != nil {
		logV1.Errorf("Failed to start shutdown manager: %+v", err)
		return err
	}
	logV1.Debug("Shutdown manager started")

	return nil
}

// cleanupBiz cleans up all resources initialized during initBiz.
// Safe to call multiple times (idempotent via Shutdown implementation).
func (s *GenericGRPCServer[Txp, Ep, Svc, Store]) cleanupBiz() error {
	if s.bizHooks != nil {
		if err := s.bizHooks.Shutdown(); err != nil {
			logV1.Errorf("Failed to cleanup business resources: %+v", err)
			return err
		}
	}
	return nil
}

// cleanupRunState cleans up resources initialized during run.
// It cancels the internal context, deregisters services, stops servers,
// and shuts down business resources. Called in defer on failure paths.
func (s *GenericGRPCServer[Txp, Ep, Svc, Store]) cleanupRunState() {
	if s.cancel != nil {
		s.cancel()
	}
	s.deregisterServices()
	s.stopGRPCServers()
	_ = s.cleanupBiz()
}

// NewGRPCServer returns a new GenericGRPCServer instance from the given config.
// It is a package-level function because the endpoint/transport type parameters
// are runtime assembly concerns that the config and options layers do not carry.
func NewGRPCServer[Txp any, Ep any, Svc any, Store any](ctx context.Context, c CompletedGRPCServerConfig[Svc, Store], bizHooks GRPCServerBizHooks[Txp, Ep, Svc, Store]) *GenericGRPCServer[Txp, Ep, Svc, Store] {
	if bizHooks == nil {
		logV1.Panicf("bizHooks must not be nil")
	}
	if ctx == nil {
		logV1.Panicf("ctx must not be nil")
	}

	// Initialize graceful shutdown manager
	gs := shutdown.New()
	gs.AddShutdownManager(posixsignal.NewPosixSignalManager())

	s := &GenericGRPCServer[Txp, Ep, Svc, Store]{
		SecureServingInfo:   c.SecureServing,
		InsecureServingInfo: c.InsecureServing,
		RAInfo:              c.RA,
		ChunkSize:           c.GRPCServing.ChunkSize,
		ReceiveTimeout:      time.Duration(c.GRPCServing.RecvTimeoutM) * time.Minute,
		PingTimeout:         c.PingTimeout,
		ShutdownTimeout:     c.ShutdownTimeout,
		healthz:             c.Healthz,
		enableMetrics:       c.EnableMetrics,
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
func (s *GenericGRPCServer[Txp, Ep, Svc, Store]) PrepareRun() PreparedGenericGRPCServer {
	logV1.Info("Preparing generic gRPC server")

	if s.SecureServingInfo == nil && s.InsecureServingInfo == nil {
		logV1.Panicf("no serving information is available for setup")
	}

	// Install RA client
	if err := s.installRAClient(); err != nil {
		logV1.Panicf("Failed to install RA client: %+v", err)
	}

	// Install all services
	if err := s.initBiz(); err != nil {
		logV1.Panicf("Failed to install services: %+v", err)
	}

	// Initialize and start shutdown manager
	if err := s.initLifecycle(); err != nil {
		logV1.Panicf("Failed to initialize lifecycle: %+v", err)
	}

	logV1.Info("Generic gRPC server prepared")
	return &preparedGenericGRPCServer[Txp, Ep, Svc, Store]{s}
}

// Run starts the gRPC server. The shutdown manager is already started in PrepareRun.
func (p *preparedGenericGRPCServer[Txp, Ep, Svc, Store]) Run() error {
	logV1.Debug("Starting prepared generic gRPC server")

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

	return p.GenericGRPCServer.run()
}

// run spawns the gRPC servers. It only returns when the port cannot be listened on initially,
// the health check fails, or the servers exit.
func (s *GenericGRPCServer[Txp, Ep, Svc, Store]) run() error {
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

	// Build fresh servers for this run cycle (grpc.Server cannot be reused after stop)
	if err := s.buildServers(); err != nil {
		s.transitionToStopped("build servers failure")
		return errors.Wrap(err, "failed to build gRPC servers")
	}

	// Register health and business services, then register to RA
	s.registerServices()

	// Start insecure server if configured
	if s.InsecureServingInfo != nil {
		if err := s.startGRPCServer(&eg, false); err != nil {
			s.transitionToStopped("insecure server startup failure")
			return err
		}
	} else {
		logV1.Info("Pass start insecure gRPC server")
	}

	// Start secure server if configured
	if s.SecureServingInfo != nil {
		if err := s.startGRPCServer(&eg, true); err != nil {
			s.transitionToStopped("secure server startup failure")
			return err
		}
	} else {
		logV1.Info("Pass start secure gRPC server")
	}

	// Ping the servers to make sure all registered services are serving.
	if s.healthz {
		pingCtx, pingCancel := context.WithTimeout(s.ctx, s.PingTimeout)
		err := s.ping(pingCtx)
		pingCancel()
		if err != nil {
			s.transitionToStopped("health check failure")
			return errors.Wrap(err, "gRPC server health check failed")
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

// buildServers creates fresh grpc.Server and health.Server instances for each serving.
// TLS credentials are reloaded per run cycle to support certificate rotation.
func (s *GenericGRPCServer[Txp, Ep, Svc, Store]) buildServers() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.SecureServingInfo != nil {
		creds, err := credentials.NewServerTLSFromFile(
			s.SecureServingInfo.CertKey.CertFile,
			s.SecureServingInfo.CertKey.KeyFile,
		)
		if err != nil {
			return errors.Wrap(err, "failed to load server TLS credentials")
		}
		s.secureServer = grpc.NewServer(grpc.Creds(creds))
		if s.healthz {
			s.secureSvrHealthz = health.NewServer()
		}
	}

	if s.InsecureServingInfo != nil {
		s.insecureServer = grpc.NewServer()
		if s.healthz {
			s.insecureSvrHealthz = health.NewServer()
		}
	}

	return nil
}

// registerServices registers the health server and business services to each serving,
// and registers service instances to the RA.
func (s *GenericGRPCServer[Txp, Ep, Svc, Store]) registerServices() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.registeredService = nil

	if s.healthz {
		if s.insecureServer != nil {
			logV1.Debug("Register health server for insecure gRPC server...")
			grpc_health_v1.RegisterHealthServer(s.insecureServer, s.insecureSvrHealthz)
		}
		if s.secureServer != nil {
			logV1.Debug("Register health server for secure gRPC server...")
			grpc_health_v1.RegisterHealthServer(s.secureServer, s.secureSvrHealthz)
		}
	}

	for svcname, register := range s.serviceRegisters {
		if register == nil {
			logV1.Warnf("Service register is nil: %s", svcname)
			continue
		}

		if s.InsecureServingInfo != nil {
			logV1.Debugf("Register service `%s` for insecure gRPC server...", svcname)
			register(s.insecureServer, s.insecureSvrHealthz)
			s.registerToRA(svcname, s.InsecureServingInfo.BindAddress, uint16(s.InsecureServingInfo.BindPort))
		}
		if s.SecureServingInfo != nil {
			logV1.Debugf("Register service `%s` for secure gRPC server...", svcname)
			register(s.secureServer, s.secureSvrHealthz)
			s.registerToRA(svcname, s.SecureServingInfo.BindAddress, uint16(s.SecureServingInfo.BindPort))
		}

		s.registeredService = append(s.registeredService, svcname)
	}
}

// registerToRA registers a service instance to the RA. Must be called with mu held.
func (s *GenericGRPCServer[Txp, Ep, Svc, Store]) registerToRA(svcname, bindHost string, bindPort uint16) {
	if s.registryClient == nil {
		return
	}

	instanceID := svcname + "-" + s.instanceSuffixID
	logV1.Debugf("Register service `%s` to RA for gRPC server...", instanceID)
	if ok := s.registryClient.Register(svcname, instanceID, bindHost, bindPort, nil, logV1.K()); !ok {
		logV1.Warnf("Failed to register service `%s` to RA", instanceID)
	}
}

// deregisterServices deregisters all service instances from the RA.
// Safe to call multiple times.
func (s *GenericGRPCServer[Txp, Ep, Svc, Store]) deregisterServices() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.registryClient == nil {
		return
	}

	for _, svcname := range s.registeredService {
		instanceID := svcname + "-" + s.instanceSuffixID
		logV1.Debugf("Deregister service `%s` from RA for gRPC server...", instanceID)
		if ok := s.registryClient.DeRegister(instanceID, logV1.K()); !ok {
			logV1.Warnf("Failed to deregister service `%s` from RA", instanceID)
		}
	}
	s.registeredService = nil
}

// startGRPCServer starts an insecure or secure gRPC server with proper lifecycle management.
// The listener is created synchronously so that port conflicts are reported immediately.
func (s *GenericGRPCServer[Txp, Ep, Svc, Store]) startGRPCServer(eg *errgroup.Group, isSecure bool) error {
	var (
		serverType string
		address    string
		server     *grpc.Server
		healthSvr  *health.Server
	)

	s.mu.Lock()
	if isSecure {
		serverType = "secure"
		address = s.SecureServingInfo.Address()
		server = s.secureServer
		healthSvr = s.secureSvrHealthz
	} else {
		serverType = "insecure"
		address = s.InsecureServingInfo.Address()
		server = s.insecureServer
		healthSvr = s.insecureSvrHealthz
	}
	s.mu.Unlock()

	logV1.Infof("Starting %s gRPC server on %s", serverType, address)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return errors.Wrapf(err, "failed to listen on %s", address)
	}

	if s.healthz {
		healthSvr.Resume()
	}

	eg.Go(func() error {
		if err := server.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			return errors.Wrapf(err, "%s gRPC server failed", serverType)
		}
		logV1.Infof("%s gRPC server on %s stopped", serverType, address)
		return nil
	})

	return nil
}

// stopGRPCServers stops all servers gracefully with a forced stop fallback.
// Safe to call when servers are nil or already stopped.
func (s *GenericGRPCServer[Txp, Ep, Svc, Store]) stopGRPCServers() {
	s.mu.Lock()
	insecureServer, secureServer := s.insecureServer, s.secureServer
	insecureHealth, secureHealth := s.insecureSvrHealthz, s.secureSvrHealthz
	s.mu.Unlock()

	var eg errgroup.Group
	if insecureServer != nil {
		s.stopGRPCServer(&eg, insecureServer, insecureHealth, "insecure")
	}
	if secureServer != nil {
		s.stopGRPCServer(&eg, secureServer, secureHealth, "secure")
	}
	_ = eg.Wait()
}

// stopGRPCServer adds a server shutdown task to the error group.
// GracefulStop is bounded by ShutdownTimeout; on timeout the server is force-stopped
// to prevent hanging on long-lived streaming RPCs.
func (s *GenericGRPCServer[Txp, Ep, Svc, Store]) stopGRPCServer(eg *errgroup.Group, server *grpc.Server, healthSvr *health.Server, serverType string) {
	eg.Go(func() error {
		logV1.Debugf("Shutting down %s gRPC server", serverType)

		done := make(chan struct{})
		go func() {
			server.GracefulStop()
			close(done)
		}()

		timer := time.NewTimer(s.ShutdownTimeout)
		defer timer.Stop()
		select {
		case <-done:
		case <-timer.C:
			logV1.Warnf("%s gRPC server graceful stop timed out after %s, forcing stop", serverType, s.ShutdownTimeout)
			server.Stop()
			<-done
		}

		if healthSvr != nil {
			healthSvr.Shutdown()
		}
		logV1.Debugf("%s gRPC server shutdown complete", serverType)
		return nil
	})
}

// Close gracefully shuts down the gRPC server.
// Safe to call multiple times; concurrent calls are idempotent.
func (s *GenericGRPCServer[Txp, Ep, Svc, Store]) Close() error {
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
func (s *GenericGRPCServer[Txp, Ep, Svc, Store]) transitionToStopped(reason string) {
	if s.state.CompareAndSwap(ServerStateRunning, ServerStateStopped) {
		logV1.Debugf("Server state transition: Running -> Stopped (reason: %s)", reason)
	}
}

// close performs the actual shutdown logic.
func (s *GenericGRPCServer[Txp, Ep, Svc, Store]) close() error {
	logV1.Info("Shutting down gRPC server")

	// Deregister from RA first so that no new traffic is routed to this instance.
	s.deregisterServices()

	// Gracefully stop servers, with a forced stop fallback after ShutdownTimeout.
	s.stopGRPCServers()

	// Cancel internal context to propagate shutdown signal to all dependent goroutines.
	// cancel may be nil if Close is called before Run creates the context.
	// This is done AFTER server graceful shutdown to allow in-flight requests to complete.
	if s.cancel != nil {
		s.cancel()
	}

	// Clean up business resources (Store/Service) regardless of server shutdown result.
	// Reuse cleanupBiz to centralize cleanup logic and error handling.
	if err := s.cleanupBiz(); err != nil {
		logV1.Error("gRPC server shutdown completed with errors")
		return err
	}

	logV1.Info("gRPC server shutdown completed")
	return nil
}

// setupPing creates a health check client for the given serving.
func (s *GenericGRPCServer[Txp, Ep, Svc, Store]) setupPing(isSecure bool) (*healthzclientv1.Client, error) {
	var (
		serverType string
		address    string
		creds      credentials.TransportCredentials
		err        error
	)

	if isSecure {
		serverType = "secure"
		address = s.SecureServingInfo.Address()
		creds, err = credentials.NewClientTLSFromFile(
			s.SecureServingInfo.CertKey.CertFile,
			s.SecureServingInfo.BindAddress,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load client TLS credentials")
		}
	} else {
		serverType = "insecure"
		address = s.InsecureServingInfo.Address()
		creds = insecure.NewCredentials()
	}

	logV1.Debugf("initialization %s server health check...", serverType)
	if strings.HasPrefix(address, "0.0.0.0") {
		idx := strings.LastIndex(address, ":")
		if idx == -1 {
			return nil, errors.Errorf("invalid address format: %s", address)
		}
		address = "127.0.0.1" + address[idx:]
	}

	return healthzclientv1.NewClient(address, grpc.WithTransportCredentials(creds))
}

// ping verifies all registered services on all servings are SERVING.
// It retries with exponential backoff until the context is done.
func (s *GenericGRPCServer[Txp, Ep, Svc, Store]) ping(ctx context.Context) error {
	healthzClients := make(map[string]*healthzclientv1.Client)
	if s.InsecureServingInfo != nil {
		client, err := s.setupPing(false)
		if err != nil {
			return errors.Wrap(err, "failed to setup insecure health check client")
		}
		defer func() {
			if err := client.Close(); err != nil {
				logV1.Warn(err.Error())
			}
		}()
		healthzClients["insecure"] = client
	}

	if s.SecureServingInfo != nil {
		client, err := s.setupPing(true)
		if err != nil {
			return errors.Wrap(err, "failed to setup secure health check client")
		}
		defer func() {
			if err := client.Close(); err != nil {
				logV1.Warn(err.Error())
			}
		}()
		healthzClients["secure"] = client
	}

	if len(healthzClients) == 0 {
		return errors.New("no serving information available for health check")
	}

	// Snapshot service names to avoid racing with deregistration.
	s.mu.Lock()
	svcnames := append([]string{}, s.registeredService...)
	s.mu.Unlock()
	if len(svcnames) == 0 {
		// Fall back to the overall server health status.
		svcnames = []string{""}
	}

	healthzOK := make(map[string]bool)

	// Exponential backoff: 1s -> 2s -> 4s -> 8s (max 8s)
	backoff := 1 * time.Second
	maxBackoff := 8 * time.Second

	for {
		for tag, client := range healthzClients {
			for _, svcname := range svcnames {
				key := tag + " " + svcname
				if healthzOK[key] {
					continue
				}

				status, err := client.Check(ctx, svcname)
				if err != nil {
					logV1.Debugf("Health check for `%s` failed: %v", key, err)
					healthzOK[key] = false
					continue
				}

				healthzOK[key] = status == healthzclientv1.SERVING
				if healthzOK[key] {
					logV1.Infof("The `%s` %s-service state is: %v", svcname, tag, healthzclientv1.StatusString(status))
				}
			}
		}

		allOK := true
		for key, ok := range healthzOK {
			if !ok {
				allOK = false
				logV1.Debugf("service `%s` is not healthy.", key)
			}
		}

		if allOK {
			logV1.Info("all services are healthy!")
			return nil
		}

		select {
		case <-ctx.Done():
			return errors.New("gRPC server health check timed out")
		case <-time.After(backoff):
		}
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}
