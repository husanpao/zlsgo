package ztype

import (
	"encoding/json"
	"reflect"

	"github.com/sohaha/zlsgo/zreflect"
)

type SliceType []Type

func (s SliceType) Len() int {
	return len(s)
}

func (s SliceType) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Value())
}

func (s SliceType) Index(i int) Type {
	if i < 0 || i >= len(s) {
		return Type{}
	}
	return s[i]
}

func (s SliceType) Last() Type {
	return s.Index(len(s) - 1)
}

func (s SliceType) First() Type {
	return s.Index(0)
}

func (s SliceType) Value() []interface{} {
	ss := make([]interface{}, 0, len(s))
	for i := range s {
		ss = append(ss, s[i].Value())
	}
	return ss
}

func (s SliceType) String() []string {
	ss := make([]string, 0, len(s))
	for i := range s {
		ss = append(ss, s[i].String())
	}
	return ss
}

func (s SliceType) Int() []int {
	ss := make([]int, 0, len(s))
	for i := range s {
		ss = append(ss, s[i].Int())
	}
	return ss
}

func (s SliceType) Maps() Maps {
	ss := make(Maps, 0, len(s))
	for i := range s {
		ss = append(ss, s[i].Map())
	}
	return ss
}

// func (s SliceType) Slice() []SliceType {
// 	ss := make([]SliceType, 0, len(s))
// 	for i := range s {
// 		ss = append(ss, s[i].Slice())
// 	}
// 	return ss
// }

// Deprecated: please use ToSlice
func Slice(value interface{}, noConv ...bool) SliceType {
	return ToSlice(value, noConv...)
}

// SliceStrToAny  []string to []interface{}
func SliceStrToAny(slice []string) []interface{} {
	ss := make([]interface{}, 0, len(slice))
	for _, val := range slice {
		ss = append(ss, val)
	}
	return ss
}

func ToSlice(value interface{}, noConv ...bool) (s SliceType) {
	s = SliceType{}
	if value == nil {
		return
	}

	switch val := value.(type) {
	case []interface{}:
		s = make(SliceType, 0, len(val))
		for i := range val {
			s = append(s, New(val[i]))
		}
	case []string:
		s = make(SliceType, 0, len(val))
		for i := range val {
			s = append(s, New(val[i]))
		}
	case []int:
		s = make(SliceType, 0, len(val))
		for i := range val {
			s = append(s, New(val[i]))
		}
	case []int64:
		s = make(SliceType, 0, len(val))
		for i := range val {
			s = append(s, New(val[i]))
		}
	default:
		var nval []interface{}
		vof := zreflect.ValueOf(&nval)
		to := func() {
			if conv.to("", value, vof) == nil {
				s = make(SliceType, 0, len(nval))
				for i := range nval {
					s = append(s, New(nval[i]))
				}
			}
		}

		switch vof.Type().Kind() {
		case reflect.Slice:
			to()
		default:
			if len(noConv) > 0 && noConv[0] {
				return
			}
			to()
		}
	}

	return
}
