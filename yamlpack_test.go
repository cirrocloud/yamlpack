package yamlpack

import (
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	yp := New()
	if yp == nil {
		t.Errorf("New() *Yp instance was nil")
	}
	if reflect.TypeOf(yp.Handlers) != reflect.TypeOf((map[string]func(string) error)(nil)) {
		t.Errorf("*Yp.Handlers does not contain 'map[string]func(string) error'")
	}
}
