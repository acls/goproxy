package conf

import (
	"encoding/json"
	"os"
	"path"

	"github.com/go-yaml/yaml"
	"go.uber.org/zap"
)

type configurer interface {
	SetDefaultsAndValidate() error
}

func parseYaml(b []byte, c configurer) error {
	return afterParse(yaml.Unmarshal(b, c), c)
}
func parseJSON(b []byte, c configurer) error {
	return afterParse(json.Unmarshal(b, c), c)
}
func parseFile(confPath string, c configurer) error {
	f, err := os.Open(confPath)
	if err != nil {
		zap.L().Warn("Open",
			zap.Any("confPath", confPath),
			zap.Error(err),
		)
		return err
	}
	defer f.Close()

	var decode func(c interface{}) error
	if path.Ext(confPath) == ".json" {
		decode = json.NewDecoder(f).Decode
	} else {
		decode = yaml.NewDecoder(f).Decode
	}
	return afterParse(decode(c), c)
}

func afterParse(err error, c configurer) error {
	if err != nil {
		zap.L().Warn("afterParse",
			zap.Any("c", c),
			zap.Error(err),
		)
		return err
	}
	return c.SetDefaultsAndValidate()
}
