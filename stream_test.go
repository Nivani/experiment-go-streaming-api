package stream

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

type person struct {
	name        string
	dateOfBirth time.Time
}

func (p person) isChild() bool {
	eighteenYears := time.Hour * 24 * 365 * 18
	return time.Since(p.dateOfBirth) < eighteenYears
}

func TestStream(t *testing.T) {
	john := person{
		name:        "John",
		dateOfBirth: time.Date(1987, 4, 17, 0, 0, 0, 0, time.UTC),
	}
	lucy := person{
		name:        "Lucy",
		dateOfBirth: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
	}
	mary := person{
		name:        "Mary",
		dateOfBirth: time.Date(1989, 3, 21, 0, 0, 0, 0, time.UTC),
	}
	spencer := person{
		name:        "Spencer",
		dateOfBirth: time.Date(2023, 9, 4, 0, 0, 0, 0, time.UTC),
	}
	rose := person{
		name:        "Rose",
		dateOfBirth: time.Date(2022, 5, 20, 0, 0, 0, 0, time.UTC),
	}

	namesOfChildren := Pipe(
		ToStream(john, lucy, mary, spencer, rose),
		Filter(func(p person) bool {
			return p.isChild()
		}),
		Map(func(p person) string {
			return p.name
		}),
		ToSlice[string](),
	)
	joined := strings.Join(namesOfChildren, ", ")

	expected := "Lucy, Spencer, Rose"
	if joined != expected {
		t.Errorf("'%s' != '%s'", joined, expected)
	}
}

type Stream[T interface{}] interface {
	Next() (T, bool)
}

func ToStream[T interface{}](items ...T) Stream[T] {
	return &sliceStream[T]{
		items: items,
		i:     0,
	}
}

func Pipe[T0 interface{}, T1 interface{}, T2 interface{}, T3 interface{}](
	initial Stream[T0],
	t1 func(Stream[T0]) Stream[T1],
	t2 func(Stream[T1]) Stream[T2],
	result func(Stream[T2]) T3,
) T3 {
	return result(t2(t1(initial)))
}

type sliceStream[T interface{}] struct {
	items []T
	i     int
}

func (a *sliceStream[T]) Next() (T, bool) {
	if a.i >= len(a.items) {
		return defaultValue[T](), false
	}

	item := a.items[a.i]
	a.i++
	return item, true
}

func Filter[T interface{}](predicate func(T) bool) func(Stream[T]) Stream[T] {
	return func(stream Stream[T]) Stream[T] {
		return &filterStream[T]{
			stream:    stream,
			predicate: predicate,
		}
	}
}

type filterStream[T interface{}] struct {
	stream    Stream[T]
	predicate func(T) bool
}

func (f filterStream[T]) Next() (T, bool) {
	item, ok := f.stream.Next()
	for !f.predicate(item) && ok {
		item, ok = f.stream.Next()
	}
	return item, ok
}

func Map[T1 interface{}, T2 interface{}](mapFunc func(T1) T2) func(Stream[T1]) Stream[T2] {
	return func(stream Stream[T1]) Stream[T2] {
		return &mapStream[T1, T2]{
			stream:  stream,
			mapFunc: mapFunc,
		}
	}
}

type mapStream[T1 interface{}, T2 interface{}] struct {
	stream  Stream[T1]
	mapFunc func(T1) T2
}

func (m *mapStream[T1, T2]) Next() (T2, bool) {
	item, ok := m.stream.Next()
	return m.mapFunc(item), ok
}

func ToSlice[T interface{}]() func(Stream[T]) []T {
	return func(stream Stream[T]) []T {
		slice := make([]T, 0)
		item, ok := stream.Next()
		for ok {
			slice = append(slice, item)
			item, ok = stream.Next()
		}
		return slice
	}
}

func defaultValue[T interface{}]() T {
	var val T
	if typ := reflect.TypeOf(val); typ.Kind() == reflect.Ptr {
		elem := typ.Elem()
		val = reflect.New(elem).Interface().(T)
	}
	return val
}
