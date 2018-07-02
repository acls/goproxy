package proxy

import "github.com/acls/goproxy/conf"

// BackendStrategy interface
type BackendStrategy interface {
	NextBackend() conf.Backend
}

// RoundRobinStrategy interface
type RoundRobinStrategy struct {
	backends []conf.Backend
	idx      int
}

// NextBackend returns the next backend configuration
func (s *RoundRobinStrategy) NextBackend() conf.Backend {
	n := len(s.backends)

	if n == 1 {
		return s.backends[0]
	}
	s.idx = (s.idx + 1) % n
	return s.backends[s.idx]
}
