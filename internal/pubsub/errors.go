package pubsub

import "errors"

var (
	// ErrNotConnected is returned when trying to use a disconnected adapter
	ErrNotConnected = errors.New("pubsub adapter not connected")

	// ErrInvalidAdapter is returned when an unknown adapter type is requested
	ErrInvalidAdapter = errors.New("invalid pubsub adapter type")

	// ErrAlreadyConnected is returned when trying to connect an already connected adapter
	ErrAlreadyConnected = errors.New("pubsub adapter already connected")
)
