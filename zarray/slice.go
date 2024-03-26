//go:build go1.18
// +build go1.18

package zarray

import (
	"math/rand"

	"github.com/sohaha/zlsgo/zstring"
)

// CopySlice copy a slice
func CopySlice[T any](l []T) []T {
	nl := make([]T, len(l))
	copy(nl, l)
	return nl
}

// Rand A random eents
func Rand[T any](collection []T) T {
	l := len(collection)
	if l == 0 {
		var zero T
		return zero
	}

	i := zstring.RandInt(0, l-1)
	return collection[i]
}

// Map manipulates a slice and transforms it to a slice of another type
func Map[T any, R any](collection []T, iteratee func(int, T) R) []R {
	res := make([]R, len(collection))

	for i, item := range collection {
		res[i] = iteratee(i, item)
	}

	return res
}

// Shuffle creates a slice of shuffled values
func Shuffle[T any](collection []T) []T {
	n := CopySlice(collection)
	rand.Shuffle(len(n), func(i, j int) {
		n[i], n[j] = n[j], n[i]
	})

	return n
}

// Reverse creates a slice of reversed values
func Reverse[T any](collection []T) []T {
	n := CopySlice(collection)
	l := len(n)
	for i := 0; i < l/2; i++ {
		n[i], n[l-i-1] = n[l-i-1], n[i]
	}

	return n
}

// Filter iterates over eents of collection
func Filter[T any](slice []T, predicate func(index int, item T) bool) []T {
	slice = CopySlice(slice)

	j := 0
	for i := range slice {
		if !predicate(i, slice[i]) {
			continue
		}
		slice[j] = slice[i]
		j++
	}

	return slice[:j:j]
}

// Contains returns true if an eent is present in a collection
func Contains[T comparable](collection []T, v T) bool {
	for _, item := range collection {
		if item == v {
			return true
		}
	}

	return false
}

// Find search an eent in a slice based on a predicate. It returns eent and true if eent was found.
func Find[T any](collection []T, predicate func(index int, item T) bool) (res T, ok bool) {
	for i := range collection {
		item := collection[i]
		if predicate(i, item) {
			return item, true
		}
	}

	return
}

// Unique returns a duplicate-free version of an array
func Unique[T comparable](collection []T) []T {
	repeat := make(map[T]struct{}, len(collection))

	return Filter(collection, func(_ int, item T) bool {
		if _, ok := repeat[item]; ok {
			return false
		}
		repeat[item] = struct{}{}
		return true
	})
}

func Diff[T comparable](list1 []T, list2 []T) ([]T, []T) {
	l, r := []T{}, []T{}

	rl, rr := map[T]struct{}{}, map[T]struct{}{}

	for _, e := range list1 {
		rl[e] = struct{}{}
	}

	for _, e := range list2 {
		rr[e] = struct{}{}
	}

	for _, e := range list1 {
		if _, ok := rr[e]; !ok {
			l = append(l, e)
		}
	}

	for _, e := range list2 {
		if _, ok := rl[e]; !ok {
			r = append(r, e)
		}
	}

	return l, r
}

func Pop[T comparable](list *[]T) (v T) {
	l := len(*list)
	if l == 0 {
		return
	}

	v = (*list)[l-1]
	*list = (*list)[:l-1]
	return
}

func Shift[T comparable](list *[]T) (v T) {
	l := len(*list)
	if l == 0 {
		return
	}

	v = (*list)[0]
	*list = (*list)[1:]
	return
}
