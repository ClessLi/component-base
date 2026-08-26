package server

// Server state constants.
const (
	// ServerStateCreated indicates the server is created but not yet running.
	ServerStateCreated int32 = iota
	// ServerStateRunning indicates the server is currently running.
	ServerStateRunning
	// ServerStateClosing indicates the server is in the process of closing.
	ServerStateClosing
	// ServerStateStopped indicates the server has been stopped.
	ServerStateStopped
)
