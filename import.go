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

//Import takes a location identifier (URI, file path, etc..) and an io.Reader
//imported data is added to the yamlPack instance
func (yp *Yp) Import(s string, r io.Reader) error {
	yf, err := importRawSections(r)
	if err != nil {
		return errors.Wrap(err, "importRawSections failed in import")
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

func (yp *Yp) ImportWithTemplateFuncAndFilters(s string, r io.Reader, tf TemplateFunc, filters []string) error {
	if err := yp.Import(s, r); err != nil {
		return errors.Wrap(err, "Import failed")
	}
	if err := yp.ApplyTemplate(s, tf, nil); err != nil {
		return errors.Wrap(err, "ApplyTemplate")
	}
	if err := yp.ApplyFilters(s, filters); err != nil {
		return errors.Wrap(err, "ApplyFilters failed")
	}
	if err := yp.YamlParse(s); err != nil {
		return errors.Wrap(err, "YamlParse failed")
	}

	return nil
}

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

func (yp *Yp) ApplyFilters(s string, filters []string) error {
	if _, ok := yp.Files[s]; !ok {
		return errors.WithFields(errors.Fields{
			"File": s,
		}).New("Apply filters failed, no such file loaded")
	}
	out, err := Filter(yp.Files[s], filters)
	if err != nil {
		return err
	}
	yp.Files[s] = out
	return nil
}

func (yp *Yp) applyNullTemplate(name string) error {
	if _, ok := yp.Files[name]; !ok {
		return errors.WithFields(errors.Fields{"Name": name}).New("File has not been imported")
	}

	tf := func(in []byte) ([]byte, error) {
		renderedBytes := bytes.NewBuffer([]byte{})
		tmpl, err := template.New("default").Parse(string(in))
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
		b, err := tf(section.OriginalBytes)
		if err != nil {
			return err
		}
		section.Bytes = b
		section.TemplateFunc = yp.DefaultTemplateFunc
	}
	return nil
}

func (yp *Yp) applyDefaultTemplate(name string, strict bool, vals interface{}) error {
	if _, ok := yp.Files[name]; !ok {
		return errors.WithFields(errors.Fields{"Name": name}).New("File has not been imported")
	}
	for _, section := range yp.Files[name] {
		//run template
		if err := section.Render(vals); err != nil {
			return err
		}
	}
	return nil
}

func (yp *Yp) ApplyDefaultTemplateStrict(name string, vals interface{}) error {
	return yp.applyDefaultTemplate(name, true, vals)
}

func (yp *Yp) ApplyDefaultTemplate(name string, vals interface{}) error {
	return yp.applyDefaultTemplate(name, false, vals)
}

func (yp *Yp) ApplyTemplate(name string, tmplFunc TemplateFunc, vals interface{}) error {
	if _, ok := yp.Files[name]; !ok {
		return errors.WithFields(errors.Fields{"Name": name}).New("File has not been imported")
	}
	for _, section := range yp.Files[name] {
		//run template
		if err := section.RenderWithTemplateFunc(tmplFunc, vals); err != nil {
			return err
		}
	}
	return nil
}
