package yamlpack

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"text/template"

	"github.com/Masterminds/sprig"
	errors "github.com/charter-oss/structured/errors"
	"github.com/ghodss/yaml"
	"github.com/spf13/viper"
)

//TemplateFunc is processed with RenderWithTemplateFunc
type TemplateFunc func([]byte, interface{}) ([]byte, error)

//YamlPack provides a set of functionality to process composite yaml documents
type YamlPack interface {
	AllSections() []*YamlSection
	ListYamls() []string
	GetString(string) string
	GetStringSlice(string) []string
	GetBool(string) bool
	Sub(string) *viper.Viper
}

//Yp is a yamlpack instance
type Yp struct {
	sync.RWMutex
	Files               map[string][]*YamlSection
	Handlers            map[string]func(string) error
	DefaultTemplateFunc TemplateFunc
}

//Viper is an alias of viper.Viper (github.com/spf13/viper)
type Viper viper.Viper

func init() {
	viper.SetKeysCaseSensitive(true)
}

//New returns a newly created and initialized *Yp
func New() *Yp {
	//Set Case sensiity true
	// this is a global in viper, nothing to be done about it
	yp := &Yp{}
	yp.Handlers = make(map[string]func(string) error)
	yp.Files = make(map[string][]*YamlSection)
	yp.DefaultTemplateFunc = defaultTemplate
	return yp
}

func (yp *Yp) newSection() *YamlSection {
	section := &YamlSection{}
	section.TemplateFunc = yp.DefaultTemplateFunc
	return section
}

//AllSections returns an array containing all yaml sections
func (yp *Yp) AllSections() []*YamlSection {
	yp.RLock()
	defer func() {
		yp.RUnlock()
	}()
	outSections := []*YamlSection{}
	for _, sections := range yp.Files {
		for _, ys := range sections {
			outSections = append(outSections, ys)
		}
	}
	return outSections
}

//ListYamls returns a list of yaml section names as defined by metadata.name
func (yp *Yp) ListYamls() []string {
	list := []string{}
	for _, ys := range yp.AllSections() {
		list = append(list, ys.Viper.Get("metadata.name").(string))
	}
	return list
}

//RegisterHandler adds a handler to this instance
func (yp *Yp) RegisterHandler(name string, f func(string) error) error {
	yp.Lock()
	defer func() {
		yp.Unlock()
	}()
	if _, exists := yp.Handlers[name]; exists {
		return fmt.Errorf("handler \"%v\" already exists", name)
	}
	yp.Handlers[name] = f
	return nil
}

//DeregisterHandler removed a previously registered handler if it exists
func (yp *Yp) DeregisterHandler(name string) {
	yp.Lock()
	defer func() {
		yp.Unlock()
	}()
	if _, exists := yp.Handlers[name]; exists {
		delete(yp.Handlers, name)
	}
	return
}

//GetString returns a string value from a doted notation key
func (section *YamlSection) GetString(identifier string) string {
	return section.Viper.GetString(identifier)
}

//GetStringSlice returns a string slice from a doted notation key
func (section *YamlSection) GetStringSlice(identifier string) []string {
	return section.Viper.GetStringSlice(identifier)
}

//GetBool returns a boolean value from a doted notation key
func (section *YamlSection) GetBool(identifier string) bool {
	return section.Viper.GetBool(identifier)
}

//Sub returns a *YamlSection instance from a yaml key identified by doted notation
func (section *YamlSection) Sub(identifier string) (*YamlSection, error) {
	viperSub := section.Viper.Sub(identifier)
	if viperSub == nil {
		return nil, nil
	}
	marshaledBytes, err := yaml.Marshal(viperSub.AllSettings())
	if err != nil {
		return nil, err
	}
	return &YamlSection{
		Bytes:        marshaledBytes,
		Viper:        viperSub,
		TemplateFunc: section.TemplateFunc,
	}, nil
}

//Render applies the sections configured template with the provided values
func (section *YamlSection) Render(vals interface{}) error {
	return section.RenderWithTemplateFunc(section.TemplateFunc, vals)
}

//Render applies the provided template function to the *YamlSection with the provided values
func (section *YamlSection) RenderWithTemplateFunc(tmplFunc TemplateFunc, vals interface{}) error {

	out, err := runTemplate(section.OriginalBytes, tmplFunc, vals)
	if err != nil {
		return err
	}
	section.Bytes = out
	//add viper
	vp := viper.New()
	vp.SetConfigType("yaml")
	if err := vp.ReadConfig(bytes.NewBuffer(section.Bytes)); err != nil {
		return errors.WithFields(errors.Fields{
			"Data": section.Bytes,
		}).Wrap(err, "failed to parse yaml section")
	}
	section.Viper = vp
	return nil
}

//AllSettings returns a value map derived from the *YamlSection data
func (section *YamlSection) AllSettings() (ret map[string]interface{}, err error) {
	ret = make(map[string]interface{})
	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrap(fmt.Errorf("%v", r), "yaml parsing failed")
		}
	}()
	return section.Viper.AllSettings(), nil
}

//Unmarshal processes *YamlSection data into the provided data structure
//Missing values or destination structure elements are ignored
func (section *YamlSection) Unmarshal(entry interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrap(fmt.Errorf("%v", r), "yaml unmarshal failed")
		}
	}()
	m, err := yaml.Marshal(sanitize(section.Viper.AllSettings()))
	if err != nil {
		err = errors.Wrap(err, "yaml intermediate marshal failed")
		return err
	}
	if err = yaml.Unmarshal(m, entry); err != nil {
		err = errors.Wrap(err, "yaml unarshal strict failed")
		return err
	}
	return nil
}

//UnmarshalStrict processes *YamlSection data into the provided data structure
//This is the strict version which has DisallowUnknownFields enabled
func (section *YamlSection) UnmarshalStrict(entry interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrap(fmt.Errorf("%v", r), "yaml unmarshal failed")
		}
	}()
	m, err := yaml.Marshal(sanitize(section.Viper.AllSettings()))
	if err != nil {
		err = errors.Wrap(err, "yaml intermediate marshal failed")
		return err
	}
	if err = yaml.Unmarshal(m, entry, yaml.DisallowUnknownFields); err != nil {
		err = errors.Wrap(err, "yaml unarshal strict failed")
		return err
	}
	return nil
}

//sanitize converts map[interface{}]interface{} to map[string]interface{}
func sanitize(input interface{}) interface{} {
	switch input.(type) {
	case map[interface{}]interface{}:
		output := make(map[string]interface{})
		for k, v := range input.(map[interface{}]interface{}) {
			switch k.(type) {
			case string:
				output[k.(string)] = sanitize(v)
			default:
				fmt.Printf("sanitize: Got unhandled inner type: %T\n", input)
			}
		}
		return output
	case map[string]interface{}:
		output := make(map[string]interface{})
		for k, v := range input.(map[string]interface{}) {
			output[k] = sanitize(v)
		}
		return output
	case []interface{}:
		output := []interface{}{}
		for _, v := range input.([]interface{}) {
			val := sanitize(v)
			output = append(output, val)
		}
		return output
	case string, []string, int, []int, bool, []bool, interface{}, nil:
		return input
	default:
		os.Stderr.WriteString(fmt.Sprintf("\t-------->>>>Got type %T\n", input))
		return input
	}
	return nil
}

func defaultTemplate(in []byte, val interface{}) ([]byte, error) {
	renderedBytes := bytes.NewBuffer([]byte{})
	tmpl, err := template.New("default").Parse(string(in))
	if err != nil {
		return nil, err
	}
	if err := tmpl.Funcs(sprig.TxtFuncMap()).Execute(renderedBytes, val); err != nil {
		fmt.Printf("---\n")
		fmt.Printf("%v\n", renderedBytes.String())
		fmt.Printf("---\n")
		return nil, errors.Wrap(err, "Failed to render template")
	}
	return renderedBytes.Bytes(), nil
}

func runTemplate(in []byte, tmplFunc TemplateFunc, vals interface{}) ([]byte, error) {
	return tmplFunc(in, vals)
}
