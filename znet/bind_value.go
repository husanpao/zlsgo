package znet

import (
	"reflect"

	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/zreflect"
	"github.com/sohaha/zlsgo/ztype"
)

func (c *Context) Bind(obj interface{}) (err error) {
	method := c.Request.Method
	if method == "GET" {
		return c.BindQuery(obj)
	}
	contentType := c.ContentType()
	if contentType == c.ContentType(ContentTypeJSON) {
		return c.BindJSON(obj)
	}
	return c.BindForm(obj)
}

func (c *Context) BindJSON(obj interface{}) error {
	body, err := c.GetDataRaw()
	if err != nil {
		return err
	}
	return zjson.Unmarshal(body, obj)
}

func (c *Context) BindQuery(obj interface{}) (err error) {
	q := c.GetAllQueryMaps()
	typ := zreflect.TypeOf(obj)
	m := make(map[string]interface{}, len(q))
	err = zreflect.ForEach(typ, func(parent []string, index int, tag string, field reflect.StructField) error {
		kind := field.Type.Kind()
		if kind == reflect.Struct {
			m[tag] = c.QueryMap(tag)
		} else if kind == reflect.Slice {
			v, _ := c.GetQueryArray(tag)
			m[tag] = v
		} else {
			v, ok := q[tag]
			if ok {
				m[tag] = v
			}
		}

		return zreflect.SkipChild
	})
	if err != nil {
		return err
	}
	return ztype.ToStruct(m, obj)
}

func (c *Context) BindForm(obj interface{}) error {
	q := c.GetPostFormAll()
	typ := zreflect.TypeOf(obj)
	m := make(map[string]interface{}, len(q))
	err := zreflect.ForEach(typ, func(parent []string, index int, tag string, field reflect.StructField) error {
		kind := field.Type.Kind()
		if kind == reflect.Struct {
			m[tag] = c.PostFormMap(tag)
		} else if kind == reflect.Slice {
			sliceTyp := field.Type.Elem().Kind()
			if sliceTyp == reflect.Struct {
				// TODO follow up support
			} else {
				m[tag], _ = q[tag]
			}
		} else {
			v, ok := q[tag]
			if ok {
				m[tag] = v[0]
			}
		}

		return zreflect.SkipChild
	})
	if err != nil {
		return err
	}
	return ztype.ToStruct(m, obj)
}
