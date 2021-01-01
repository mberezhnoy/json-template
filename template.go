package json_template

import (
	"encoding/json"
	"reflect"
	"text/template"
)

type Options struct {
	functions map[string]reflect.Value
	constants map[string]interface{}
	prototype interface{}
	strTml    map[string]string
	strFunc   template.FuncMap
}

type Template struct {
	functions   []reflect.Value
	constData   []reflect.Value
	varDataSize int
	code        []vmCmd
}

func ParseTemplate(deps *Options, code string) (*Template, error) {
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

func NewOptions() *Options {
	return &Options{
		constants: map[string]interface{}{},
		functions: map[string]reflect.Value{},
		strTml:    map[string]string{},
	}
}

func (o *Options) Prototype(v interface{}) *Options {
	o.prototype = v
	return o
}

var reservedKeywords = map[string]bool{
	"result": true,
	"args":   true,
	"if":     true,
	"else":   true,
	"end":    true,
	"for":    true,
	"in":     true,
}

func (o *Options) checkName(name string) error {
	if !rxName.MatchString(name) {
		return ErrIncorrectName
	}
	if reservedKeywords[name] {
		return ErrIncorrectName
	}
	return nil
}

func (o *Options) Const(name string, v interface{}) error {
	err := o.checkName(name)
	if err != nil {
		return err
	}
	o.constants[name] = v
	return nil
}

func (o *Options) Func(name string, v interface{}) error {
	err := o.checkName(name)
	if err != nil {
		return err
	}

	//check: v is function
	rFn := reflect.ValueOf(v)
	if rFn.Kind() != reflect.Func {
		return ErrNotFunction
	}
	if rFn.IsNil() {
		return ErrNotFunction
	}

	//check signature:
	//support /func(...) someType/ or /func(...) (someType, error)/
	tFn := rFn.Type()
	if tFn.NumOut() == 0 || tFn.NumOut() > 2 {
		return ErrIncorrectFunction
	}
	if tFn.NumOut() == 2 {
		if tFn.Out(1).String() != "error" {
			return ErrIncorrectFunction
		}
	}

	o.functions[name] = rFn
	return nil
}

func (o *Options) StringTemplate(name string, tml string) error {
	err := o.checkName(name)
	if err != nil {
		return err
	}
	o.strTml[name] = tml
	return nil
}

func (o *Options) StringFunctions(funcMap template.FuncMap) *Options {
	o.strFunc = funcMap
	return o
}
