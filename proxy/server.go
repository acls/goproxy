package proxy

import (
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"

	vhost "github.com/acls/go-vhost"
	"github.com/acls/goproxy/conf"
	"go.uber.org/zap"
)

const (
	muxTimeout = 10 * time.Second
)

var loadTLSConfig = func(crtPath, keyPath string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(crtPath, keyPath)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}, nil
}

// Server listens and proxies connections
type Server struct {
	Name string
	*zap.Logger
	*conf.Binding

	frontendsL sync.Mutex
	frontends  map[string]*frontend

	stop chan struct{}

	// these are for easier testing
	mux   muxer
	ready chan struct{}
}
type muxer interface {
	Listen(name string) (net.Listener, error)
	NextError() (net.Conn, error)
	Close()
}

// Init func
func (s *Server) Init() {
	if s.ready == nil {
		s.ready = make(chan struct{})
	}
}

// Ready func
func (s *Server) Ready() <-chan struct{} {
	return s.ready
}

// Run starts the server
func (s *Server) Run() error {
	if s.ready == nil {
		return fmt.Errorf("%s must call init", s.Name)
	}
	if s.stop != nil {
		return fmt.Errorf("%s already running", s.Name)
	}
	s.stop = make(chan struct{})

	// bind to port
	l, err := net.Listen("tcp", s.BindAddr)
	if err != nil {
		return err
	}
	s.Info("Serving connections", zap.String("addr", l.Addr().String()))

	// start muxing on port
	if s.Secure {
		s.mux, err = vhost.NewTLSMuxer(l, muxTimeout)
	} else {
		s.mux, err = vhost.NewHTTPMuxer(l, muxTimeout)
	}
	if err != nil {
		return err
	}

	defer s.RemoveFrontends()

	// setup muxing for each frontend
	for _, front := range s.Frontends {
		if err := s.AddFrontend(front); err != nil {
			s.Warn("Failed to add frontend",
				zap.String("name", front.Name),
				zap.Error(err),
			)
			continue
		}
		s.Debug("Added frontend",
			zap.String("name", front.Name),
		)
	}

	// custom error handler so we can log errors
	go func() {
		for {
			conn, err := s.mux.NextError()

			switch err.(type) {
			case vhost.BadRequest:
				s.Error("got a bad request!", zap.Error(err))
				conn.Write([]byte("bad request"))
			case vhost.NotFound:
				s.Error("got a connection for an unknown vhost", zap.Error(err))
				conn.Write([]byte("vhost not found"))
			case vhost.Closed:
				s.Error("closed conn", zap.Error(err))
			default:
				if conn != nil {
					conn.Write([]byte("server error"))
				}
			}

			if conn != nil {
				conn.Close()
			}
		}
	}()

	// signal we're ready
	close(s.ready)

	<-s.stop
	s.stop = nil

	return nil
}

// Stop stops the server
func (s *Server) Stop() {
	if s.stop != nil {
		close(s.stop)
	}
}

// ReplaceFrontend replaces the frontend
func (s *Server) ReplaceFrontend(front *conf.Frontend) error {
	s.frontendsL.Lock()
	defer s.frontendsL.Unlock()

	if f, ok := s.frontends[front.Name]; ok {
		s.removeFrontend(f)
	}
	return s.addFrontend(front)
}

// AddFrontend adds the frontend
func (s *Server) AddFrontend(front *conf.Frontend) error {
	s.frontendsL.Lock()
	defer s.frontendsL.Unlock()

	return s.addFrontend(front)
}
func (s *Server) addFrontend(front *conf.Frontend) error {
	if s.frontends == nil {
		s.frontends = make(map[string]*frontend)
	}

	f, ok := s.frontends[front.Name]
	if ok {
		return fmt.Errorf("Frontend %s already exists", front.Name)
	}

	var tlsConfig *tls.Config
	if front.TLSCrt != "" || front.TLSKey != "" {
		var err error
		if tlsConfig, err = loadTLSConfig(front.TLSCrt, front.TLSKey); err != nil {
			err = fmt.Errorf("%s: Failed to load TLS configuration for frontend '%v': %v", s.Name, front.Name, err)
			return err
		}
	}

	l, err := s.mux.Listen(front.Name)
	if err != nil {
		return err
	}

	f = &frontend{
		Name:      front.Name,
		BoundAddr: front.BoundAddr,
		Logger:    s.Logger,
		TLSConfig: tlsConfig,
		Listener:  l,
		// always round-robin strategy for now
		Strategy: &RoundRobinStrategy{backends: front.Backends},
	}
	s.frontends[f.Name] = f

	go f.Run()
	return nil
}

// RemoveFrontend removes the frontend
func (s *Server) RemoveFrontend(name string) {
	s.frontendsL.Lock()
	defer s.frontendsL.Unlock()

	if s.frontends == nil {
		s.frontends = make(map[string]*frontend)
	}

	f, ok := s.frontends[name]
	if !ok {
		f.Warn("Frontend doesn't exist",
			zap.String("name", name),
		)
	} else {
		s.removeFrontend(f)
	}
}

// RemoveFrontends removes all the frontends
func (s *Server) RemoveFrontends() {
	s.frontendsL.Lock()
	defer s.frontendsL.Unlock()
	for _, f := range s.frontends {
		s.removeFrontend(f)
	}
}

func (s *Server) removeFrontend(f *frontend) {
	delete(s.frontends, f.Name)
	if err := f.Stop(); err != nil {
		s.Warn("Stop frontend connection error",
			zap.String("name", f.Name),
			zap.Error(err),
		)
	}
}
