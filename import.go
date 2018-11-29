package yamlpack

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/spf13/viper"
)

//YamlFile stores raw file bytes and Viper struct
type YamlSection struct {
	Bytes []byte
	Viper *viper.Viper
}

//Import reads data from a single YAML file and adds its data to this *Yp instance
func (yp *Yp) ImportFile(s string) error {
	r, err := os.Open(s)
	if err != nil {
		return err
	}
	yf, err := importYaml(bufio.NewReader(r))
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

//Import takes a location identifier (URI, file path, etc..) and an io.Reader
//imported data is added to the yamlPack instance
func (yp *Yp) Import(s string, r io.Reader) error {
	yf, err := importYaml(r)
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

func importYaml(r io.Reader) ([]*YamlSection, error) {
	sections := []*YamlSection{}
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(r)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("could not read data %v", err))
	}
	data := buf.Bytes()
	rxChunks := regexp.MustCompile(`---`)
	chunks := rxChunks.FindAllIndex(data, -1)
	if len(chunks) == 0 {
		//This is missing the end delimiter ('---') or both,
		//either way we are treating the whole file as 1 section
		chunks = [][]int{[]int{
			int(0), int(len(data)),
		}}
	}
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
