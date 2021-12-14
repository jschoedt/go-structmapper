[![Go](https://github.com/jschoedt/go-structmapper/actions/workflows/github-ci.yaml/badge.svg)](https://github.com/jschoedt/go-structmapper/actions/workflows/github-ci.yaml)
[![Coverage Status](https://coveralls.io/repos/github/jschoedt/go-structmapper/badge.svg)](https://coveralls.io/github/jschoedt/go-structmapper)
[![Go Report Card](https://goreportcard.com/badge/github.com/jschoedt/go-structmapper)](https://goreportcard.com/report/github.com/jschoedt/go-structmapper)
[![GoDoc](https://godoc.org/github.com/jschoedt/go-structmapper?status.svg)](https://godoc.org/github.com/jschoedt/go-structmapper)
[![GitHub](https://img.shields.io/github/license/jschoedt/go-structmapper)](https://github.com/jschoedt/go-structmapper/blob/master/LICENSE)

# go-structmapper

Convert any struct into a map and vice versa.

# Description

This library can recursively convert a struct to a map of type ```map[string]interface{}``` where the keys are the struct field names and the values are the field values.
Similarly, the library can set the fields of a struct using a map.

A mapping function can be used to convert keys or values before they are set.

#### Features

- Handles composed structs
- Handles nested structs
- Handles reference cycles
- Supports unexported fields
- Supports field mapping or conversion

#### Prerequisites

```
go get -u github.com/jschoedt/go-structmapper
```

#### Default usage

```go
// convert struct to map
s := &SomeStruct{Name: "John"}
mapper := mapper.New()
m, err := mapper.StructToMap(s) // m["Name"] == "John"

// convert map to struct
s = &SomeStruct{}
err := mapper.MapToStruct(m, s) // s.Name == "John"
```

#### Using a conversion mapping

A MapFunc can be used to map a key or value to some other key or value. Returning the MappingType mapper.Ignore will ignore that field. The MapFunc will be called on every field
that is encountered in the struct

```go
s := &SomeStruct{Name: "John"}
mapper := mapper.New()
mapper.MapFunc = func(inKey string, inVal interface{}) (mt MappingType, outKey string, outVal interface{}) {
	return mapper.Default, strings.ToLower(inKey), "Deere"
}
m, err := mapper.StructToMap(s) // m["name"] == "Deere"

// convert map to struct
s = &SomeStruct{}
mapper.CaseSensitive = false // now 'name' will match 'Name'
err := mapper.MapToStruct(m, s) // s.Name == "Deere"
```

[More examples](https://github.com/jschoedt/go-structmapper/blob/master/mappers_test.go)


