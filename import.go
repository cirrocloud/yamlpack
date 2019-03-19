package yamlpack

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"

	errors "github.com/charter-se/structured/errors"
	"github.com/spf13/viper"
)

type TemplateFunc func([]byte) ([]byte, error)

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

func (yp *Yp) ImportFileWithTemplateFunc(s string, tmplFunc TemplateFunc) error {
	r, err := os.Open(s)
	if err != nil {
		return err
	}
	yf, err := importYamlWithTemplateFunc(bufio.NewReader(r), tmplFunc)
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

//ImportWithTemplateFuncAndFilter takes a location identifier (URI, file path, etc..), io.Reader, a templating function, and a filter function
//imported data is added to the yamlPack instance
func (yp *Yp) WithTemplateFuncAndFilter(s string, r io.Reader, tmplFunc TemplateFunc) error {
	yf, err := importYamlWithTemplateFunc(r, tmplFunc)
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
func (yp *Yp) ImportWithTemplateFunc(s string, r io.Reader, tmplFunc TemplateFunc) error {
	yf, err := importYamlWithTemplateFunc(r, tmplFunc)
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

func importYamlWithTemplateFunc(r io.Reader, tmplFunc TemplateFunc) ([]*YamlSection, error) {
	return importYamlWithTemplateFuncAndFilters(r, tmplFunc, []string{".+"})
}

//ImportWithTemplateFuncAndFilters takes a location identifier (URI, file path, etc..) and an io.Reader, template function, and filter string array (regexp)
//imported data is added to the yamlPack instance
//sections within that do not match any of the supplied regexp filters will be ignored
//the template function will be run on all sections that pass the filter before the section is added to the *Yp instance
func (yp *Yp) ImportWithTemplateFuncAndFilters(s string, r io.Reader, tmplFunc TemplateFunc, filters []string) error {
	yf, err := importYamlWithTemplateFuncAndFilters(r, tmplFunc, filters)
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

func importYamlWithTemplateFuncAndFilters(r io.Reader, tmplFunc TemplateFunc, filters []string) ([]*YamlSection, error) {
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
		//Each chunk is a document, needs to be filtered so the consumer gets the sections they want
		// then run the supplied template, then Vipered
		// this avoids running templates on unrelated sections that we may not have the data for

		//extract section
		var b []byte
		if i < len(chunks)-1 {
			b = data[chunks[i][1]:chunks[i+1][0]]
		} else {
			b = data[chunks[i][1]:]
		}

		//run filters
		if !func() bool {
			for _, v := range filters {
				rx := regexp.MustCompile(v)
				scanner := bufio.NewScanner(bytes.NewBuffer(b))
				for scanner.Scan() {
					if rx.MatchString(scanner.Text()) {
						return true
					}
				}
			}
			return false
		}() {
			continue
		}

		//run template
		b, err = tmplFunc(b)
		if err != nil {
			return nil, err
		}

		//add viper
		vp := viper.New()
		vp.SetConfigType("yaml")
		if err := vp.ReadConfig(bytes.NewBuffer(b)); err != nil {
			fmt.Printf("---\n%v\n", string(b))
			return nil, errors.Wrap(err, "failed to import yaml section")
		}

		//save completed section
		sections = append(sections, &YamlSection{Bytes: b, Viper: vp})
	}
	return sections, nil
}

func importYaml(r io.Reader) ([]*YamlSection, error) {
	templateFunc := func(b []byte) ([]byte, error) {
		return b, nil
	}
	return importYamlWithTemplateFunc(r, templateFunc)
}
