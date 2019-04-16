package yamlpack

import (
	"fmt"
	"regexp"

	yaml "gopkg.in/yaml.v2"
)

func MapFromBytes(data []byte) (map[string]interface{}, error) {
	ret := make(map[string]interface{})
	err := yaml.Unmarshal(data, ret)
	return sanitize(ret).(map[string]interface{}), err
}

func ExtractDoc(data []byte, criteria map[string]string) (extracted []byte, remaining []byte) {

	rxChunks := regexp.MustCompile(`---`)
	chunks := rxChunks.FindAllIndex(data, -1)
	if len(chunks) == 0 {
		//This is missing the end delimiter ('---') or both,
		//either way we are treating the whole file as 1 section
		chunks = [][]int{[]int{
			int(0), int(len(data)),
		}}
	}

	var filterExp []*regexp.Regexp
	for k, v := range criteria {
		filter := fmt.Sprintf("%v:\\s*\"?%v\"?", k, v)
		filterExp = append(filterExp, regexp.MustCompile(filter))
	}

	extracted = []byte{}
	remaining = []byte{}

	for i := range chunks {
		var b []byte
		if i < len(chunks)-1 {
			b = data[chunks[i][1]:chunks[i+1][0]]
		} else {
			b = data[chunks[i][1]:]
		}
		if func() bool {
			for _, fv := range filterExp {
				if !fv.Match(b) {
					remaining = append(remaining, []byte("---\n")...)
					remaining = append(remaining, b...)
					return false
				}
			}
			return true
		}() {
			extracted = append(extracted, []byte("---\n")...)
			extracted = append(extracted, b...)
		}

	}
	return extracted, remaining
}
