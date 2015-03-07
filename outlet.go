package shuttle

type Outlet interface {
	// Processes batches and blocks until done sending.
	Outlet()
}

type NewOutletFunc func(s *Shuttle) Outlet
