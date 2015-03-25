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
	p := Person{"Eugeny", "Tsarykau", "Universe"}

	//struct
	cases := []FieldReadAssertion{
		//struct value
		FieldReadAssertion{p, "Eugeny", "firstname", false},
		FieldReadAssertion{p, "Eugeny", "Firstname", false},
		FieldReadAssertion{p, "Tsarykau", "last_name", false},
		FieldReadAssertion{p, "Tsarykau", "lastName", false},
		FieldReadAssertion{p, "Tsarykau", "LastName", false},

		//invalid
		FieldReadAssertion{p, nil, "first_name", true},
		FieldReadAssertion{p, nil, "there_is_no_field_like_this", true},
		FieldReadAssertion{p, nil, "get_address_for_country", true},
		FieldReadAssertion{p, nil, "getAddressForCountry", true},

		//struct pointer
		FieldReadAssertion{&p, "Eugeny", "firstname", false},
		FieldReadAssertion{&p, "Eugeny", "Firstname", false},
		FieldReadAssertion{&p, "Tsarykau", "last_name", false},
		FieldReadAssertion{&p, "Tsarykau", "lastName", false},
		FieldReadAssertion{&p, "Tsarykau", "LastName", false},
		FieldReadAssertion{&p, "Universe", "address", false},
		FieldReadAssertion{&p, "Universe", "Address", false},
		FieldReadAssertion{&p, "Universe", "getAddress", false},
		FieldReadAssertion{&p, "Universe", "get_Address", false},
		FieldReadAssertion{&p, "Universe", "GetAddress", false},
		//invalid
		FieldReadAssertion{&p, nil, "first_name", true},
		FieldReadAssertion{&p, nil, "there_is_no_field_like_this", true},
		//getter with non zero parameters count is invalid
		FieldReadAssertion{&p, nil, "get_address_for_country", true},
		FieldReadAssertion{&p, nil, "getAddressForCountry", true},
	}

	m := map[string]string{"firstname": "Eugeny", "lastName": "Tsarykau", "address": "Universe"}

	m2 := map[int]string{0: "Eugeny"}

	//map
	cases = append(
		cases,
		FieldReadAssertion{m, "Eugeny", "firstname", false},
		FieldReadAssertion{m, "Tsarykau", "lastName", false},
		FieldReadAssertion{m, "Universe", "address", false},
		//exact key match required
		FieldReadAssertion{m, "Eugeny", "first_name", true},
		FieldReadAssertion{m, "Tsarykau", "lastname", true},
		FieldReadAssertion{m, "Universe", "Address", true},

		//map key must be a string
		FieldReadAssertion{m2, "Eugeny", "first_name", true},
	)

	fields := Fields{map[string]interface{}{"firstname": "Eugeny", "lastName": "Tsarykau", "address": "Universe"}}

	//FieldReader
	cases = append(
		cases,
		FieldReadAssertion{fields, "Eugeny", "firstname", false},
		FieldReadAssertion{fields, "Tsarykau", "lastName", false},
		FieldReadAssertion{fields, "Universe", "Address", true},
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

	assert.NoError(Write("firstname", &m, "Eugeny"))
	assert.Equal("Eugeny", m["firstname"])

	assert.NoError(Write("lastname", &m, "Tsarykau"))
	assert.Equal("Tsarykau", m["lastname"])

	assert.NoError(Write("Address_", &m, "Universe"))
	assert.Equal("Universe", m["Address_"])

	//struct
	p := Person{"a", "b", "c"}

	//exported field
	assert.NoError(Write("firstname", &p, "Eugeny"))
	assert.Equal("Eugeny", p.Firstname)

	//setter
	assert.NoError(Write("last_name", &p, "Tsarykau"))
	assert.Equal("Tsarykau", p.LastName())

	//FieldWriter
	fields := Fields{map[string]interface{}{"firstname": "hello"}}

	assert.NoError(Write("firstname", &fields, "Eugeny"))
	assert.Equal("Eugeny", fields.Map["firstname"])

	assert.NoError(Write("Lastname", &fields, "Tsarykau"))
	assert.Equal("Tsarykau", fields.Map["Lastname"])

	var anything interface{}

	assert.NoError(Write("field", &anything, "value"))
	assert.Equal(map[string]interface{}{"field": "value"}, anything)

	assert.NoError(Write("field2.field21", &anything, "value"))
	assert.Equal(map[string]interface{}{"field": "value", "field2": map[string]interface{}{"field21": "value"}}, anything)
}
