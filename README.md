[![Go](https://github.com/jschoedt/go-structmapper/actions/workflows/github-ci.yaml/badge.svg)](https://github.com/jschoedt/go-structmapper/actions/workflows/github-ci.yaml)
[![Coverage Status](https://coveralls.io/repos/github/jschoedt/go-structmapper/badge.svg)](https://coveralls.io/github/jschoedt/go-structmapper)
[![Go Report Card](https://goreportcard.com/badge/github.com/jschoedt/go-structmapper)](https://goreportcard.com/report/github.com/jschoedt/go-structmapper)
[![GoDoc](https://godoc.org/github.com/jschoedt/go-structmapper?status.svg)](https://godoc.org/github.com/jschoedt/go-structmapper)
[![GitHub](https://img.shields.io/github/license/jschoedt/go-structmapper)](https://github.com/jschoedt/go-structmapper/blob/master/LICENSE)

# go-structmapper

Convert a struct into a map and vice versa.

# Description

Convert a struct into a map and vice versa.

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
mapper := New()
m, err := mapper.MapStructToMap(s) // m["Name"] == "John"

// convert map to struct
s = &SomeStruct{}
mapper.MapToStruct(m, &s) // s.Name == "John"
```

#### Using a conversion mapping

A MapFunc can be used to map a key or value to some other key or value. Returning the MappingType Ignore will ignore that field. The MapFunc will be called on every field that is
encountered in the struct

```go
s := &SomeStruct{Name: "John"}
mapFunc := func(inKey string, inVal interface{}) (mt MappingType, outKey string, outVal interface{}) {
	return Default, strings.ToLower(inKey), "Deere"
}
mapper := NewWithMapFunc(mapFunc)
m, err := mapper.MapStructToMap(s) // m["name"] == "Deere"
```