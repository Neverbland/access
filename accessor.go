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
	indexReaderInterface reflect.Type
	indexWriterInterface reflect.Type
	fieldReaderInterface reflect.Type
	fieldWriterInterface reflect.Type
)

func init() {
	indexReaderInterface = reflect.TypeOf((*IndexPathReader)(nil)).Elem()
	indexWriterInterface = reflect.TypeOf((*IndexPathWriter)(nil)).Elem()
	fieldReaderInterface = reflect.TypeOf((*FieldPathReader)(nil)).Elem()
	fieldWriterInterface = reflect.TypeOf((*FieldPathWriter)(nil)).Elem()
}

func New(s string) PropertyPath {

	selector := s

	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Sprintf("Parsing error `%s` at `%s`: %s ", s, selector, r))
		}
	}()

	selector = strings.TrimSpace(selector)

	if len(selector) == 0 {
		panic("empty selector")
	}

	parts := []PropertyAccessor{}

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
			parts = append(parts, Field(field))
		}

		if index != "" {
			ni, err := strconv.Atoi(index)
			if err != nil {
				panic("numeric index expected")
			}
			parts = append(parts, Index(ni))
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

	return PropertyPath(parts)
}

type PropertyPath []PropertyAccessor

func (p PropertyPath) String() string {
	path := ""
	for i, accessor := range p {
		part := accessor.String()
		if _, ok := accessor.(Field); ok && i != 0 {
			part = "." + part
		}
		path += part
	}

	return path
}

func (path PropertyPath) Write(v interface{}, val interface{}) error {

	rv := reflect.ValueOf(v)

	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		panic("Non-pointer variable")
	}

	last := len(path) - 1

	for i, reader := range path {

		if _, ok := reader.(Index); ok {
			rv = indirect(rv, indexWriterInterface)
			if iw, ok := rv.Interface().(IndexPathWriter); ok {
				return iw.WriteIndexPath(path[i:], val)
			}
		}

		if _, ok := reader.(Field); ok {
			rv = indirect(rv, fieldWriterInterface)
			if fw, ok := rv.Interface().(FieldPathWriter); ok {
				return fw.WriteFieldPath(path[i:], val)
			}
		}

		if i == last {
			continue
		}

		if r, err := reader.Read(rv); err != nil {
			return fmt.Errorf("[%T]Read error at `%s`: %s", reader, PropertyPath(path[i:]), err)
		} else {
			if !r.IsValid() {
				return fmt.Errorf("Got invalid value at `%s`: %s", PropertyPath(path[i:]), r)
			}
			if !r.CanAddr() && r.Kind() != reflect.Ptr {
				return fmt.Errorf("Got unadressable value at `%s`: %s", PropertyPath(path[i:]), r)
			}
			rv = r
		}
	}

	writer := path[last]

	if err := writer.Write(rv, val); err != nil {
		return fmt.Errorf("[%T]Write error at `%s`: %s", writer, path, err)
	}
	return nil
}

func (path PropertyPath) Read(v interface{}) (interface{}, error) {

	rv := reflect.ValueOf(v)

	for i, reader := range path {
		if _, ok := reader.(Index); ok {
			rv = indirect(rv, indexReaderInterface)
			if ir, ok := rv.Interface().(IndexPathReader); ok {
				return ir.ReadIndexPath(path[i:])
			}
		}

		if _, ok := reader.(Field); ok {
			rv = indirect(rv, fieldReaderInterface)
			if fr, ok := rv.Interface().(FieldPathReader); ok {
				return fr.ReadFieldPath(path[i:])
			}
		}

		if r, err := reader.Read(rv); err != nil {
			return nil, fmt.Errorf("[%T]Read error at `%s`: %s", reader, PropertyPath(path[:i]), err)
		} else {
			if !r.IsValid() {
				return nil, fmt.Errorf("Invalid value at `%s`: %s", PropertyPath(path[:i]), r)
			}

			rv = r
		}
	}

	return rv.Interface(), nil
}

func (path PropertyPath) MustRead(v interface{}, dv ...interface{}) (value interface{}) {
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

type PropertyAccessor interface {
	Read(reflect.Value) (reflect.Value, error)
	Write(reflect.Value, interface{}) error
	String() string
}

type IndexPathReader interface {
	ReadIndexPath(PropertyPath) (interface{}, error)
}

type IndexPathWriter interface {
	WriteIndexPath(PropertyPath, interface{}) error
}

type FieldPathReader interface {
	ReadFieldPath(PropertyPath) (interface{}, error)
}

type FieldPathWriter interface {
	WriteFieldPath(PropertyPath, interface{}) error
}

func Write(selector string, v interface{}, val interface{}) error {
	return New(selector).Write(v, val)
}

func Read(selector string, v interface{}) (interface{}, error) {
	return New(selector).Read(v)
}

func MustRead(selector string, v interface{}, dv ...interface{}) (value interface{}) {
	return New(selector).MustRead(v, dv...)
}

func NotReadable(o interface{}, path interface{}) error {

	if index, ok := path.(int); ok {
		return fmt.Errorf("Can't read %T[%v]", o, index)
	}

	return fmt.Errorf("Can't read %T.%v", o, path)

}

func NotWriteable(o interface{}, path interface{}, val interface{}) error {

	if index, ok := path.(int); ok {
		return fmt.Errorf("Can't write %T to %T[%v]", val, o, index)
	}

	return fmt.Errorf("Can't write %T to %T.%v", val, o, path)
}

func indirect(v reflect.Value, iface reflect.Type) reflect.Value {

	// If v is a named type and is addressable,
	// start with its address, so that if the type has pointer methods,
	// we find them.
	if v.Kind() != reflect.Ptr && v.Type().Name() != "" && v.CanAddr() {
		v = v.Addr()
	}
	for {

		if iface != nil && v.Type().Implements(iface) {
			break
		}

		// Load value from interface
		if v.Kind() == reflect.Interface {
			v = v.Elem()
		}

		if v.Kind() != reflect.Ptr {
			break
		}

		v = v.Elem()
	}

	return v
}
