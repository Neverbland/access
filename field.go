package access

import (
	"fmt"
	"reflect"
	"strings"
)

type Field string

func (f Field) String() string {
	return string(f)
}

func (f Field) Camelcased() string {
	return strings.Replace(strings.Title(strings.Replace(string(f), "_", " ", -1)), " ", "", -1)
}

func (f Field) Read(v reflect.Value) (reflect.Value, error) {

	fieldname := string(f)

	vt := v.Type()

	switch vt.Kind() {
	case reflect.Map:

		if kk := vt.Key().Kind(); kk != reflect.String {
			return reflect.Value{}, NotReadable(v.Interface(), fieldname)
		}

		fv := v.MapIndex(reflect.ValueOf(fieldname))

		if fv.IsValid() {
			return fv, nil
		}

		return reflect.Value{}, NotReadable(v.Interface(), fieldname)

	case reflect.Struct:

		camelcased := f.Camelcased()

		if field, ok := vt.FieldByName(camelcased); ok {

			return v.FieldByIndex(field.Index), nil
		}

		vp := v

		if v.CanAddr() {
			vp = v.Addr()
		}

		for _, name := range []string{camelcased, "Get" + camelcased} {

			if method := vp.MethodByName(name); method.IsValid() {

				if mt := method.Type(); mt.NumIn() != 0 || mt.NumOut() == 0 {
					continue
				}

				return method.Call([]reflect.Value{})[0], nil
			}
		}

		return reflect.Value{}, NotReadable(v.Interface(), fieldname)
	default:
		return reflect.Value{}, fmt.Errorf("struct or map instance expected. Given %T", v.Interface())
	}
}

func (f Field) Write(v reflect.Value, val interface{}) error {
	fieldname := string(f)

	vt := v.Type()

	switch vt.Kind() {
	case reflect.Map:

		rval := reflect.ValueOf(val)
		rvalt := rval.Type()

		if kk := vt.Key().Kind(); kk != reflect.String {
			return fmt.Errorf("map key type expected to be a string. Got %s", kk)
		}

		if !rvalt.AssignableTo(vt.Elem()) {
			return NotWriteable(v.Interface(), fieldname, val)
		}

		v.SetMapIndex(reflect.ValueOf(fieldname), rval)

		return nil

	case reflect.Struct:

		rval := reflect.ValueOf(val)
		rvalt := rval.Type()
		camelcased := f.Camelcased()

		if field, ok := vt.FieldByName(camelcased); ok {
			if !rvalt.AssignableTo(field.Type) {
				return NotWriteable(v.Interface(), fieldname, val)
			}
			v.FieldByIndex(field.Index).Set(rval)
			return nil
		}

		vp := v

		if v.CanAddr() {
			vp = v.Addr()
		}

		for _, name := range []string{"Set" + camelcased} {

			if method := vp.MethodByName(name); method.IsValid() {

				mt := method.Type()
				numIn := mt.NumIn()

				if numIn > 2 || numIn == 0 || (numIn == 2 && !mt.IsVariadic()) {
					continue
				}

				if !rvalt.ConvertibleTo(mt.In(0)) {
					return NotWriteable(v.Interface(), fieldname, val)
				}

				method.Call([]reflect.Value{rval})
				return nil
			}
		}

		return fmt.Errorf("can't access  field `%s` in struct `%s.%s`", fieldname, vt.PkgPath(), vt.Name())
	default:
		return fmt.Errorf("struct or map instance expected. Given %T", v)
	}

	return nil
}
