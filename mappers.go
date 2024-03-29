package mapper

import (
	"errors"
	"reflect"
	"strconv"
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
	// MapFunc maps keys and values to some other key or value
	MapFunc MapFunc
	// CaseSensitive if MapToStruct should be case-sensitive
	CaseSensitive bool
}

// New creates a new mapper
func New() *Mapper {
	m := &Mapper{NilMapFunc, true}
	return m
}

// NilMapFunc default mapper that returns the same field name and value
func NilMapFunc(inKey string, inVal interface{}) (mt MappingType, outKey string, outVal interface{}) {
	if IsNil(inVal) {
		return Ignore, inKey, nil
	}
	return Default, inKey, inVal
}

// IsNil returns true if val is nil or a nil pointer
func IsNil(val interface{}) bool {
	if val == nil {
		return true
	} else if reflect.ValueOf(val).Kind() == reflect.Ptr && reflect.ValueOf(val).IsNil() {
		return true
	}
	return false
}

// MapToStruct takes a map or a struct ptr (as fromPtr) and maps to a struct ptr
func (mapper *Mapper) MapToStruct(fromPtr interface{}, toPtr interface{}) error {
	c := make(map[interface{}]reflect.Value)
	return mapper.cachedMapMapToStruct(fromPtr, toPtr, c)
}

func (mapper *Mapper) cachedMapMapToStruct(fromPtr interface{}, toPtr interface{}, c map[interface{}]reflect.Value) error {
	toStruct := reflect.Indirect(reflect.ValueOf(toPtr))
	fromStruct := reflect.Indirect(reflect.ValueOf(fromPtr))
	if fromStruct.Kind() == reflect.Map {
		valMap := make(map[string]reflect.Value, fromStruct.Len())
		for _, k := range fromStruct.MapKeys() {
			valMap[k.String()] = fromStruct.MapIndex(k)
		}
		return mapper.mapMapToValues(valMap, toStruct, c)
	} else if fromStruct.Kind() == reflect.Struct {
		fromMap, err := mapper.StructToMap(fromStruct)
		if err != nil {
			return err
		}
		return mapper.cachedMapMapToStruct(fromMap, toStruct, c)
	}
	return errors.New("fromPtr must be either map or struct")
}

func (mapper *Mapper) mapMapToValues(fromMap map[string]reflect.Value, toPtr reflect.Value, c map[interface{}]reflect.Value) error {
	toStruct := reflect.Indirect(toPtr) // entity is a pointer
	toMap := mapper.flatten(toStruct)
	var errStrings []string

	//fmt.Printf("toMap: %v  \n", toMap)
	//fmt.Printf("fromMap: %v  \n", fromMap)
	for fromName, fromField := range fromMap {
		mt, fromName, fromMapping := mapper.MapFunc(fromName, getDefaultValue(fromField))
		fromField := reflect.ValueOf(fromMapping)

		if toField, ok := toMap[mapper.setCasing(fromName)]; ok {
			kind := fromField.Kind()
			if kind == reflect.Invalid {
				continue
			}

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
			var err error
			switch kind {
			case reflect.Map: // try to map to object
				fromField, err = mapper.getFromValue(c, fromField, toField.Type())
				if err != nil {
					errStrings = append(errStrings, "from map: "+fromName+": "+err.Error())
				}
			case reflect.Slice: // try to map to slice of objects
				if fromField.Len() == 0 {
					continue
				} else {
					elemSlice := reflect.MakeSlice(toField.Type(), fromField.Len(), fromField.Len())
					for i := 0; i < fromField.Len(); i++ {
						if value, err := mapper.getFromValue(c, fromField.Index(i), toField.Type().Elem()); err != nil {
							errStrings = append(errStrings, "from slice: "+fromName+"["+strconv.Itoa(i)+"]: "+err.Error())
						} else {
							setField(value, elemSlice.Index(i))
						}
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
		return "nil of type:" + val.String()
	}
	if val.CanInterface() {
		i := val.Interface()
		if v, ok := i.(reflect.Value); ok {
			return stringVal(v)
		}
		if v, ok := i.(string); ok {
			return v
		}
		return val.Type().String()
	}

	return "nil of type: " + val.Type().String()
}

// Handles the creation of a value or a pointer to a value according to toType
func (mapper *Mapper) getFromValue(c map[interface{}]reflect.Value, fromField reflect.Value, toType reflect.Type) (reflect.Value, error) {
	var result reflect.Value
	//log.Printf("from: %v", reflect.TypeOf(fromField.Interface()))
	//log.Printf("to: %v", toType)
	if e, ok := c[fromField]; ok {
		result = e
	} else if reflect.TypeOf(fromField.Interface()).ConvertibleTo(toType) {
		return reflect.ValueOf(fromField.Interface()), nil
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
			if err := mapper.cachedMapMapToStruct(fromField.Interface(), result.Interface(), c); err != nil {
				return result, err
			}
		} else {
			result = reflect.New(toType)
			c[fromField] = result
			if err := mapper.cachedMapMapToStruct(fromField.Interface(), result.Interface(), c); err != nil {
				return reflect.Indirect(result), err
			}
		}

	}
	if toType.Kind() == reflect.Ptr {
		return result, nil
	}
	return reflect.Indirect(result), nil
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
			if f.IsNil() {
				fields[key] = getDefaultValue(f)
				break
			}

			val := reflect.Indirect(f)
			if val.Kind() == reflect.Invalid || val.Kind() != reflect.Struct {
				fields[key] = getDefaultValue(f)
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
		return nil
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
					fields[mapper.setCasing(k)] = v
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
			fields[mapper.setCasing(sf.Name)] = f
		}
	}
	return fields
}

func (mapper *Mapper) setCasing(s string) string {
	if mapper.CaseSensitive == false {
		return strings.ToLower(s)
	}
	return s
}
