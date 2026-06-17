package v2beta1

import "sync"

// Client is the entry point for the v2beta1 API. It is defined fully in this
// file; this stub is extended by task-005 with the full constructor, options,
// and service fields.
type Client struct {
	mu               sync.RWMutex
	defaultOrgID     string
	defaultProjectID string
}
