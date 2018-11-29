package yamlpack

import (
	"fmt"
	"sync"

	"github.com/spf13/viper"
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

//New returns a newly created and initialized *Yp
func New() *Yp {
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

func (section *YamlSection) Sub(s string) *viper.Viper {
	return section.Viper.Sub(s)
}
