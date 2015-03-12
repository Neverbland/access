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

func (f Fields) ReadFieldPath(path PropertyPath) (interface{}, error) {
	return path.Read(&f.Map)
}

func (f *Fields) WriteFieldPath(path PropertyPath, v interface{}) error {
	return path.Write(&f.Map, v)
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
			assert.NotNil(t, err)
		} else {
			v, err := Read(c.Path, c.Object)
			assert.Equal(t, c.Expected, v)
			assert.Nil(t, err)
		}
	}
}

func TestFieldWrite(t *testing.T) {

	//map

	m := map[string]string{"firstname": ""}

	err := Write("firstname", &m, "Eugeny")
	assert.Nil(t, err)
	assert.Equal(t, "Eugeny", m["firstname"])

	err = Write("lastname", &m, "Tsarykau")
	assert.Nil(t, err)
	assert.Equal(t, "Tsarykau", m["lastname"])

	err = Write("Address_", &m, "Universe")
	assert.Nil(t, err)
	assert.Equal(t, "Universe", m["Address_"])

	//struct
	p := Person{"a", "b", "c"}

	//exported field
	err = Write("firstname", &p, "Eugeny")
	assert.Nil(t, err)
	assert.Equal(t, "Eugeny", p.Firstname)

	//setter
	err = Write("last_name", &p, "Tsarykau")
	assert.Nil(t, err)
	assert.Equal(t, "Tsarykau", p.LastName())

	//FieldWriter
	fields := Fields{map[string]interface{}{"firstname": "hello"}}

	err = Write("firstname", &fields, "Eugeny")
	assert.Nil(t, err)
	assert.Equal(t, "Eugeny", fields.Map["firstname"])

	err = Write("Lastname", &fields, "Tsarykau")
	assert.Nil(t, err)
	assert.Equal(t, "Tsarykau", fields.Map["Lastname"])
}
