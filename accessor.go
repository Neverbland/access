package access

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

const pathRegexStr = `(?i)(?P<field>\w+)?(?P<index>\[\d+\])?(?P<dot>\.?)`

var pathRegex = regexp.MustCompile(pathRegexStr)

var (
	pathReaderInterface = reflect.TypeOf((*PathReader)(nil)).Elem()
	pathWriterInterface = reflect.TypeOf((*PathWriter)(nil)).Elem()
)

func New(v interface{}) Path {

	var str string

	if s, ok := v.(fmt.Stringer); ok {
		str = s.String()
	} else {
		switch s := v.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			str = fmt.Sprintf("[%d]", v)
		case string:
			str = s
		default:
			panic(fmt.Sprintf("Can't build path from %#v", v))
		}
	}
	str = strings.TrimSpace(str)
	selector := str

	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Sprintf("Parsing error `%s` at `%s`: %s ", str, selector, r))
		}
	}()

	if len(selector) == 0 {
		return Path{}
	}

	parts := []interface{}{}

	var (
		dot           bool
		field         string
		index         string
		fieldExpected bool
	)

	if selector[0:1] != "[" {
		fieldExpected = true
	}

	for len(selector) > 0 {

		match := pathRegex.FindStringSubmatch(selector)
		if len(match) == 0 {
			panic("Doesn't match selector")
		}

		dot = false
		field = ""
		index = ""

		for i, name := range pathRegex.SubexpNames() {

			if i == 0 || match[i] == "" {
				continue
			}

			if name == "field" {
				field = match[i]
				continue
			}

			if name == "index" {
				index = match[i][1 : len(match[i])-1] // get value from [%v]
				continue
			}

			if name == "dot" {
				dot = true
				continue
			}
		}

		if field == "" && fieldExpected {
			panic("field expected")
		}

		if field != "" && !fieldExpected {
			panic("field not expected")
		}

		if field == "" && index == "" {
			panic("field or index expected")
		}

		if field != "" {
			parts = append(parts, field)
		}

		if index != "" {
			ni, err := strconv.Atoi(index)
			if err != nil {
				panic("numeric index expected")
			}
			parts = append(parts, ni)
		}

		selector = strings.Replace(selector, match[0], "", 1)

		fieldExpected = false

		if dot {
			if len(selector) == 0 {
				panic("Unexpected dot")
			}

			fieldExpected = true
		}

	}

	return Path(parts)
}

type Path []interface{}

func (p Path) String() string {
	path := ""
	for i, accessor := range p {
		switch s := accessor.(type) {
		case string:
			if i != 0 {
				path += "." + s
			} else {
				path += s
			}

		case int:
			path += fmt.Sprintf("[%d]", s)
		}
	}

	return path
}

func (p Path) write(v reflect.Value, w reflect.Value, wt reflect.Type) (err error) {

	if !v.CanAddr() {
		return Error{fmt.Errorf("Got unadressable value"), []interface{}{}}
	}

	if len(p) == 0 {
		return indirectWrite(v, w,wt)
	}

	if writer, ok := indirectRead(v, pathWriterInterface).Interface().(PathWriter); ok {
		return writer.WritePath(p, w.Interface())
	}

	var rpath *Path

	if len(p) > 1 {
		path := p[1:]
		rpath = &path
	}

	switch s := p[0].(type) {
	case string:
		err = writeField(v, s, rpath, w, wt)
	case int:
		err = writeIndex(v, s, rpath, w, wt)
	}

	if err != nil {
		e, ok := err.(Error)

		if !ok {
			err = Error{err, []interface{}{}}
		} else {
			err = e.back(p[0])
		}
	}

	return err
}

func (p Path) read(v reflect.Value) (rv reflect.Value, err error) {

	if len(p) == 0 {
		return v, nil
	}

	if reader, ok := indirectRead(v, pathReaderInterface).Interface().(PathReader); ok {
		val, err := reader.ReadPath(p)
		return reflect.ValueOf(val), err
	}

	var rpath *Path

	if len(p) > 1 {
		path := p[1:]
		rpath = &path
	}

	switch s := p[0].(type) {
	case string:
		rv, err = readField(v, s, rpath)
	case int:
		rv, err = readIndex(v, s, rpath)
	}

	if err != nil {
		e, ok := err.(Error)

		if !ok {
			err = Error{err, []interface{}{}}
		} else {
			err = e.back(p[0])
		}
	}

	return rv, err
}

func (path Path) Write(v interface{}, w interface{}) error {

	rv := reflect.ValueOf(v)

	if rv.Kind() != reflect.Ptr {
		return Error{fmt.Errorf("Non pointer value"), []interface{}{}}
	}

	return path.write(rv.Elem(), reflect.ValueOf(w), reflect.TypeOf(w))
}

func (path Path) Read(v interface{}) (interface{}, error) {

	rv := reflect.ValueOf(v)

	re, err := path.read(rv)

	if err != nil || !re.IsValid() {
		return nil, err
	}

	return re.Interface(), err
}

func (path Path) MustRead(v interface{}, dv ...interface{}) (value interface{}) {
	var dval interface{}
	if len(dv) == 1 {
		dval = dv[0]
	}

	defer func() {
		if r := recover(); r != nil {
			value = dval
		}
	}()

	t, err := path.Read(v)

	if err != nil || (t == nil && dval != nil) {
		return dval
	}

	return t
}

type Error struct {
	error
	Path []interface{}
}

func (e Error) Error() string {
	return fmt.Sprintf("%s at `%s`", e.error, Path(e.Path))
}

func (e Error) back(s interface{}) Error {
	e.Path = append([]interface{}{s}, e.Path...)
	return e
}

type PathReader interface {
	ReadPath(Path) (interface{}, error)
}

type PathWriter interface {
	PathReader
	WritePath(Path, interface{}) error
}

func Write(s interface{}, v interface{}, val interface{}) error {
	return New(s).Write(v, val)
}

func Read(s interface{}, v interface{}) (interface{}, error) {
	return New(s).Read(v)
}

func MustRead(s interface{}, v interface{}, dv ...interface{}) (value interface{}) {
	return New(s).MustRead(v, dv...)
}

//set value allocating pointers if needed
func indirectWrite(v reflect.Value, w reflect.Value,wt reflect.Type) (err error) {

	nilValue:=(wt == nil)

	for {

		if !nilValue && v.CanSet() && wt.AssignableTo(v.Type()) {
			v.Set(w)
			return nil
		}

		// Load value from interface if value inside is a pointer
		if v.Kind() == reflect.Interface && !v.IsNil() {

			e := v.Elem()

			if e.Kind() == reflect.Ptr && !e.IsNil() {
				v = e
				continue
			}
		}

		if v.Kind() != reflect.Ptr {
			break
		}

		if v.Elem().Kind() != reflect.Ptr && nilValue && v.CanSet() {
			break
		}

		if v.IsNil() {
			nv := reflect.New(v.Type().Elem())

			defer func(v, nv reflect.Value) {
				if err == nil {
					v.Set(nv)
				}
			}(v, nv)

			v = nv
		}

 		v = v.Elem()
	}


	if !v.CanSet() {
		return fmt.Errorf("got value that couldn't be changed")
	}

	if nilValue {
		switch v.Kind() {
		case reflect.Interface, reflect.Ptr, reflect.Map, reflect.Slice:
			v.Set(reflect.Zero(v.Type()))
			return nil
		}
		return fmt.Errorf("value must be nullable like interface,pointer,map or slice")
	}

	if !wt.AssignableTo(v.Type()) {
		return fmt.Errorf("can't assign")
	}

	v.Set(w)
	return nil
}

func indirectRead(v reflect.Value, accessor reflect.Type) reflect.Value {

	// If v is not a pointer and is addressable,
	// start with its address, so that if the type has pointer methods,
	// we find them.
	if v.Kind() != reflect.Ptr && v.CanAddr() {
		v = v.Addr()
	}

	for {

		if accessor != nil && v.Type().NumMethod() > 0 && v.Type().Implements(accessor) {
			break
		}

		// Load value from interface if value inside is a pointer
		if v.Kind() == reflect.Interface && !v.IsNil() {

			e := v.Elem()

			if e.Kind() == reflect.Ptr && !e.IsNil() {
				v = e
				continue
			}
		}

		if v.Kind() != reflect.Ptr {
			break
		}

		v = v.Elem()
	}

	return v
}

//create new pointer for value, so that it became addressable
func allocateNew(v reflect.Value) reflect.Value {
	c := reflect.New(v.Type())
	c.Elem().Set(v)
	return c.Elem()
}
