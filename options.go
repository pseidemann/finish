package finish

import "time"

// An Option can be used to change the behavior when registering a server via [Finisher.Add].
type Option option

type option func(keeper *serverKeeper) error

// WithTimeout overrides the global [Finisher].Timeout for the server to be registered via [Finisher.Add].
func WithTimeout(timeout time.Duration) Option {
	return func(keeper *serverKeeper) error {
		keeper.timeout = timeout
		return nil
	}
}

// WithName sets a custom name for the server to be registered via [Finisher.Add].
//
// If there will be only one server registered, the name defaults to “server”.
// Otherwise, the names of the servers default to “server #<num>”.
func WithName(name string) Option {
	return func(keeper *serverKeeper) error {
		keeper.name = name
		return nil
	}
}
