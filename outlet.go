package shuttle

// Outlet takes batches of log lines, applies zero or more formatters and
// delivers them to a target destination.
type Outlet interface {
	// Processes batches and blocks until done sending.
	Outlet()
}

// NewOutletFunc defines a function for creating and outlet from a given config
type NewOutletFunc func(s *Shuttle) (Outlet, error)
