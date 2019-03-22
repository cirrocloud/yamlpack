package yamlpack

import (
	"fmt"
	"os"
	"sync"

	errors "github.com/charter-se/structured/errors"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

type YamlPack interface {
	AllSections() []*YamlSection
	ListYamls() []string
	GetString(s string) string
	GetStringSlice(s string) []string
	GetBool(s string) bool
	Sub(s string) *viper.Viper
}

//Yp is a yamlpack instance
type Yp struct {
	sync.RWMutex
	Files    map[string][]*YamlSection
	Handlers map[string]func(string) error
}

type Viper viper.Viper

//New returns a newly created and initialized *Yp
func New() *Yp {
	//Set Case sensiity true
	// this is a global in viper, nothing to be done about it
	viper.SetKeysCaseSensitive(true)
	yp := &Yp{}
	yp.Handlers = make(map[string]func(string) error)
	yp.Files = make(map[string][]*YamlSection)
	return yp
}

//AllSections returns an array containing all yaml sections
func (yp *Yp) AllSections() []*YamlSection {
	yp.RLock()
	defer func() {
		yp.RUnlock()
	}()
	ret := []*YamlSection{}
	for _, f := range yp.Files {
		for _, ys := range f {
			ret = append(ret, ys)
		}
	}
	return ret
}

//ListYamls returns a list of yaml section names as defined by metadata.name
func (yp *Yp) ListYamls() []string {
	ret := []string{}
	for _, ys := range yp.AllSections() {
		ret = append(ret, ys.Viper.Get("metadata.name").(string))
	}
	return ret
}

//RegisterHandler adds a handler to this instance
func (yp *Yp) RegisterHandler(s string, f func(string) error) error {
	yp.Lock()
	defer func() {
		yp.Unlock()
	}()
	if _, exists := yp.Handlers[s]; exists {
		return fmt.Errorf("handler \"%v\" already exists", s)
	}
	yp.Handlers[s] = f
	return nil
}

//DeregisterHandler removed a previously registered handler if it exists
func (yp *Yp) DeregisterHandler(s string) {
	yp.Lock()
	defer func() {
		yp.Unlock()
	}()
	if _, exists := yp.Handlers[s]; exists {
		delete(yp.Handlers, s)
	}
	return
}

func (section *YamlSection) GetString(s string) string {
	return section.Viper.GetString(s)
}

func (section *YamlSection) GetStringSlice(s string) []string {
	return section.Viper.GetStringSlice(s)
}

func (section *YamlSection) GetBool(s string) bool {
	return section.Viper.GetBool(s)
}

func (section *YamlSection) Sub(s string) (*YamlSection, error) {
	v := section.Viper.Sub(s)
	if v == nil {
		return nil, nil
	}
	b, err := yaml.Marshal(v.AllSettings())
	if err != nil {
		return nil, err
	}
	return &YamlSection{
		Bytes: b,
		Viper: v,
	}, nil
}

func (section *YamlSection) AllSettings() (ret map[string]interface{}, err error) {
	ret = make(map[string]interface{})
	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrap(fmt.Errorf("%v", r), "yaml parsing failed")
		}
	}()
	return section.Viper.AllSettings(), nil
}

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
	if err = yaml.UnmarshalStrict(m, entry); err != nil {
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
