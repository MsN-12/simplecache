package simplecache

import "reflect"

type cloneVisit struct {
	typ reflect.Type
	ptr uintptr
}

// DeepClone returns a reflection-based deep copy of value.
//
// DeepClone supports common Go values including structs, arrays, slices, maps,
// pointers, and interfaces. Unsupported runtime/resource values such as funcs,
// channels, and unsafe pointers are copied as-is. Unexported struct fields are
// shallow-copied because Go reflection cannot safely set them.
//
// For production-critical values, resource-owning values, or values with
// invariants, prefer a custom CloneFunc that copies exactly what must be copied.
func DeepClone[V any](value V) V {
	cloned := deepCloneValue(reflect.ValueOf(value), make(map[cloneVisit]reflect.Value))
	if !cloned.IsValid() {
		return value
	}

	return cloned.Interface().(V)
}

func deepCloneValue(value reflect.Value, seen map[cloneVisit]reflect.Value) reflect.Value {
	if !value.IsValid() {
		return value
	}

	switch value.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128,
		reflect.String:
		return value
	case reflect.Pointer:
		return deepClonePointer(value, seen)
	case reflect.Interface:
		return deepCloneInterface(value, seen)
	case reflect.Struct:
		return deepCloneStruct(value, seen)
	case reflect.Array:
		return deepCloneArray(value, seen)
	case reflect.Slice:
		return deepCloneSlice(value, seen)
	case reflect.Map:
		return deepCloneMap(value, seen)
	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return value
	default:
		return value
	}
}

func deepClonePointer(value reflect.Value, seen map[cloneVisit]reflect.Value) reflect.Value {
	if value.IsNil() {
		return reflect.Zero(value.Type())
	}

	visit := cloneVisit{typ: value.Type(), ptr: value.Pointer()}
	if cloned, ok := seen[visit]; ok {
		return cloned
	}

	cloned := reflect.New(value.Type().Elem())
	seen[visit] = cloned
	cloned.Elem().Set(deepCloneValue(value.Elem(), seen))
	return cloned
}

func deepCloneInterface(value reflect.Value, seen map[cloneVisit]reflect.Value) reflect.Value {
	if value.IsNil() {
		return reflect.Zero(value.Type())
	}

	clonedElem := deepCloneValue(value.Elem(), seen)
	cloned := reflect.New(value.Type()).Elem()
	cloned.Set(clonedElem)
	return cloned
}

func deepCloneStruct(value reflect.Value, seen map[cloneVisit]reflect.Value) reflect.Value {
	cloned := reflect.New(value.Type()).Elem()
	cloned.Set(value)

	for i := 0; i < value.NumField(); i++ {
		field := cloned.Field(i)
		if !field.CanSet() {
			continue
		}

		field.Set(deepCloneValue(value.Field(i), seen))
	}

	return cloned
}

func deepCloneArray(value reflect.Value, seen map[cloneVisit]reflect.Value) reflect.Value {
	cloned := reflect.New(value.Type()).Elem()
	for i := 0; i < value.Len(); i++ {
		cloned.Index(i).Set(deepCloneValue(value.Index(i), seen))
	}

	return cloned
}

func deepCloneSlice(value reflect.Value, seen map[cloneVisit]reflect.Value) reflect.Value {
	if value.IsNil() {
		return reflect.Zero(value.Type())
	}

	cloned := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
	if value.Len() > 0 {
		visit := cloneVisit{typ: value.Type(), ptr: value.Pointer()}
		if existing, ok := seen[visit]; ok {
			return existing
		}
		seen[visit] = cloned
	}

	for i := 0; i < value.Len(); i++ {
		cloned.Index(i).Set(deepCloneValue(value.Index(i), seen))
	}

	return cloned
}

func deepCloneMap(value reflect.Value, seen map[cloneVisit]reflect.Value) reflect.Value {
	if value.IsNil() {
		return reflect.Zero(value.Type())
	}

	visit := cloneVisit{typ: value.Type(), ptr: value.Pointer()}
	if cloned, ok := seen[visit]; ok {
		return cloned
	}

	cloned := reflect.MakeMapWithSize(value.Type(), value.Len())
	seen[visit] = cloned

	iter := value.MapRange()
	for iter.Next() {
		cloned.SetMapIndex(
			deepCloneValue(iter.Key(), seen),
			deepCloneValue(iter.Value(), seen),
		)
	}

	return cloned
}
