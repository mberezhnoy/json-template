package json_template

import (
	"encoding/json"
	"reflect"
	"text/template"
)

type TemplateDeps struct {
	Func      map[string]interface{}
	Const     map[string]interface{}
	Prototype interface{}
	StrTml    map[string]string
	StrFunc   template.FuncMap
}

type Template struct {
	functions   []reflect.Value
	constData   []reflect.Value
	varDataSize int
	code        []vmCmd
}

func ParseTemplate(deps *TemplateDeps, code string) (*Template, error) {
	cmp := compiler{deps: deps}
	err := cmp.compile(code)
	if err != nil {
		return nil, err
	}
	t := Template{
		functions:   cmp.functions,
		constData:   cmp.constData,
		varDataSize: cmp.varDataSize,
		code:        cmp.vmCode,
	}
	return &t, nil
}

func (t *Template) Execute(params interface{}) (interface{}, error) {
	v := vm{}
	v.data[0] = t.constData
	v.data[1] = make([]reflect.Value, t.varDataSize)
	v.data[1][0] = zeroPrototype
	v.data[1][1] = reflect.ValueOf(params)
	v.functions = t.functions
	v.code = t.code
	return v.run()
}

var zeroPrototype = reflect.ValueOf(json.RawMessage(`null`))
