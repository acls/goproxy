package conf

import (
	"encoding/json"
	"fmt"

	"github.com/go-yaml/yaml"
)

type Configuration map[string]*Binding

type Binding struct {
	Secure    bool                 `yaml:"secure" json:"secure"`
	BindAddr  string               `yaml:"bind_addr" json:"bindAddr"`
	Frontends map[string]*Frontend `yaml:"frontends" json:"frontends"`

	DefaultFrontend *Frontend `yaml:"-" json:"-"`
}

type Frontend struct {
	Backends []Backend `yaml:"backends" json:"backends"`
	Strategy string    `yaml:"strategy" json:"strategy"`
	Default  bool      `yaml:"default" json:"default"`
	Autocert bool      `yaml:"autocert" json:"autocert"`
	TLSCrt   string    `yaml:"tls_crt" json:"tlsCrt"`
	TLSKey   string    `yaml:"tls_key" json:"tlsKey"`
}

type Backend struct {
	Addr           string `yaml:"addr" json:"addr"`
	ConnectTimeout int    `yaml:"connect_timeout" json:"connectTimeout"`
}

const (
	defaultConnectTimeout = 10000 // milliseconds
)

func ParseConfigYaml(configBuf []byte) (config Configuration, err error) {
	return parseConfig(configBuf, yaml.Unmarshal)
}
func ParseConfigJson(configBuf []byte) (config Configuration, err error) {
	return parseConfig(configBuf, json.Unmarshal)
}
func parseConfig(configBuf []byte, unmarshal func([]byte, interface{}) error) (config Configuration, err error) {
	// deserialize/parse the config
	config = make(Configuration)
	if err = unmarshal(configBuf, &config); err != nil {
		err = fmt.Errorf("Error parsing configuration file: %v", err)
		return
	}

	for key, val := range config {
		// configuration validation / normalization
		if val.BindAddr == "" {
			err = fmt.Errorf("%s: Must specify a bind_addr", key)
			return
		}

		if len(val.Frontends) == 0 {
			err = fmt.Errorf("%s: Must specify at least one frontend", key)
			return
		}

		for name, front := range val.Frontends {
			if len(front.Backends) == 0 {
				err = fmt.Errorf("%s: Must specify at least one backend for frontend '%v'", key, name)
				return
			}

			if front.Default {
				if val.DefaultFrontend != nil {
					err = fmt.Errorf("%s: Only one frontend may be the default", key)
					return
				}
				val.DefaultFrontend = front
			}

			for _, back := range front.Backends {
				if back.ConnectTimeout == 0 {
					back.ConnectTimeout = defaultConnectTimeout
				}

				if back.Addr == "" {
					err = fmt.Errorf("%s: Must specify an addr for each backend on frontend '%v'", key, name)
					return
				}
			}
		}
	}

	return
}
