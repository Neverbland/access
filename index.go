package access

import (
	"fmt"
	"reflect"
)

var (
	indexReaderInterface = reflect.TypeOf((*IndexReader)(nil)).Elem()
	indexWriterInterface = reflect.TypeOf((*IndexWriter)(nil)).Elem()
)

type IndexReader interface {
	Index(int) (interface{}, error)
}

type IndexWriter interface {
IndexReader
	SetIndex(int, interface{}) error
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
	case reflect.Interface:
		return readIndex(v.Elem(), index, path)
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
	case reflect.Interface:
		var e reflect.Value

		if v.IsNil() && v.NumMethod() == 0 {
			array := make([]interface{}, index+1)
			e = reflect.ValueOf(&array).Elem()
		} else {
			e = v.Elem()
			e = makeAddressable(e)
		}

		err := writeIndex(e, index, path, w, wt)

		if err == nil {
			v.Set(e)
		}

		return err

	case reflect.Array, reflect.Slice:

		if path != nil {
			var iv reflect.Value

			if index >= v.Len() {
				iv = reflect.New(vt.Elem()).Elem()
			} else {
				iv = makeAddressable(v.Index(index))
			}

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
				cap += index-cap+1
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
