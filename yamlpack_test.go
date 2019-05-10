package yamlpack

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNew(t *testing.T) {
	Convey("New yamlpack", t, func() {
		yp := New()
		So(yp, ShouldNotBeNil)
		So(yp, ShouldHaveSameTypeAs, &Yp{})
	})
}
