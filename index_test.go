package access

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

type IndexReadAssertion struct {
	Object   interface{}
	Expected interface{}
	Path     int
	Invalid  bool
}

type Indexes struct {
	Slice []string
}

func (i Indexes) ReadIndexPath(path PropertyPath) (interface{}, error) {
	return path.Read(&i.Slice)
}

func (i *Indexes) WriteIndexPath(path PropertyPath, v interface{}) error {
	return path.Write(&i.Slice, v)
}

func TestIndexRead(t *testing.T) {
	p := [3]string{"Eugeny", "Tsarykau", "Universe"}

	//array
	cases := []IndexReadAssertion{
		IndexReadAssertion{p, "Eugeny", 0, false},
		IndexReadAssertion{p, "Tsarykau", 1, false},
		IndexReadAssertion{p, "Universe", 2, false},

		//pointer
		IndexReadAssertion{&p, "Eugeny", 0, false},
		IndexReadAssertion{&p, "Tsarykau", 1, false},
		IndexReadAssertion{&p, "Universe", 2, false},

		//out of range
		IndexReadAssertion{p, nil, 3, true},
		IndexReadAssertion{&p, nil, 3, true},
	}

	s := p[:]

	//slice
	cases = append(
		cases,
		IndexReadAssertion{s, "Eugeny", 0, false},
		IndexReadAssertion{s, "Tsarykau", 1, false},
		IndexReadAssertion{s, "Universe", 2, false},

		//pointer
		IndexReadAssertion{&s, "Eugeny", 0, false},
		IndexReadAssertion{&s, "Tsarykau", 1, false},
		IndexReadAssertion{&s, "Universe", 2, false},
		//out of range
		IndexReadAssertion{s, nil, 3, true},
		IndexReadAssertion{&s, nil, 3, true},
	)

	ir := Indexes{s}

	//IndexReader
	cases = append(
		cases,
		IndexReadAssertion{ir, "Eugeny", 0, false},
		IndexReadAssertion{ir, "Tsarykau", 1, false},
		IndexReadAssertion{ir, "Universe", 2, false},

		//pointer
		IndexReadAssertion{&ir, "Eugeny", 0, false},
		IndexReadAssertion{&ir, "Tsarykau", 1, false},
		IndexReadAssertion{&ir, "Universe", 2, false},
		//out of range
		IndexReadAssertion{ir, nil, 3, true},
		IndexReadAssertion{&ir, nil, 3, true},
	)

	for _, c := range cases {

		selector := fmt.Sprintf("[%d]", c.Path)

		if c.Invalid {
			_, err := Read(selector, c.Object)
			assert.NotNil(t, err)
		} else {
			v, err := Read(selector, c.Object)
			assert.Equal(t, c.Expected, v)
			assert.Nil(t, err)
		}
	}
}

func TestIndexWrite(t *testing.T) {

	//array
	p := [3]string{"Eugeny", "Tsarykau", "Universe"}

	err := Write("[0]", &p, "Aleksandra")
	assert.Nil(t, err)
	assert.Equal(t, "Aleksandra", p[0])

	//slice
	s := p[:]

	err = Write("[2]", &s, "World")
	assert.Nil(t, err)
	assert.Equal(t, "World", s[2])

	//IndexWriter
	ir := Indexes{s}

	err = Write("[1]", &ir, "Yudina")
	assert.Nil(t, err)
	assert.Equal(t, "Yudina", ir.Slice[1])

}
