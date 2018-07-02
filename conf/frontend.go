package conf

import (
	"fmt"

	"go.uber.org/zap"
)

const (
	defaultConnectTimeout = 10000 // milliseconds
)

// NewFrontend returns a new Configuration
func NewFrontend(bindAddr, name string, backends []Backend) *Frontend {
	return &Frontend{
		BoundAddr: bindAddr,
		Name:      name,
		Backends:  backends,
	}
}

// Frontend struct
type Frontend struct {
	Name      string `yaml:"-" json:"-"`
	BoundAddr string `yaml:"-" json:"-"`

	Backends []Backend `yaml:"backends" json:"backends"`
	Strategy string    `yaml:"strategy" json:"strategy"`
	Autocert bool      `yaml:"autocert" json:"autocert"`
	TLSCrt   string    `yaml:"tls_crt" json:"tlsCrt"`
	TLSKey   string    `yaml:"tls_key" json:"tlsKey"`
	// Default  bool      `yaml:"default" json:"default"`
}

// Backend struct
type Backend struct {
	Addr           string `yaml:"addr" json:"addr"`
	ConnectTimeout int    `yaml:"connect_timeout" json:"connectTimeout"`
}

// ParseYaml func
func (f *Frontend) ParseYaml(b []byte) error {
	return parseYaml(b, f)
}

// ParseJSON func
func (f *Frontend) ParseJSON(b []byte) error {
	return parseJSON(b, f)
}

// ParseFile func
func (f *Frontend) ParseFile(confPath string) error {
	return parseFile(confPath, f)
}

// SetDefaultsAndValidate sets defaults and validates
func (f *Frontend) SetDefaultsAndValidate() error {
	if len(f.Backends) == 0 {
		return fmt.Errorf("%s: Must specify at least one backend for frontend '%v'", f.BoundAddr, f.Name)
	}

	// if f.Default {
	// 	if val.DefaultFrontend != nil {
	// 		return fmt.Errorf("%s: Only one frontend may be the default", f.BoundAddr)
	// 	}
	// 	val.DefaultFrontend = f
	// }

	for i := range f.Backends {
		back := &f.Backends[i]
		if back.ConnectTimeout == 0 {
			back.ConnectTimeout = defaultConnectTimeout
		} else {
			zap.L().Debug("SetDefaultsAndValidate", zap.Int("asdf", back.ConnectTimeout))
		}

		if back.Addr == "" {
			return fmt.Errorf("%s: Must specify an addr for each backend on frontend '%v'", f.BoundAddr, f.Name)
		}
	}

	return nil
}
