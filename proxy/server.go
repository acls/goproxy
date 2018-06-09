package proxy

import (
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"

	vhost "github.com/inconshreveable/go-vhost"
	"go.uber.org/zap"

	"slt/conf"
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

func (s *Server) Run() error {
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
	for name, front := range s.Frontends {
		if err := s.AddFrontend(name, front); err != nil {
			s.Warn("Failed to add frontend",
				zap.String("name", name),
				zap.Error(err),
			)
		}
	}

	// custom error handler so we can log errors
	go func() {
		for {
			conn, err := s.mux.NextError()

			if conn == nil {
				s.Error("Failed to mux next connection", zap.Error(err))
				if _, ok := err.(vhost.Closed); ok {
					return
				}
				continue
			} else {
				// if _, ok := err.(vhost.NotFound); ok && s.DefaultFrontend != nil {
				// 	go s.DefaultFrontend.proxyConnection(conn)
				// } else {
				s.Error("Failed to mux connection",
					zap.String("from", conn.RemoteAddr().String()),
					zap.Error(err),
				)
				// XXX: respond with valid TLS close messages
				conn.Close()
				// }
			}
		}
	}()

	// we're ready, signal it for testing
	if s.ready != nil {
		close(s.ready)
	}

	<-s.stop
	s.stop = nil

	return nil
}
func (s *Server) Stop() {
	if s.stop != nil {
		close(s.stop)
	}
}

func (s *Server) AddFrontend(name string, front *conf.Frontend) error {
	s.frontendsL.Lock()
	defer s.frontendsL.Unlock()

	if s.frontends == nil {
		s.frontends = make(map[string]*frontend)
	}

	f, ok := s.frontends[name]
	if ok {
		return fmt.Errorf("Frontend %s already exists", name)
	}

	var tlsConfig *tls.Config
	if front.TLSCrt != "" || front.TLSKey != "" {
		var err error
		if tlsConfig, err = loadTLSConfig(front.TLSCrt, front.TLSKey); err != nil {
			err = fmt.Errorf("%s: Failed to load TLS configuration for frontend '%v': %v", s.Name, name, err)
			return err
		}
	}

	l, err := s.mux.Listen(name)
	if err != nil {
		return err
	}

	f = &frontend{
		Name:      name,
		Logger:    s.Logger,
		TLSConfig: tlsConfig,
		Listener:  l,
		// always round-robi,n strategy for now
		Strategy: &RoundRobinStrategy{backends: front.Backends},
	}
	s.frontends[name] = f

	go f.Run()
	return nil
}

func (s *Server) RemoveFrontend(name string) error {
	s.frontendsL.Lock()
	defer s.frontendsL.Unlock()

	if s.frontends == nil {
		s.frontends = make(map[string]*frontend)
	}

	f, ok := s.frontends[name]
	if !ok {
		return fmt.Errorf("Frontend %s does not exist", name)
	}

	s.removeFrontend(name, f)
	return nil
}

func (s *Server) RemoveFrontends() {
	s.frontendsL.Lock()
	defer s.frontendsL.Unlock()
	for name, f := range s.frontends {
		s.removeFrontend(name, f)
	}
}

func (s *Server) removeFrontend(name string, f *frontend) {
	delete(s.frontends, name)
	if err := f.Listener.Close(); err != nil {
		s.Warn("Close frontend connection error",
			zap.String("name", name),
			zap.Error(err),
		)
	}
}
