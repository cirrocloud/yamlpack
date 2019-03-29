package yamlpack

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"text/template"

	errors "github.com/charter-se/structured/errors"
	"github.com/spf13/viper"
)

type TemplateFunc func([]byte) ([]byte, error)

//YamlFile stores raw file bytes and Viper struct
type YamlSection struct {
	Bytes         []byte
	OriginalBytes []byte // Pre-template functions
	Viper         *viper.Viper
}

//Import takes a location identifier (URI, file path, etc..) and an io.Reader
//imported data is added to the yamlPack instance
func (yp *Yp) Import(s string, r io.Reader) error {
	yf, err := importRawSections(r)
	if err != nil {
		return err
	}
	yp.Lock()
	defer func() {
		yp.Unlock()
	}()
	yp.Files[s] = yf
	yp.applyNullTemplate(s)
	return yp.YamlParse(s)
}

//Import reads data from a single YAML file and adds its data to this *Yp instance
func (yp *Yp) ImportFile(s string) error {
	r, err := os.Open(s)
	if err != nil {
		return err
	}
	defer r.Close()
	return yp.Import(s, bufio.NewReader(r))
}

func (yp *Yp) YamlParse(name string) error {
	if _, ok := yp.Files[name]; !ok {
		return errors.WithFields(errors.Fields{
			"Name": name,
		}).New("File not imported")
	}
	for k, v := range yp.Files {
		for _, section := range v {
			//add viper
			vp := viper.New()
			vp.SetConfigType("yaml")
			if err := vp.ReadConfig(bytes.NewBuffer(section.Bytes)); err != nil {
				fmt.Printf("---\n%v\n", string(section.Bytes))
				return errors.WithFields(errors.Fields{
					"File": k,
				}).Wrap(err, "failed to parse yaml section")
			}
			section.Viper = vp
		}
	}
	return nil
}

// func (yp *Yp) ImportFileWithTemplateFunc(s string, tmplFunc TemplateFunc) error {
// 	r, err := os.Open(s)
// 	if err != nil {
// 		return err
// 	}
// 	yf, err := importYamlWithTemplateFunc(bufio.NewReader(r), tmplFunc)
// 	if err != nil {
// 		return err
// 	}
// 	yp.Lock()
// 	defer func() {
// 		yp.Unlock()
// 	}()
// 	yp.Files[s] = yf
// 	return nil
// }

// //ImportWithTemplateFuncAndFilter takes a location identifier (URI, file path, etc..), io.Reader, a templating function, and a filter function
// //imported data is added to the yamlPack instance
// func (yp *Yp) WithTemplateFuncAndFilter(s string, r io.Reader, tmplFunc TemplateFunc) error {
// 	yf, err := importYamlWithTemplateFunc(r, tmplFunc)
// 	if err != nil {
// 		return err
// 	}
// 	yp.Lock()
// 	defer func() {
// 		yp.Unlock()
// 	}()
// 	yp.Files[s] = yf
// 	return nil
// }

// //Import takes a location identifier (URI, file path, etc..) and an io.Reader
// //imported data is added to the yamlPack instance
// func (yp *Yp) ImportWithTemplateFunc(s string, r io.Reader, tmplFunc TemplateFunc) error {
// 	yf, err := importYamlWithTemplateFunc(r, tmplFunc)
// 	if err != nil {
// 		return err
// 	}
// 	yp.Lock()
// 	defer func() {
// 		yp.Unlock()
// 	}()
// 	yp.Files[s] = yf
// 	return nil
// }

// func importYamlWithTemplateFunc(r io.Reader, tmplFunc TemplateFunc) ([]*YamlSection, error) {
// 	return importYamlWithTemplateFuncAndFilters(r, tmplFunc, []string{".+"})
// }

// //ImportWithTemplateFuncAndFilters takes a location identifier (URI, file path, etc..) an io.Reader, template function, and filter string array (regexp)
// //imported data is added to the yamlPack instance
// //sections within that do not match any of the supplied regexp filters will be ignored
// //the template function will be run on all sections that pass the filter before the section is added to the *Yp instance
// func (yp *Yp) ImportWithTemplateFuncAndFilters(s string, r io.Reader, tmplFunc TemplateFunc, filters []string) error {
// 	yf, err := importYamlWithTemplateFuncAndFilters(r, tmplFunc, filters)
// 	if err != nil {
// 		return err
// 	}
// 	yp.Lock()
// 	defer func() {
// 		yp.Unlock()
// 	}()
// 	yp.Files[s] = yf
// 	return nil
// }

//importRawSections parses an io.Reader and returns []*YamlSections without the yaml handlers
// this allows for all sections to be processed in without necassarily filling in all of the template values at import time
func importRawSections(r io.Reader) ([]*YamlSection, error) {
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

		//add viper
		vp := viper.New()
		vp.SetConfigType("yaml")
		if err := vp.ReadConfig(bytes.NewBuffer(b)); err != nil {
			fmt.Printf("---\n%v\n", string(b))
			return nil, errors.Wrap(err, "failed to import yaml section")
		}

		//save completed section
		sections = append(sections, &YamlSection{
			Bytes:         b,
			OriginalBytes: b,
		})
	}
	return sections, nil
}

func Filter(in []*YamlSection, filters []string) ([]*YamlSection, error) {
	ret := []*YamlSection{}
	for _, section := range in {
		//Each section is a document, needs to be filtered so the consumer gets the sections they want
		// then run the supplied template, then Vipered
		// this avoids running templates on unrelated sections that we may not have the data for

		//run filters
		if !func() bool {
			for _, v := range filters {
				rx := regexp.MustCompile(v)
				scanner := bufio.NewScanner(bytes.NewBuffer(section.OriginalBytes))
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
		//save completed section
		ret = append(ret, section)
	}
	return ret, nil
}

func (yp *Yp) applyNullTemplate(name string) error {
	if _, ok := yp.Files[name]; !ok {
		return errors.WithFields(errors.Fields{"Name": name}).New("File has not been imported")
	}

	tf := func(in []byte) ([]byte, error) {
		renderedBytes := bytes.NewBuffer([]byte{})
		tmpl, err := template.New("default").Funcs(sprig.TxtFuncMap()).Parse(string(in))
		if err != nil {
			return nil, err
		}
		if err := tmpl.Execute(renderedBytes, make(map[string]interface{})); err != nil {
			fmt.Printf("---\n")
			fmt.Printf("%v\n", renderedBytes.String())
			fmt.Printf("---\n")
			return nil, errors.Wrap(err, "Failed to render template")
		}
		return renderedBytes.Bytes(), nil
	}

	for _, section := range yp.Files[name] {
		//run template
		b, err := tmplFunc(section.OriginalBytes)
		if err != nil {
			return err
		}
		section.Bytes = b
	}
	return nil
}

func (yp *Yp) ApplyTemplate(name string, tmplFunc TemplateFunc) error {
	if _, ok := yp.Files[name]; !ok {
		return errors.WithFields(errors.Fields{"Name": name}).New("File has not been imported")
	}
	for _, section := range yp.Files[name] {
		//run template
		b, err := tmplFunc(section.OriginalBytes)
		if err != nil {
			return err
		}
		section.Bytes = b
	}
	return nil
}

// func importYamlWithTemplateFuncAndFilters(r io.Reader, tmplFunc TemplateFunc, filters []string) ([]*YamlSection, error) {
// 	sections, err := importRawSections(r)
// 	if err != nil {
// 		return nil, err
// 	}

// 	sections = Filter(sections, filters)

// 		//run template
// 		section.Bytes, err = tmplFunc(section.OriginalBytes)
// 		if err != nil {
// 			return nil, err
// 		}

// 		//add/replace viper
// 		vp := viper.New()
// 		vp.SetConfigType("yaml")
// 		if err := vp.ReadConfig(bytes.NewBuffer(section.Bytes)); err != nil {
// 			fmt.Printf("---\n%v\n", string(section.Bytes))
// 			return nil, errors.Wrap(err, "failed to import yaml section")
// 		}

// 		//save completed section
// 		sections = append(sections, &YamlSection{Bytes: b, Viper: vp})
// 	}
// 	return sections, nil
// }
