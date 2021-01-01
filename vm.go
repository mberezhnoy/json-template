package json_template

import (
	"fmt"
	"reflect"
)

type vm struct {
	data      [2][]reflect.Value
	functions []reflect.Value
	code      []vmCmd
	ptr       int
}

type vmCmdType int

const (
	vmCmdCall vmCmdType = iota
	vmCmdJmp
	vmCmdJmpIfEmpty
	vmCmdJmpIfNotEmpty

	//virtual cmd: used before build final code
	opCmdLabel
	opCmdTmpVarFree
	opCmdConst
)

var vmCmdTypeNames = []string{"call", "jmp", "jmpIfEmpty", "kmpIfNotEmpty", "label", "tmpVarFree", "const"}

func (t vmCmdType) String() string {
	if t >= 0 && int(t) < len(vmCmdTypeNames) {
		return vmCmdTypeNames[t]
	}
	return fmt.Sprintf("Cmd#%d", t)
}

type vmCmd struct {
	cmd     vmCmdType
	target  int
	fn      int
	fnArgs  []vmFnArg
	codePos Position
}

type vmFnArg struct {
	isVar, dataId int
}

func (v *vm) run() (interface{}, error) {
	for v.ptr < len(v.code) {
		err := v.doCmd()
		if err != nil {
			return nil, RuntimeError{
				Err: err,
				Pos: v.code[v.ptr].codePos,
			}
		}
	}
	return v.data[1][0].Interface(), nil
}

func (v *vm) doCmd() error {
	cmd := v.code[v.ptr]
	switch cmd.cmd {
	case vmCmdCall:
		err := v.cmdCall(cmd)
		if err != nil {
			return err
		}
	case vmCmdJmp:
		v.ptr = cmd.target
		return nil
	case vmCmdJmpIfEmpty:
		vPtr := cmd.fnArgs[0]
		if isEmpty(v.data[vPtr.isVar][vPtr.dataId]) {
			v.ptr = cmd.target
			return nil
		}
	case vmCmdJmpIfNotEmpty:
		vPtr := cmd.fnArgs[0]
		if !isEmpty(v.data[vPtr.isVar][vPtr.dataId]) {
			v.ptr = cmd.target
			return nil
		}
	}

	v.ptr++
	return nil
}

func (v *vm) cmdCall(cmd vmCmd) error {
	fn := v.functions[cmd.fn]
	args := make([]reflect.Value, len(cmd.fnArgs))
	for i, ptr := range cmd.fnArgs {
		args[i] = v.data[ptr.isVar][ptr.dataId]
	}
	res, err := safeCall(fn, args)
	if err != nil {
		return err
	}
	v.data[1][cmd.target] = res
	return nil
}

// safeCall runs fun.Call(args), and returns the resulting value and error, if
// any. If the call panics, the panic value is returned as an error.
func safeCall(fun reflect.Value, args []reflect.Value) (val reflect.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", r)
			}
		}
	}()
	ret := fun.Call(args)
	if len(ret) == 2 && !ret[1].IsNil() {
		return ret[0], ret[1].Interface().(error)
	}
	return ret[0], nil
}

func isEmpty(val reflect.Value) bool {
	//todo: correct work with json.RawMessage & struct
	if !val.IsValid() {
		return true
	}
	switch val.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return val.Len() == 0
	case reflect.Bool:
		return !val.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return val.Int() == 0
	case reflect.Float32, reflect.Float64:
		return val.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return val.Uint() == 0
	}
	return false
}
