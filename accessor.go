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
	indexReaderInterface = reflect.TypeOf((*IndexReader)(nil)).Elem()
	indexWriterInterface = reflect.TypeOf((*IndexWriter)(nil)).Elem()
	fieldReaderInterface = reflect.TypeOf((*FieldReader)(nil)).Elem()
	fieldWriterInterface = reflect.TypeOf((*FieldWriter)(nil)).Elem()
}

func New(v interface{}) PropertyPath {

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
		panic("empty selector")
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

	return PropertyPath(parts)
}

type PropertyPath []interface{}

func (p PropertyPath) String() string {
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

func (p PropertyPath) write(v reflect.Value, w reflect.Value, wt reflect.Type) (err error) {

	if !v.CanAddr() {
		return Error{fmt.Errorf("Got unadressable value"), []interface{}{}}
	}

	var rpath *PropertyPath

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

func (p PropertyPath) read(v reflect.Value) (rv reflect.Value, err error) {

	var rpath *PropertyPath

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

func readIndex(v reflect.Value, index int, path *PropertyPath) (reflect.Value, error) {
	v = indirect(v, indexReaderInterface)
	vt := v.Type()

	if r, ok := v.Interface().(IndexReader); ok {
		val, err := r.Index(index)
		iv := reflect.ValueOf(val)
		if err != nil {
			return iv, err
		}
		if path != nil {
			return path.read(iv)
		}
		return iv, err
	}

	switch vt.Kind() {
	case reflect.Array, reflect.Slice:

		if index >= v.Len() {
			return reflect.Value{}, fmt.Errorf("Index %d out of range %d.", index, v.Len())
		}

		iv := v.Index(index)

		if path == nil {
			return iv, nil
		}

		return path.read(iv)
	default:
		return reflect.Value{}, fmt.Errorf("slice, array or IndexWriter instance expected")
	}
}

func readField(v reflect.Value, field string, path *PropertyPath) (reflect.Value, error) {

	v = indirect(v, fieldReaderInterface)

	vt := v.Type()
	vf := reflect.ValueOf(field)

	if r, ok := v.Interface().(FieldReader); ok {
		val, err := r.Field(field)
		fv := reflect.ValueOf(val)
		if err != nil {
			return fv, err
		}
		if path != nil {
			return path.read(fv)
		}
		return fv, err
	}

	switch vt.Kind() {
	case reflect.Map:

		if kk := vt.Key().Kind(); kk != reflect.String {
			return reflect.Value{}, fmt.Errorf("Map key type is not a string")
		}

		fv := v.MapIndex(vf)

		if !fv.IsValid() {
			return fv, fmt.Errorf("Map key not exists")
		}

		if path == nil {
			return fv, nil
		}

		return path.read(fv)

	case reflect.Struct:

		field = camelcased(field)

		if ft, ok := vt.FieldByName(field); ok {
			fv := v.FieldByIndex(ft.Index)

			if path == nil {
				return fv, nil
			}

			return path.read(fv)
		}

		if v.CanAddr() {
			v = v.Addr()
		}

		methods := []string{field, "Get" + field}
		for _, m := range methods {

			if mv := v.MethodByName(m); mv.IsValid() {

				if mt := mv.Type(); mt.NumIn() != 0 || mt.NumOut() != 1 {
					continue
				}

				fv := mv.Call([]reflect.Value{})[0]

				if path == nil {
					return fv, nil
				}
				return path.read(fv)
			}
		}

		return reflect.Value{}, fmt.Errorf("Struct has no field %s nor methods %v which satisfy signature func(...) (interface{}) ", camelcased, methods)
	default:
		return reflect.Value{}, fmt.Errorf("struct,map or FieldReader instance expected")
	}
}

func writeIndex(v reflect.Value, index int, path *PropertyPath, w reflect.Value, wt reflect.Type) error {

	v = indirect(v, indexWriterInterface)

	if r, ok := v.Interface().(IndexWriter); ok {

		if path != nil {
			val, err := r.Index(index)
			if err != nil {
				return err
			}

			iv := makeAddressable(reflect.ValueOf(val))
			if err := path.write(iv, w, wt); err != nil {
				return err
			}

			return writeIndex(v, index, nil, iv, iv.Type())
		}

		r.SetIndex(index, w.Interface())

		return nil
	}

	vt := v.Type()

	switch vt.Kind() {
	case reflect.Array, reflect.Slice:

		if path != nil {
			iv := makeAddressable(v.Index(index))
			if err := path.write(iv, w, wt); err != nil {
				return err
			}
			return writeIndex(v, index, nil, iv, iv.Type())
		}

		if !wt.AssignableTo(vt.Elem()) {
			return fmt.Errorf("Can't assign value")
		}

		// Grow slice if necessary

		if v.Kind() == reflect.Slice {
			cap := v.Cap()
			if index >= cap {
				cap += index - cap + 1
				newv := reflect.MakeSlice(v.Type(), v.Len(), cap)
				reflect.Copy(newv, v)
				v.Set(newv)
			}

			if index >= v.Len() {
				v.SetLen(index + 1)
			}
		}

		if index >= v.Len() {
			return fmt.Errorf("Index %d out of range %d.", index, v.Len())
		}

		v.Index(index).Set(w)

		return nil
	default:
		return fmt.Errorf("slice, array or IndexWriter instance expected")
	}
}

func writeField(v reflect.Value, field string, path *PropertyPath, w reflect.Value, wt reflect.Type) error {

	v = indirect(v, fieldWriterInterface)

	if r, ok := v.Interface().(FieldWriter); ok {

		if path != nil {
			val, err := r.Field(field)
			if err != nil {
				return err
			}
			fv := makeAddressable(reflect.ValueOf(val))
			if err := path.write(fv, w, wt); err != nil {
				return err
			}
			return writeField(v, field, nil, fv, fv.Type())
		}

		return r.SetField(field, w.Interface())
	}

	vt := v.Type()
	vf := reflect.ValueOf(field)

	switch vt.Kind() {
	case reflect.Map:

		if kk := vt.Key().Kind(); kk != reflect.String {
			return fmt.Errorf("Map key type is not a string")
		}

		if path != nil {
			fv := v.MapIndex(vf)

			if !fv.IsValid() {
				return fmt.Errorf("Map key not exists")
			}

			if fv.Kind() == reflect.Interface {
				fv = fv.Elem()
			}

			fv = makeAddressable(fv)

			if err := path.write(fv, w, wt); err != nil {
				return err
			}

			return writeField(v, field, nil, fv, fv.Type())
		}

		v.SetMapIndex(vf, w)
		return nil

	case reflect.Struct:

		field = camelcased(field)

		if ft, ok := vt.FieldByName(field); ok {
			fv := v.FieldByIndex(ft.Index)
			if path != nil {
				return path.write(fv, w, wt)
			}

			fv.Set(w)
			return nil
		}

		if v.CanAddr() {
			v = v.Addr()
		}

		if path != nil {
			methods := []string{field, "Get" + field}
			for _, m := range methods {

				if mv := v.MethodByName(m); mv.IsValid() {

					if mt := mv.Type(); mt.NumIn() != 0 || mt.NumOut() != 0 {
						continue
					}

					fv := mv.Call([]reflect.Value{})[0]

					if err := path.write(fv, w, wt); err != nil {
						return err
					}
					return writeField(v, field, nil, fv, fv.Type())
				}
			}

			return fmt.Errorf("Struct has no field %s nor methods %v which satisfy signature func() (interface{}) ", camelcased, methods)
		}

		methods := []string{field, "Set" + field}
		for _, m := range methods {

			if mv := v.MethodByName(m); mv.IsValid() {

				mt := mv.Type()
				numIn := mt.NumIn()

				if numIn > 2 || numIn == 0 || (numIn == 2 && !mt.IsVariadic()) {
					continue
				}

				if !wt.ConvertibleTo(mt.In(0)) {
					return fmt.Errorf("Can't call %s(%s)", m, wt)
				}

				mv.Call([]reflect.Value{w})
				return nil
			}
		}
		return fmt.Errorf("Struct has no field %s nor methods %v which satisfy signature func(interface{},...) (...) ", camelcased, methods)

	default:
		return fmt.Errorf("struct,map or FieldWriter instance expected")
	}
}

func (path PropertyPath) Write(v interface{}, w interface{}) error {

	rv := reflect.ValueOf(v)

	if rv.Kind() != reflect.Ptr {
		return Error{fmt.Errorf("Non pointer value"), []interface{}{}}
	}

	rw := reflect.ValueOf(w)

	return path.write(rv.Elem(), rw, rw.Type())
}

func (path PropertyPath) Read(v interface{}) (interface{}, error) {

	rv := reflect.ValueOf(v)

	re, err := path.read(rv)

	if err != nil || !re.IsValid() {
		return nil, err
	}

	return re.Interface(), err
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

type Error struct {
	error
	Path []interface{}
}

func (e Error) Error() string {
	return fmt.Sprintf("%s at `%s`", e.error, PropertyPath(e.Path))
}

func (e Error) back(s interface{}) Error {
	e.Path = append([]interface{}{s}, e.Path...)
	return e
}

type IndexReader interface {
	Index(int) (interface{}, error)
}

type IndexWriter interface {
	IndexReader
	SetIndex(int, interface{}) error
}

type FieldReader interface {
	Field(string) (interface{}, error)
}

type FieldWriter interface {
	FieldReader
	SetField(string, interface{}) error
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

func makeAddressable(v reflect.Value) reflect.Value {
	c := reflect.New(v.Type())
	c.Elem().Set(v)
	return c.Elem()
}

func camelcased(s string) string {
	return strings.Replace(strings.Title(strings.Replace(s, "_", " ", -1)), " ", "", -1)
}
