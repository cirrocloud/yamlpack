package yamlpack

import (
	"fmt"

	yaml "gopkg.in/yaml.v2"
)

func (ys *YamlSection) Yaml() (string, error) {
	s, err := yaml.Marshal(ys.Viper.Sub("data").AllSettings())
	if err != nil {
		return "", fmt.Errorf("Failed to export yaml: %v", err)
	}
	return string(s), nil
}
