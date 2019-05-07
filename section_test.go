package yamlpack

import (
	"io"
	"strings"
	"testing"

	"github.com/lithammer/dedent"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSection(t *testing.T) {
	Convey("sections from reader", t, func() {
		yp := New()
		err := yp.Import("file1", sectionData())
		So(err, ShouldBeNil)
		Convey("Can export generate *Sections", func() {
			sections := yp.AllSections()
			So(sections, ShouldHaveLength, 2)
			Convey("can read exported sections", func() {
				sectionNumber := sections[0].GetString("SectionNumber")
				So(sectionNumber, ShouldEqual, "1")
			})
			Convey("exported sections have File name", func() {
				So(sections[0].File, ShouldEqual, "file1")
			})
		})
	})
	Convey("sections from file", t, func() {
		yp := New()
		err := yp.ImportFile("testdata/filenameTest.yaml")
		So(err, ShouldBeNil)
		Convey("Can export generate *Sections", func() {
			sections := yp.AllSections()
			So(sections, ShouldHaveLength, 1)
			Convey("exported sections have File name", func() {
				So(sections[0].File, ShouldEqual, "testdata/filenameTest.yaml")
			})
			Convey("section can export subsection", func() {
				subSection, err := sections[0].Sub("subField")
				So(err, ShouldBeNil)
				So(subSection, ShouldNotBeNil)
				Convey("sub sections have File", func() {
					So(subSection.File, ShouldEqual, "testdata/filenameTest.yaml")
				})
			})
		})
	})

}

func sectionData() io.Reader {
	return strings.NewReader(dedent.Dedent(`
		---
		SectionNumber: 1
		---
		SectionNumber: 2
	`))
}
