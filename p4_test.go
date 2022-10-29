package p4

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
func runUnmarshall(t *testing.T, testFile string) ([]map[string]string, []error) {
	results := make([]map[string]string, 0)
	errors := []error{}
	fname := path.Join("testdata", testFile)
	buf, err := ioutil.ReadFile(fname)
	if err != nil {
		assert.Fail(t, fmt.Sprintf("Can't read file: %s", fname))
	}
	mbuf := bytes.NewBuffer(buf)
	for {
		r, err := Unmarshal(mbuf)
		if err == io.EOF {
			break
		}
		if err == nil {
			if r == nil {
				// Empty result for the end of the object
				break
			}
			results = append(results, r.(map[string]string))
		} else {
			errors = append(errors, err)
			break
		}
	}
	return results, errors
}
*/
func TestFormatSpec(t *testing.T) {
	spec := map[string]string{"Change": "new",
		"Description": "My line\nSecond line\nThird line\n",
	}
	// Order of lines isn't deterministic, maps don't retain order
	res := formatSpec(spec)
	assert.Regexp(t, regexp.MustCompile("Change: new\n\n"), res)
	assert.Regexp(t, regexp.MustCompile("Description:\n My line\n Second line\n Third line\n\n"), res)

}

type parseErrorTest struct {
	input map[string]string
	want  error
}

var parseErrorTests = []parseErrorTest{
	{
		input: map[string]string{
			"code":     "error",
			"data":     "//fake/depot/... - must refer to client 'HOSTNAME'.",
			"generic":  "2",
			"severity": "3",
		},
		want: errors.New("P4Error -> No such area '//fake/depot/...', please check your path"),
	},
	{
		input: map[string]string{
			"code":     "error",
			"data":     "some unknown error",
			"generic":  "2",
			"severity": "3",
		},
		want: errors.New("P4Error -> some unknown error"),
	},
}

func TestParseError(t *testing.T) {
	for _, tst := range parseErrorTests {
		err := parseError(tst.input)
		assert.Equal(t, tst.want, err)
	}
}

func TestSave(t *testing.T) {
	ds := map[string]string{
		"Job":         "DEV-123",
		"Status":      "open",
		"User":        "a.person",
		"Description": "Desc2",
	}
	p4 := NewP4Params("localhost:1666", "brett", "bb_ws")
	res, err := p4.SaveTxt("job", ds, []string{})
	assert.Nil(t, err)
	fmt.Println(res)
}

func BenchmarkInfo(b *testing.B) {
	p4 := NewP4Params("localhost:1666", "brett", "bb_ws")
	for n := 0; n < b.N; n++ {
		_, err := p4.Run([]string{"info"})
		assert.Nil(b, err)
	}
}
