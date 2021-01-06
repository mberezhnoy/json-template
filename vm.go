package json_template

import (
	"bytes"
	"encoding/json"
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

var nilVal reflect.Value

func isEmpty(val reflect.Value) bool {
	if !val.IsValid() {
		return true
	}
	for val.Kind() == reflect.Interface {
		val = val.Elem()
		if val == nilVal {
			return true
		}
	}

	switch val.Kind() {
	case reflect.Slice:
		if val.Len() == 0 {
			return true
		}
		jrm, ok := val.Interface().(json.RawMessage)
		if ok {
			return isEmptyJson(jrm)
		}
		return false
	case reflect.Map:
		return val.Len() == 0
	}

	return val.IsZero()
}

func isEmptyJson(msg json.RawMessage) bool {
	data := bytes.TrimSpace(msg)
	switch string(data[0]) {
	case `null`, `false`, `0`, `""`:
		return true
	}
	if (data[0] == '{' && data[len(data)-1] == '}') ||
		(data[0] == '[' && data[len(data)-1] == ']') {
		data = bytes.TrimSpace(data[1 : len(data)-1])
	}
	return len(data) == 0
}
