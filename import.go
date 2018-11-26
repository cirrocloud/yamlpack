package yamlpack

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"regexp"

	"github.com/spf13/viper"
)

//YamlFile stores raw file bytes and Viper struct
type YamlSection struct {
	Bytes []byte
	Viper *viper.Viper
}

//Import reads data from a single YAML file and adds its data to this *Yp instance
func (yp *Yp) Import(s string) error {
	yf, err := importYaml(s)
	if err != nil {
		return err
	}
	yp.Lock()
	defer func() {
		yp.Unlock()
	}()
	yp.Files[s] = yf
	return nil
}

func importYaml(s string) ([]*YamlSection, error) {
	sections := []*YamlSection{}
	data, err := ioutil.ReadFile(s)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("could not read file '%v': %v", s, err))
	}
	rxChunks := regexp.MustCompile(`---`)
	chunks := rxChunks.FindAllIndex(data, -1)
	for i := range chunks {
		vp := viper.New()
		vp.SetConfigType("yaml")
		var b []byte
		if i < len(chunks)-1 {
			b = data[chunks[i][1]:chunks[i+1][0]]
		} else {
			b = data[chunks[i][1]:]
		}
		vp.ReadConfig(bytes.NewBuffer(b))
		sections = append(sections, &YamlSection{Bytes: b, Viper: vp})
	}
	return sections, nil
}
