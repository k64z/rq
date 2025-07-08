package rq

import (
	"net/http"
	"time"
)

var defaultClient = &http.Client{
	Timeout: 30 * time.Second,
}

// ClientOption defines a function type for configuring HTTP clients
type ClientOption func(*http.Client)
