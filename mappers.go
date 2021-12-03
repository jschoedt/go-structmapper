package mapper

import (
	"errors"
	"reflect"
	"strings"
	"unsafe"
)

// MappingType the type of mapping to perform
type MappingType int

const (
	// Default use the default mapping logic provided
	Default MappingType = iota
	// Custom force the key to be a value
	Custom
	// Ignore ignores the field entirely
	Ignore
)

func (t MappingType) String() string {
	return [...]string{"Default", "Custom", "Ignore"}[t]
}

// MapFunc used to map a field name and value to another field name and value
type MapFunc func(inKey string, inVal interface{}) (mt MappingType, outKey string, outVal interface{})

// Mapper used for mapping structs to maps or other structs
type Mapper struct {
	MapFunc MapFunc
}

// New creates a new mapper
func New() *Mapper {
	m := &Mapper{DefaultMapFunc}
	return m
}

// NewWithMapFunc with a custom MapFunc
func NewWithMapFunc(mapFunc MapFunc) *Mapper {
	m := &Mapper{mapFunc}
	return m
}

// DefaultMapFunc default mapper that returns the same field name and value
func DefaultMapFunc(inKey string, inVal interface{}) (mt MappingType, outKey string, outVal interface{}) {
	return Default, inKey, inVal
}

// MapToStruct takes a map or a struct ptr (as fromPtr) and maps to a struct ptr
func (mapper *Mapper) MapToStruct(fromPtr interface{}, toPtr interface{}) error {
	c := make(map[interface{}]reflect.Value)
	return mapper.cachedMapMapToStruct(fromPtr, toPtr, c)
}

func (mapper *Mapper) cachedMapMapToStruct(fromPtr interface{}, toPtr interface{}, c map[interface{}]reflect.Value) error {
	toStruct := reflect.Indirect(reflect.ValueOf(toPtr))
	fromStruct := reflect.Indirect(reflect.ValueOf(fromPtr))
	m, ok := fromPtr.(map[string]interface{})
	if ok {
		valMap := make(map[string]reflect.Value, len(m))
		for k, v := range m {
			valMap[k] = reflect.ValueOf(v)
		}
		return mapper.mapMapToValues(valMap, toStruct, c)
	}
	fromMap, err := mapper.StructToMap(fromStruct)
	if err != nil {
		return err
	}
	return mapper.cachedMapMapToStruct(fromMap, toStruct, c)
}

func (mapper *Mapper) mapMapToValues(fromMap map[string]reflect.Value, toPtr reflect.Value, c map[interface{}]reflect.Value) error {
	toStruct := reflect.Indirect(toPtr) // entity is a pointer
	toMap := mapper.flatten(toStruct)
	var errStrings []string

	//fmt.Printf("toMap: %v  \n", toMap)
	//fmt.Printf("fromMap: %v  \n", fromMap)
	for fromName, fromField := range fromMap {
		if toField, ok := toMap[fromName]; ok {
			kind := fromField.Kind()
			if kind == reflect.Invalid {
				continue
			}

			mt, fromName, fromMapping := mapper.MapFunc(fromName, fromField.Interface())
			fromField := reflect.ValueOf(fromMapping)
			switch mt {
			case Ignore:
				continue
			case Custom:
				setField(fromField, toField)
				continue
			}

			// if same type just set it
			if fromField.Type().ConvertibleTo(toField.Type()) {
				setField(fromField, toField)
				continue
			}

			// convert the types
			switch kind {
			case reflect.Map: // try to map to object
				fromField = mapper.getFromValue(c, fromField, toField.Type())
			case reflect.Slice: // try to map to slice of objects
				if fromField.Len() == 0 {
					continue
				} else {
					elemSlice := reflect.MakeSlice(toField.Type(), fromField.Len(), fromField.Len())
					for i := 0; i < fromField.Len(); i++ {
						setField(mapper.getFromValue(c, fromField.Index(i), toField.Type().Elem()), elemSlice.Index(i))
					}
					fromField = elemSlice
				}
			}

			// try to set the value to target after conversion
			if fromField.Type().ConvertibleTo(toField.Type()) {
				setField(fromField, toField)
			} else {
				errStrings = append(errStrings, fromName+":["+stringVal(fromField)+" -> "+stringVal(toField)+"]")
			}
		}
	}
	if len(errStrings) > 0 {
		return errors.New(strings.Join(errStrings, "\n"))
	}
	return nil
}

func stringVal(val reflect.Value) string {
	if !val.IsValid() {
		return "nil"
	}
	i := val.Interface()
	if v, ok := i.(reflect.Value); ok {
		return stringVal(v)
	}
	if v, ok := i.(string); ok {
		return v
	}
	return "Unknown"
}

// Handles the creation of a value or a pointer to a value according to toType
func (mapper *Mapper) getFromValue(c map[interface{}]reflect.Value, fromField reflect.Value, toType reflect.Type) reflect.Value {
	var result reflect.Value
	//log.Printf("from: %v", reflect.TypeOf(fromField.Interface()))
	//log.Printf("to: %v", toType)
	if e, ok := c[fromField]; ok {
		result = e
	} else if reflect.TypeOf(fromField.Interface()).ConvertibleTo(toType) {
		return reflect.ValueOf(fromField.Interface())
	} else {
		if toType.Kind() == reflect.Map {
			result = reflect.MakeMapWithSize(toType, fromField.Len())
			for _, k := range fromField.MapKeys() {
				//fmt.Printf("from field: %v  \n", fromField.MapIndex(k))
				if fromField.MapIndex(k).Elem().Type().ConvertibleTo(toType.Elem()) {
					result.SetMapIndex(k, fromField.MapIndex(k).Elem().Convert(toType.Elem()))
				}
			}
			c[fromField] = result
		} else if toType.Kind() == reflect.Slice {
			result = reflect.MakeSlice(toType, fromField.Len(), fromField.Len())
			for i := 0; i < fromField.Len(); i++ {
				fromEmlPtr := fromField.Index(i)
				if fromEmlPtr.Elem().Type().ConvertibleTo(toType.Elem()) {
					result.Index(i).Set(fromField.Index(i).Elem().Convert(toType.Elem()))
				}
			}
			c[fromField] = result
		} else if toType.Kind() == reflect.Ptr {
			result = reflect.New(toType.Elem())
			c[fromField] = result
			mapper.cachedMapMapToStruct(fromField.Interface(), result.Interface(), c)
		} else {
			result = reflect.New(toType)
			c[fromField] = result
			mapper.cachedMapMapToStruct(fromField.Interface(), result.Interface(), c)
		}

	}
	if toType.Kind() == reflect.Ptr {
		return result
	}
	return reflect.Indirect(result)
}

func setField(fromField reflect.Value, toField reflect.Value) {
	if !toField.CanSet() {
		// now we can set unexported fields
		toField = reflect.NewAt(toField.Type(), unsafe.Pointer(toField.UnsafeAddr())).Elem()
	}
	toField.Set(fromField.Convert(toField.Type()))
}

type flattenResolver struct {
	cache       map[interface{}]map[string]interface{} // pointer to result map
	toResolve   map[interface{}][]func(m map[string]interface{})
	isResolving map[interface{}]bool
}

// StructToMap maps a struct pointer to a map. Including nested structs
func (mapper *Mapper) StructToMap(sp interface{}) (map[string]interface{}, error) {
	flattenResolver := &flattenResolver{
		isResolving: make(map[interface{}]bool),
		cache:       make(map[interface{}]map[string]interface{}),
		toResolve:   make(map[interface{}][]func(m map[string]interface{})),
	}

	m, err := mapper.cachedFlattenStruct(sp, flattenResolver)

	// resolve pointers
	for k, v := range flattenResolver.cache {
		for _, e := range flattenResolver.toResolve[k] {
			e(v)
		}
	}
	return m, err
}

func (mapper *Mapper) cachedFlattenStruct(s interface{}, resolver *flattenResolver) (map[string]interface{}, error) {
	//fmt.Printf("flatten: %v  \n", v)
	toStruct := reflect.ValueOf(s)
	if v, ok := s.(reflect.Value); ok {
		toStruct = v
	}
	v := reflect.Indirect(toStruct) // entity is a pointer
	fields := make(map[string]interface{}, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		sf := v.Type().Field(i)

		key := sf.Name
		mt, key, val := mapper.MapFunc(key, getDefaultValue(f))
		switch mt {
		case Ignore:
			continue
		case Custom:
			fields[key] = val
			continue
		}

		//log.Printf("Name: " + sf.Name)

		switch f.Kind() {
		case reflect.Interface:
			fallthrough
		case reflect.Ptr:
			// handle nil pointers and other types than struct
			val := reflect.Indirect(f)
			if val.Kind() == reflect.Invalid || val.Kind() != reflect.Struct {
				fields[key] = getDefaultValue(val)
				break
			}

			// are we resolving this pointer already? Then add to the resolver list
			ptr := getDefaultValue(f)
			if resolver.isResolving[ptr] == true {
				resolver.toResolve[ptr] = append(resolver.toResolve[ptr], func(m map[string]interface{}) {
					fields[key] = m // closure ok
				})
			} else {
				// ok resolve it then
				resolver.isResolving[ptr] = true
				if m, err := mapper.cachedFlattenStruct(f, resolver); err != nil {
					return nil, err
				} else {
					resolver.cache[ptr] = m
					fields[key] = m
				}
			}
		case reflect.Struct:
			if m, err := mapper.cachedFlattenStruct(f, resolver); err != nil {
				return nil, err
			} else {
				if sf.Anonymous {
					for k, v := range m {
						fields[k] = v
					}
				} else {
					fields[key] = m
				}
			}
		case reflect.Array:
			fallthrough
		case reflect.Slice:
			typ := f.Type().Elem()
			if typ.Kind() == reflect.Ptr {
				typ = typ.Elem()
			}
			if typ.Kind() == reflect.Struct { // convert to slice of maps
				elemType := reflect.TypeOf((map[string]interface{})(nil))
				elemSlice := reflect.MakeSlice(reflect.SliceOf(elemType), f.Len(), f.Len())
				for i := 0; i < f.Len(); i++ {
					if m, err := mapper.cachedFlattenStruct(f.Index(i), resolver); err != nil {
						return nil, err
					} else {
						elemSlice.Index(i).Set(reflect.ValueOf(m))
					}
				}
				fields[key] = elemSlice.Interface()
				break
			}
			fallthrough
		default:
			fields[key] = getDefaultValue(f)
		}
	}
	return fields, nil
}

func getDefaultValue(f reflect.Value) interface{} {
	if !f.IsValid() {
		return reflect.ValueOf(nil)
	}
	if !f.CanInterface() {
		// now we can get unexported fields
		//fmt.Printf("unexported field: %v  \n", sf.Name)
		f = reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
	}
	return f.Interface()
}

func (mapper *Mapper) flatten(v reflect.Value) map[string]reflect.Value {
	//fmt.Printf("flatten: %v  \n", v)
	fields := make(map[string]reflect.Value, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		sf := v.Type().Field(i)

		//log.Printf("Name: " + sf.Name)

		switch f.Kind() {
		case reflect.Struct:
			if sf.Anonymous {
				embedFields := mapper.flatten(f)
				for k, v := range embedFields {
					fields[k] = v
				}
				break
			}
			fallthrough
		default:
			if !f.CanInterface() {
				// now we can get unexported fields
				//fmt.Printf("unexported field: %v  \n", sf.Name)
				f = reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
			}
			fields[sf.Name] = f
		}
	}
	return fields
}
