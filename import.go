package yamlpack

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"text/template"

	errors "github.com/charter-oss/structured/errors"
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

//YamlParse adds viper instances to imported file sections
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

//ImportWithTemplateFuncAndFilters offers a way to import yaml from an io.Reader, applies a template, and filters sections based on a string array
func (yp *Yp) ImportWithTemplateFuncAndFilters(identifier string, r io.Reader, tf TemplateFunc, filters []string) error {
	if err := yp.Import(identifier, r); err != nil {
		return errors.Wrap(err, "Import failed")
	}
	if err := yp.ApplyTemplate(identifier, tf, nil); err != nil {
		return errors.Wrap(err, "ApplyTemplate")
	}
	if err := yp.ApplyFilters(identifier, filters); err != nil {
		return errors.Wrap(err, "ApplyFilters failed")
	}
	if err := yp.YamlParse(identifier); err != nil {
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

//Filter removes *YamlSections from a list based on text filters
func Filter(in []*YamlSection, filters []string) ([]*YamlSection, error) {
	sections := []*YamlSection{}
	for _, section := range in {
		//Each section is a document, needs to be filtered so the consumer gets the sections they want
		// then run the supplied template, then Vipered
		// this avoids running templates on unrelated sections that we may not have the data for

		//run filters
		if !filterMatches(section.OriginalBytes, filters) {
			continue
		}
		//save completed section
		sections = append(sections, section)
	}
	return sections, nil
}

func filterMatches(sectionBytes []byte, filters []string) bool {
	for _, v := range filters {
		rx := regexp.MustCompile(v)
		scanner := bufio.NewScanner(bytes.NewBuffer(sectionBytes))
		for scanner.Scan() {
			if rx.MatchString(scanner.Text()) {
				return true
			}
		}
	}
	return false
}

//ApplyFilters removes *YamlSections from a yamlpack instance based on text filter data
func (yp *Yp) ApplyFilters(s string, filters []string) error {
	sections, ok := yp.Files[s]
	if !ok {
		return errors.WithFields(errors.Fields{
			"File": s,
		}).New("Apply filters failed, no such file loaded")
	}
	out, err := Filter(sections, filters)
	if err != nil {
		return err
	}
	yp.Files[s] = out
	return nil
}

func (yp *Yp) applyNullTemplate(name string) error {
	sections, ok := yp.Files[name]
	if !ok {
		return errors.WithFields(errors.Fields{"Name": name}).New("File has not been imported")
	}

	for _, section := range sections {
		//run template
		b, err := nullTemplate(section.OriginalBytes)
		if err != nil {
			return err
		}
		section.Bytes = b
		section.TemplateFunc = yp.DefaultTemplateFunc
	}
	return nil
}

func nullTemplate(in []byte) ([]byte, error) {
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

func (yp *Yp) applyDefaultTemplate(name string, strict bool, vals interface{}) error {
	sections, ok := yp.Files[name]
	if !ok {
		return errors.WithFields(errors.Fields{"Name": name}).New("File has not been imported")
	}
	for _, section := range sections {
		//run template
		if err := section.Render(vals); err != nil {
			return err
		}
	}
	return nil
}

//ApplyDefaultTemplateStrict runs the default template function and errors on any failure such as missing data
func (yp *Yp) ApplyDefaultTemplateStrict(name string, vals interface{}) error {
	return yp.applyDefaultTemplate(name, true, vals)
}

//ApplyDefaultTemplate runs the default template function but only errors on parse failures
func (yp *Yp) ApplyDefaultTemplate(name string, vals interface{}) error {
	return yp.applyDefaultTemplate(name, false, vals)
}

//ApplyTemplate executes RenderWithTemplateFunc on every section in a yamlpack instance
func (yp *Yp) ApplyTemplate(name string, tmplFunc TemplateFunc, vals interface{}) error {
	sections, ok := yp.Files[name]
	if !ok {
		return errors.WithFields(errors.Fields{"Name": name}).New("File has not been imported")
	}
	for _, section := range sections {
		//run template
		if err := section.RenderWithTemplateFunc(tmplFunc, vals); err != nil {
			return err
		}
	}
	return nil
}
