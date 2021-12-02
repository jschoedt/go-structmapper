package mapper

import (
	"github.com/google/go-cmp/cmp"
	"strings"
	"testing"
	"time"
)

type ID struct {
	ID string
}

type Person struct {
	ID
	Name      string
	Spouse    *Person
	Relations []*Relation
}

type Car struct {
	Make       string
	Owner      *Person
	Driver     Person
	Passengers []Person
	Tags       []string
	Numbers    []int
	Year       time.Time
}

type Relation struct {
	Name    string
	Friends []*Person
}

func TestStructToMap(t *testing.T) {
	john := Person{Name: "John"}
	mary := Person{Name: "Mary"}
	john.Spouse = &mary
	mary.Spouse = &john

	friend1 := &Person{Name: "Friend1"}
	friend2 := &Person{Name: "Friend2"}

	// Add the nested relation
	john.Relations = []*Relation{{Friends: []*Person{friend1, friend2}}}

	now := time.Now()

	car := &Car{
		Make:       "Toyota",
		Owner:      &john,
		Driver:     Person{Name: "Mark"}, // embedded entity
		Passengers: []Person{john, mary},
		Tags:       []string{"tag1", "tag2"},
		Numbers:    []int{1, 2, 3},
		Year:       now,
	}

	mapper := New()
	m, err := mapper.StructToMap(car)
	if err != nil {
		t.Errorf("Could not convert struct to map %v", err)
	}

	newCar := &Car{}
	if err := mapper.MapToStruct(m, newCar); err != nil {
		t.Errorf("Could not map to struct %v", err)
	}

	if car.Owner.Spouse.Name != newCar.Owner.Spouse.Name || car.Owner.Spouse.Spouse.Name != newCar.Owner.Spouse.Spouse.Name {
		t.Errorf("The structs cycle did not match %v - %v", car, newCar)
	}

	//  cmp.Equal does not handle cycles so break it
	car.Owner.Spouse.Spouse = nil
	newCar.Owner.Spouse.Spouse = nil

	if !cmp.Equal(car, newCar) {
		t.Errorf("The structs were not the same %v - %v", car, newCar)
	}
}

func TestFilter(t *testing.T) {
	john := Person{Name: "John"}

	mapFunc := func(inKey string, inVal interface{}) (mt MappingType, outKey string, outVal interface{}) {
		return Default, strings.ToLower(inKey), inVal
	}
	mapper := NewWithMapFunc(mapFunc)
	m, err := mapper.StructToMap(&john)
	if err != nil {
		t.Errorf("Could not convert struct to map %v", err)
	}

	if _, ok := m["name"]; !ok {
		t.Errorf("The lowercase key:'name' was not set om the map")
	}

	m["Name"] = "Deere"
	if err := mapper.MapToStruct(m, &john); err != nil {
		t.Errorf("Could not map to struct %v", err)
	}

	if john.Name != "Deere" {
		t.Errorf("Name should me Deere")
	}
}
