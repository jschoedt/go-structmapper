[![Go](https://github.com/jschoedt/go-structmapper/actions/workflows/github-ci.yaml/badge.svg)](https://github.com/jschoedt/go-structmapper/actions/workflows/github-ci.yaml)
[![Coverage Status](https://coveralls.io/repos/github/jschoedt/go-structmapper/badge.svg)](https://coveralls.io/github/jschoedt/go-structmapper)
[![Go Report Card](https://goreportcard.com/badge/github.com/jschoedt/go-structmapper)](https://goreportcard.com/report/github.com/jschoedt/go-structmapper)
[![GoDoc](https://godoc.org/github.com/jschoedt/go-structmapper?status.svg)](https://godoc.org/github.com/jschoedt/go-structmapper)

# go-structmapper

Convert a struct into a map and vice versa.

#### Features

- Handles composed structs
- Handles nested structs
- Handles reference cycles
- Supports field filtering or conversion

#### Prerequisites

```
go get -u github.com/jschoedt/go-structmapper
```

#### Default usage

```go
s := SomeStruce{Name: "John"}
mapper := New()
m, err := mapper.MapStructToMap(s)
m["Name"] == "John"
```

#### Using a conversion mapping

```go
s := SomeStruce{Name: "John"}
mapper := NewWithFilter(LowercaseMapFunk)
m, err := mapper.MapStructToMap(s)
m["name"] == "John"
```