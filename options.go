package finish

import "time"

// Option is what a functional option returns. The function which is returned
// applies the actual values when invoked later by the Option recipient.
type Option func(keeper *serverKeeper) error

// WithTimeout overrides the global Finisher.Timeout for this specific server.
func WithTimeout(timeout time.Duration) Option {
	return func(keeper *serverKeeper) error {
		keeper.timeout = timeout
		return nil
	}
}

// WithName sets a custom name for the server to register.
//
// The default name is "server" if there will be only one server registered,
// otherwise the names default to "server #<num>".
func WithName(name string) Option {
	return func(keeper *serverKeeper) error {
		keeper.name = name
		return nil
	}
}
