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
	p := [3]string{"foo", "bar", "baz"}

	//array
	cases := []IndexReadAssertion{
		IndexReadAssertion{p, "foo", 0, false},
		IndexReadAssertion{p, "bar", 1, false},
		IndexReadAssertion{p, "baz", 2, false},

		//pointer
		IndexReadAssertion{&p, "foo", 0, false},
		IndexReadAssertion{&p, "bar", 1, false},
		IndexReadAssertion{&p, "baz", 2, false},

		//out of range
		IndexReadAssertion{p, nil, 3, true},
		IndexReadAssertion{&p, nil, 3, true},
	}

	s := p[:]

	//slice
	cases = append(
		cases,
		IndexReadAssertion{s, "foo", 0, false},
		IndexReadAssertion{s, "bar", 1, false},
		IndexReadAssertion{s, "baz", 2, false},

		//pointer
		IndexReadAssertion{&s, "foo", 0, false},
		IndexReadAssertion{&s, "bar", 1, false},
		IndexReadAssertion{&s, "baz", 2, false},
		//out of range
		IndexReadAssertion{s, nil, 3, true},
		IndexReadAssertion{&s, nil, 3, true},
	)

	ir := Indexes{s}

	//IndexReader
	cases = append(
		cases,
		IndexReadAssertion{ir, "foo", 0, false},
		IndexReadAssertion{ir, "bar", 1, false},
		IndexReadAssertion{ir, "baz", 2, false},

		//pointer
		IndexReadAssertion{&ir, "foo", 0, false},
		IndexReadAssertion{&ir, "bar", 1, false},
		IndexReadAssertion{&ir, "baz", 2, false},
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
	p := [3]string{"foo", "bar", "baz"}

	assert.NoError(Write("[0]", &p, "qux"))
	assert.Equal("qux", p[0])

	//slice
	s := p[:]

	assert.NoError(Write(2, &s, "foo"))
	assert.Equal("foo", s[2])

	assert.NoError(Write(10, &s, "New"))
	assert.Equal("New", s[10])

	//pointers
	refs := make([]*string, 2)

	assert.NoError(Write(0, &refs, "test"))
	assert.Equal("test", *(refs[0]))

	strval := "test2"

	assert.NoError(Write(1, &refs, &strval))
	assert.Equal(&strval, refs[1])

	//IndexWriter
	ir := Indexes{s}

	assert.NoError(Write(1, &ir, "bar"))
	assert.Equal("bar", ir.Slice[1])

	var anything interface{}

	assert.NoError(Write(2, &anything, "bar"))
	assert.Equal("bar", MustRead(2, anything))

	assert.NoError(Write("[3].name", &anything, "foo"))
	assert.Equal("foo", MustRead("[3].name", anything))

	assert.NoError(Write(0, &anything, &strval))
	assert.Equal(&strval, MustRead(0, anything))

	assert.NoError(Write(0, &anything, "test"))
	assert.Equal("test", MustRead(0, anything))

}
