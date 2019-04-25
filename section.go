package yamlpack

import "github.com/spf13/viper"

//YamlFile stores raw file bytes and Viper struct
type YamlSection struct {
	Bytes         []byte
	OriginalBytes []byte // Pre-template functions
	Viper         *viper.Viper
	TemplateFunc  TemplateFunc
}
