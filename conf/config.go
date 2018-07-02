package conf

import (
	"fmt"
)

// NewConfiguration returns a new Configuration
func NewConfiguration() Configuration {
	return make(Configuration)
}

// Configuration map
type Configuration map[string]*Binding

// Binding struct
type Binding struct {
	BindAddr  string               `yaml:"bind_addr" json:"bindAddr"`
	Watch     bool                 `yaml:"watch" json:"watch"`
	Secure    bool                 `yaml:"secure" json:"secure"`
	Frontends map[string]*Frontend `yaml:"frontends" json:"frontends"`

	// DefaultFrontend *Frontend `yaml:"-" json:"-"`
}

// ParseYaml func
func (c Configuration) ParseYaml(b []byte) error {
	return parseYaml(b, c)
}

// ParseJSON func
func (c Configuration) ParseJSON(b []byte) error {
	return parseJSON(b, c)
}

// ParseFile func
func (c Configuration) ParseFile(confPath string) error {
	return parseFile(confPath, c)
}

// SetDefaultsAndValidate sets defaults and validates
func (c Configuration) SetDefaultsAndValidate() error {
	for key, val := range c {
		val.BindAddr = key

		if !val.Watch && len(val.Frontends) == 0 {
			return fmt.Errorf("%s: Must specify at least one frontend", key)
		}

		for name, front := range val.Frontends {
			front.Name = name
			front.BoundAddr = val.BindAddr
			if err := front.SetDefaultsAndValidate(); err != nil {
				return err
			}
		}
	}

	return nil
}
