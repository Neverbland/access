package access

import (
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

func (i Indexes) Index(index int) (interface{}, error) {
	return Read(index, &i.Slice)
}

func (i *Indexes) SetIndex(index int, v interface{}) error {
	return Write(index, &i.Slice, v)
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

		if c.Invalid {
			_, err := Read(c.Path, c.Object)
			assert.NotNil(t, err)
		} else {
			v, err := Read(c.Path, c.Object)
			assert.Equal(t, c.Expected, v)
			assert.Nil(t, err)
		}
	}
}

func TestIndexWrite(t *testing.T) {

	assert := assert.New(t)

	//array
	p := [3]string{"Eugeny", "Tsarykau", "Universe"}

	assert.NoError(Write("[0]", &p, "Aleksandra"))
	assert.Equal("Aleksandra", p[0])

	//slice
	s := p[:]

	assert.NoError(Write(2, &s, "World"))
	assert.Equal("World", s[2])

	assert.NoError(Write(10, &s, "New"))
	assert.Equal("New", s[10])

	//IndexWriter
	ir := Indexes{s}

	assert.NoError(Write(1, &ir, "Yudina"))
	assert.Equal("Yudina", ir.Slice[1])

}
