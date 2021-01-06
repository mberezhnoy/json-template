package json_template

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"text/template"
)

var buildInFunctions = map[string]reflect.Value{}

func init() {
	buildInFunctions["@initIteratorK"] = reflect.ValueOf(initIteratorK)
	buildInFunctions["@initIteratorV"] = reflect.ValueOf(initIteratorV)
	buildInFunctions["@initIteratorKV"] = reflect.ValueOf(initIteratorKV)
	buildInFunctions["@iteratorStep"] = reflect.ValueOf(iteratorStep)
	buildInFunctions["@iteratorKey"] = reflect.ValueOf(iteratorKey)
	buildInFunctions["@iteratorVal"] = reflect.ValueOf(iteratorValue)
	buildInFunctions["@strTemplate"] = reflect.ValueOf(strTemplate)
	buildInFunctions["@get"] = reflect.ValueOf(jsonGet)
	buildInFunctions["@jsonSet"] = reflect.ValueOf(jsonSet)
	buildInFunctions["@append"] = reflect.ValueOf(jsonAppend)
	buildInFunctions["@clone"] = reflect.ValueOf(clone)

	buildInFunctions["eq"] = reflect.ValueOf(eq)
	buildInFunctions["sum"] = reflect.ValueOf(sum)

	buildInFunctions["or"] = reflect.ValueOf(or)
	buildInFunctions["and"] = reflect.ValueOf(and)
	buildInFunctions["not"] = reflect.ValueOf(not)
}

func clone(v interface{}) (interface{}, error) {
	switch v.(type) {
	case int, string, float64, json.RawMessage, bool, nil:
		return v, nil
	}

	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func strTemplate(t *template.Template, params interface{}) (string, error) {
	buf := bytes.Buffer{}
	err := t.Execute(&buf, params)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func jsonGet(val interface{}, path ...interface{}) (interface{}, error) {
	if len(path) == 0 {
		return val, nil
	}
	switch tv := val.(type) {
	case nil, string, float64, int, bool:
		return nil, nil
	case map[string]interface{}:
		key, err := jsonStringKey(path[0])
		if err != nil {
			return nil, err
		}
		return jsonGet(tv[key], path[1:]...)
	case []interface{}:
		key, valid, err := jsonIntKey(path[0])
		if err != nil {
			return nil, err
		}
		if !valid || key < 0 || key >= len(tv) {
			return nil, nil
		}
		return jsonGet(tv[key], path[1:]...)
	}

	//todo: optimization
	d, err := json.Marshal(val)
	if err != nil {
		return nil, err
	}

	var v interface{}
	err = json.Unmarshal(d, &v)
	if err != nil {
		return nil, err
	}
	return jsonGet(v, path...)
}

func jsonStringKey(v interface{}) (string, error) {
	switch tv := v.(type) {
	case string:
		return tv, nil
	case json.RawMessage:
		var v2 interface{}
		err := json.Unmarshal(tv, &v2)
		if err != nil {
			return "", err
		}
		return jsonStringKey(v2)
	case []byte:
		return string(tv), nil
	case fmt.Stringer:
		return tv.String(), nil
	}
	return fmt.Sprint(v), nil
}

func jsonIntKey(v interface{}) (int, bool, error) {
	switch tv := v.(type) {
	case int:
		return tv, true, nil
	case float64:
		key := math.Round(tv)
		if math.Abs(key-tv) < 0.001 {
			return int(key), true, nil
		}
		return 0, false, nil
	}
	strKey, err := jsonStringKey(v)
	if err != nil {
		return 0, false, err
	}
	key, err := strconv.Atoi(strKey)
	if err != nil {
		return 0, false, nil
	}
	return key, true, nil
}

func jsonSet(data, val interface{}, path ...interface{}) (interface{}, error) {
	if len(path) == 0 {
		return val, nil
	}
	switch vData := data.(type) {
	case nil, string, float64, int, bool:
		return jsonNew(val, path...)
	case map[string]interface{}:
		key, err := jsonStringKey(path[0])
		if err != nil {
			return nil, err
		}
		v, err := jsonSet(vData[key], val, path[1:]...)
		if err != nil {
			return nil, err
		}
		vData[key] = v
		return vData, nil
	case []interface{}:
		key, valid, err := jsonIntKey(path[0])
		if err != nil {
			return nil, err
		}
		if !valid {
			return nil, fmt.Errorf("can`t use `%v` as array index", path[0])
		}
		if key >= len(vData) {
			v, err := jsonNew(val, path[1:]...)
			if err != nil {
				return nil, err
			}
			extend := make([]interface{}, len(vData)+1-key)
			extend[len(extend)-1] = v
			return append(vData, extend...), nil
		}
		if key >= -len(vData) {
			if key < 0 {
				key = len(vData) + key
			}
			v, err := jsonSet(vData[key], val, path[1:]...)
			if err != nil {
				return nil, err
			}
			vData[key] = v
			return vData, nil
		}

		v, err := jsonNew(val, path[1:]...)
		if err != nil {
			return nil, err
		}
		prepend := make([]interface{}, -len(vData)-key)
		prepend[0] = v
		return append(prepend, vData...), nil
	}

	//todo: optimization
	d, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var v interface{}
	err = json.Unmarshal(d, &v)
	if err != nil {
		return nil, err
	}
	return jsonSet(v, val, path...)
}

func jsonNew(val interface{}, path ...interface{}) (interface{}, error) {
	if len(path) == 0 {
		return val, nil
	}
	switch path[0].(type) {
	case int, float64:
		key, valid, err := jsonIntKey(path[0])
		if err != nil {
			return nil, err
		}
		if valid && key >= 0 {
			data := make([]interface{}, key+1)
			data[key], err = jsonNew(val, path[1:]...)
			if err != nil {
				return nil, err
			}
			return data, nil
		}
	}

	key, err := jsonStringKey(path[0])
	if err != nil {
		return nil, err
	}
	data, err := jsonNew(val, path[1:]...)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{key: data}, nil
}

func jsonAppend(data, val interface{}, path ...interface{}) (interface{}, error) {
	if len(path) == 0 {
		return jsonAppendCur(data, val)
	}
	lastNode, err := jsonGet(data, path...)
	if err != nil {
		return nil, err
	}
	lastNode, err = jsonAppendCur(lastNode, val)
	if err != nil {
		return nil, err
	}
	data, err = jsonSet(data, lastNode, path...)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func jsonAppendCur(data, val interface{}) (interface{}, error) {
	switch tv := data.(type) {
	case nil, string, float64, int, bool:
		return []interface{}{val}, nil
	case map[string]interface{}:
		i := 0
		_, isSet := tv[strconv.Itoa(i)]
		for isSet {
			i++
			_, isSet = tv[strconv.Itoa(i)]
		}
		tv[strconv.Itoa(i)] = val
		return tv, nil
	case []interface{}:
		return append(tv, val), nil
	}

	d, err := json.Marshal(val)
	if err != nil {
		return nil, err
	}

	var v interface{}
	err = json.Unmarshal(d, &v)

	return jsonAppendCur(v, val)
}

type iterator struct {
	withKey, withVal bool
	cur, len         int
	keys             []interface{}
	values           []interface{}
}

func (i *iterator) init(data interface{}) error {
	i.cur = -1
	i.len = 0
	rm, ok := data.(json.RawMessage)
	if ok {
		var v interface{}
		err := json.Unmarshal(rm, &v)
		if err != nil {
			return err
		}
		return i.init(v)
	}
	rv := reflect.ValueOf(data)
	switch rv.Kind() {
	case reflect.Struct:
		d, err := json.Marshal(data)
		if err != nil {
			return err
		}
		var v interface{}
		err = json.Unmarshal(d, &v)
		if err != nil {
			return err
		}
		return i.init(v)
	case reflect.Slice, reflect.Array:
		i.len = rv.Len()
		for key := 0; key < i.len; key++ {
			if i.withKey {
				i.keys = append(i.keys, key)
			}
			if i.withVal {
				i.values = append(i.values, rv.Index(key).Interface())
			}
		}
	case reflect.Map:
		rKeys := rv.MapKeys()
		i.len = rv.Len()
		for _, rKey := range rKeys {
			if i.withKey {
				i.keys = append(i.keys, rKey.Interface())
			}
			if i.withVal {
				i.values = append(i.values, rv.MapIndex(rKey).Interface())
			}
		}
	case reflect.Chan:
		errors.New("foreach by chan not supported")
	default:
		return nil
	}
	return nil
}

func initIteratorK(data interface{}) (*iterator, error) {
	i := &iterator{
		withKey: true,
	}
	err := i.init(data)
	return i, err
}

func initIteratorV(data interface{}) (*iterator, error) {
	i := &iterator{
		withVal: true,
	}
	err := i.init(data)
	return i, err
}

func initIteratorKV(data interface{}) (*iterator, error) {
	i := &iterator{
		withKey: true,
		withVal: true,
	}
	err := i.init(data)
	return i, err
}

func iteratorStep(i *iterator) bool {
	i.cur++
	return i.cur < i.len
}

func iteratorKey(i *iterator) interface{} {
	if !i.withKey {
		return nil
	}
	return i.keys[i.cur]
}

func iteratorValue(i *iterator) interface{} {
	if !i.withVal {
		return nil
	}
	return i.values[i.cur]
}

func eq(v1, v2 interface{}) (bool, error) {
	var err error
	jr, ok := v1.(json.RawMessage)
	if ok {
		err = json.Unmarshal(jr, &v1)
		if err != nil {
			return false, err
		}
	}
	jr, ok = v2.(json.RawMessage)
	if ok {
		err := json.Unmarshal(jr, &v2)
		if err != nil {
			return false, err
		}
	}
	rv1 := reflect.ValueOf(v1)
	rv2 := reflect.ValueOf(v2)
	if rv1.Type() == rv2.Type() {
		return reflect.DeepEqual(rv1, rv2), nil
	}

	err = marshalUnmarshal(v1, &v1)
	if err != nil {
		return false, err
	}
	err = marshalUnmarshal(v2, &v2)
	if err != nil {
		return false, err
	}
	return reflect.DeepEqual(rv1, rv2), nil
}

func marshalUnmarshal(in interface{}, out interface{}) error {
	data, err := json.Marshal(in)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &out)
	if err != nil {
		return err
	}
	return nil
}

func sum(v1, v2 interface{}) (interface{}, error) {
	var iv1, iv2 int
	allInt := true
	iv1, allInt = v1.(int)
	if allInt {
		iv2, allInt = v2.(int)
	}
	if allInt {
		return iv1 + iv2, nil
	}

	var fv1, fv2 float64
	allFloat := true
	fv1, allFloat = v1.(float64)
	if allFloat {
		fv2, allFloat = v2.(float64)
	}
	if allFloat {
		return fv1 + fv2, nil
	}

	err := marshalUnmarshal(v1, &iv1)
	if err == nil {
		err := marshalUnmarshal(v2, &iv2)
		if err == nil {
			return iv1 + iv2, nil
		}
	}

	err = marshalUnmarshal(v1, &fv1)
	if err != nil {
		return 0, fmt.Errorf("first argument is not numeric")
	}
	err = marshalUnmarshal(v2, &fv2)
	if err != nil {
		return 0, fmt.Errorf("second argument is not numeric")
	}
	return fv1 + fv2, nil
}

func or(list ...interface{}) bool {
	for _, v := range list {
		if !isEmpty(reflect.ValueOf(v)) {
			return true
		}
	}
	return false
}

func and(list ...interface{}) bool {
	for _, v := range list {
		if isEmpty(reflect.ValueOf(v)) {
			return false
		}
	}
	return true
}

func not(v interface{}) bool {
	return isEmpty(reflect.ValueOf(v))
}
