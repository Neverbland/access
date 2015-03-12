package access

import (
	"fmt"
	"reflect"
)

type Index int

func (i Index) String() string {
	return fmt.Sprintf("[%v]", int(i))
}

func (i Index) Read(v reflect.Value) (reflect.Value, error) {

	index := int(i)

	vt := v.Type()

	switch vt.Kind() {
	case reflect.Array, reflect.Slice:

		if index >= v.Len() {
			return reflect.Value{}, NotReadable(v.Interface(), index)
		}

		return v.Index(index), nil
	default:
		return reflect.Value{}, fmt.Errorf("slice, array or IndexReader instance expected. Given %T", v.Interface())
	}
}

func (i Index) Write(v reflect.Value, val interface{}) error {

	index := int(i)

	vt := v.Type()

	if vt.Kind() == reflect.Ptr {
		v = v.Elem()
		vt = v.Type()
	}

	switch vt.Kind() {
	case reflect.Array, reflect.Slice:
		rval := reflect.ValueOf(val)
		rvalt := rval.Type()

		if !rvalt.AssignableTo(vt.Elem()) {
			return NotWriteable(v.Interface(), index, rval.Interface())
		}

		// Get element of array, growing if necessary.
		if v.Kind() == reflect.Slice {
			cap := v.Cap()
			// Grow slice if necessary
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

		v.Index(index).Set(rval)

		return nil
	default:
		return fmt.Errorf("slice, array or IndexWriter instance expected. Given %T", v.Interface())
	}
}
