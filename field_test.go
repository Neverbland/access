package access

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type FieldReadAssertion struct {
	Object   interface{}
	Expected interface{}
	Path     string
	Invalid  bool
}

type Person struct {
	Firstname string
	lastname  string
	address   string
	Name      *string
	Name2     ***string
}

func (p Person) LastName() string { return p.lastname }

func (p *Person) SetLastName(lastname string) { p.lastname = lastname }

func (p *Person) GetAddress() string { return p.address }

func (p Person) GetAddressForCountry(_ string) string { return "" }

type Fields struct {
	Map map[string]interface{}
}

func (f Fields) Field(field string) (interface{}, error) {
	return Read(field, f.Map)
}

func (f *Fields) SetField(field string, v interface{}) error {
	return Write(field, &f.Map, v)
}

func TestFieldRead(t *testing.T) {
	p := Person{"foo", "boo", "baz", nil, nil}

	//struct
	cases := []FieldReadAssertion{
		//struct value
		FieldReadAssertion{p, "foo", "firstname", false},
		FieldReadAssertion{p, "foo", "Firstname", false},
		FieldReadAssertion{p, "boo", "last_name", false},
		FieldReadAssertion{p, "boo", "lastName", false},
		FieldReadAssertion{p, "boo", "LastName", false},
		FieldReadAssertion{p, (*string)(nil), "Name", false},

		//invalid
		FieldReadAssertion{p, nil, "first_name", true},
		FieldReadAssertion{p, nil, "there_is_no_field_like_this", true},
		FieldReadAssertion{p, nil, "get_address_for_country", true},
		FieldReadAssertion{p, nil, "getAddressForCountry", true},

		//struct pointer
		FieldReadAssertion{&p, "foo", "firstname", false},
		FieldReadAssertion{&p, "foo", "Firstname", false},
		FieldReadAssertion{&p, "boo", "last_name", false},
		FieldReadAssertion{&p, "boo", "lastName", false},
		FieldReadAssertion{&p, "boo", "LastName", false},
		FieldReadAssertion{&p, "baz", "address", false},
		FieldReadAssertion{&p, "baz", "Address", false},
		FieldReadAssertion{&p, "baz", "getAddress", false},
		FieldReadAssertion{&p, "baz", "get_Address", false},
		FieldReadAssertion{&p, "baz", "GetAddress", false},
		FieldReadAssertion{p, (*string)(nil), "Name", false},

		//invalid
		FieldReadAssertion{&p, nil, "first_name", true},
		FieldReadAssertion{&p, nil, "there_is_no_field_like_this", true},
		//getter with non zero parameters count is invalid
		FieldReadAssertion{&p, nil, "get_address_for_country", true},
		FieldReadAssertion{&p, nil, "getAddressForCountry", true},
	}

	m := map[string]string{"firstname": "foo", "lastName": "boo", "address": "baz"}

	m2 := map[int]string{0: "foo"}

	//map
	cases = append(
		cases,
		FieldReadAssertion{m, "foo", "firstname", false},
		FieldReadAssertion{m, "boo", "lastName", false},
		FieldReadAssertion{m, "baz", "address", false},
		//exact key match required
		FieldReadAssertion{m, "foo", "first_name", true},
		FieldReadAssertion{m, "boo", "lastname", true},
		FieldReadAssertion{m, "baz", "Address", true},

		//map key must be a string
		FieldReadAssertion{m2, "foo", "first_name", true},
	)

	fields := Fields{map[string]interface{}{"firstname": "foo", "lastName": "boo", "address": "baz"}}

	//FieldReader
	cases = append(
		cases,
		FieldReadAssertion{fields, "foo", "firstname", false},
		FieldReadAssertion{fields, "boo", "lastName", false},
		FieldReadAssertion{fields, "baz", "Address", true},
	)

	for _, c := range cases {

		if c.Invalid {
			_, err := Read(c.Path, c.Object)
			assert.Error(t, err)
		} else {
			v, err := Read(c.Path, c.Object)
			assert.NoError(t, err)
			assert.Equal(t, c.Expected, v)
		}
	}
}

func TestFieldWrite(t *testing.T) {

	assert := assert.New(t)
	//map

	m := map[string]string{"firstname": ""}

	assert.NoError(Write("firstname", &m, "foo"))
	assert.Equal("foo", m["firstname"])

	assert.NoError(Write("lastname", &m, "boo"))
	assert.Equal("boo", m["lastname"])

	assert.NoError(Write("Address_", &m, "baz"))
	assert.Equal("baz", m["Address_"])

	//struct
	p := Person{"a", "b", "c", nil, nil}

	//exported field
	assert.NoError(Write("firstname", &p, "foo"))
	assert.Equal("foo", p.Firstname)

	//string pointer type

	strval := "test"

	assert.NoError(Write("name", &p, strval))
	assert.Equal(strval, *(p.Name))

	assert.NoError(Write("name", &p, &strval))
	assert.Equal(&strval, p.Name)

	assert.NoError(Write("name2", &p, strval))
	assert.Equal(strval, ***(p.Name2))

	//setter
	assert.NoError(Write("last_name", &p, "boo"))
	assert.Equal("boo", p.LastName())

	//FieldWriter
	fields := Fields{map[string]interface{}{"firstname": "hello"}}

	assert.NoError(Write("firstname", &fields, "foo"))
	assert.Equal("foo", fields.Map["firstname"])

	assert.NoError(Write("Lastname", &fields, "boo"))
	assert.Equal("boo", fields.Map["Lastname"])

	var anything interface{}

	assert.NoError(Write("field", &anything, "value"))
	assert.Equal(map[string]interface{}{"field": "value"}, anything)

	assert.NoError(Write("field2.field21", &anything, "value"))
	assert.Equal(map[string]interface{}{"field": "value", "field2": map[string]interface{}{"field21": "value"}}, anything)

	assert.NoError(Write("f3", &anything, &strval))
	assert.Equal(&strval, MustRead("f3", anything))

	assert.NoError(Write("f3", &anything, "test"))
	assert.Equal("test", MustRead("f3", anything))
}
