package yamlpack

import (
	"fmt"

	yaml "gopkg.in/yaml.v2"
)

//Yaml returns the "data" value as a string
//DEPRECATED: this is used only in yaml2vars and will be removed in the future
func (ys *YamlSection) Yaml() (string, error) {
	s, err := yaml.Marshal(ys.Viper.Sub("data").AllSettings())
	if err != nil {
		return "", fmt.Errorf("Failed to export yaml: %v", err)
	}
	return string(s), nil
}

//String returns the sections processed data as a string
func (ys *YamlSection) String() string {
	return string(ys.Bytes)
}
