package json_template

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

const codeSample1 = `result=%%{"x":null}%% 
	if args.x
		result.x = args.x 
	end
`

func checkExecuteRes(res interface{}, expect string) error {
	var v1, v2 interface{}
	jv1, err := json.Marshal(res)
	if err != nil {
		return err
	}
	err = json.Unmarshal(jv1, &v1)
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(expect), &v2)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(v1, v2) {
		return fmt.Errorf("res=%s", string(jv1))
	}
	return nil
}

func TestTemplate1(t *testing.T) {
	tml, err := ParseTemplate(nil, codeSample1)
	if err != nil {
		t.Fatal(err)
	}

	res, err := tml.Execute(nil)
	if err != nil {
		t.Fatal(err)
	}
	err = checkExecuteRes(res, `{"x":null}`)
	if err != nil {
		t.Fatal(err)
	}

	res, err = tml.Execute(`{"x":2}`)
	if err != nil {
		t.Fatal(err)
	}
	err = checkExecuteRes(res, `{"x":null}`)
	if err != nil {
		t.Fatal(err)
	}

	res, err = tml.Execute(json.RawMessage(`{"x":[1,2,3]}`))
	if err != nil {
		t.Fatal(err)
	}
	err = checkExecuteRes(res, `{"x":[1,2,3]}`)
	if err != nil {
		t.Fatal(err)
	}

	res, err = tml.Execute(map[string]string{"x": "y"})
	if err != nil {
		t.Fatal(err)
	}
	err = checkExecuteRes(res, `{"x":"y"}`)
	if err != nil {
		t.Fatal(err)
	}

	var args struct {
		X int `json:"x"`
	}
	args.X = 123
	res, err = tml.Execute(args)
	if err != nil {
		t.Fatal(err)
	}
	err = checkExecuteRes(res, `{"x":123}`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTemplateBoolFunc(t *testing.T) {
	code := `
	if or(args.x, args.y)
		result.or = args.x
    end
	result.and = and(args.x, args.y)
	result.not = not(args.x)
	`
	tml, err := ParseTemplate(nil, code)
	if err != nil {
		t.Fatal(err)
	}

	res, err := tml.Execute(map[string]interface{}{"x": "", "y": false})
	if err != nil {
		t.Fatal(err)
	}
	err = checkExecuteRes(res, `{"and":false, "not": true}`)
	if err != nil {
		t.Fatal(err)
	}

	res, err = tml.Execute(map[string]interface{}{"y": 1})
	if err != nil {
		t.Fatal(err)
	}
	err = checkExecuteRes(res, `{"or":null,"and":false, "not": true}`)
	if err != nil {
		t.Fatal(err)
	}

	res, err = tml.Execute(map[string]interface{}{"x": 1, "y": "1"})
	if err != nil {
		t.Fatal(err)
	}
	err = checkExecuteRes(res, `{"or":1,"and":true, "not": false}`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTemplateIfArgs(t *testing.T) {
	code := `result=0 if args.x result=1 end`
	tml, err := ParseTemplate(nil, code)
	if err != nil {
		t.Fatal(err)
	}
	res, err := tml.Execute(nil)
	if err != nil {
		t.Fatal(err)
	}
	err = checkExecuteRes(res, `0`)
	if err != nil {
		t.Fatal(err)
	}

	res, err = tml.Execute(map[string]interface{}{"x": ""})
	if err != nil {
		t.Fatal(err)
	}
	err = checkExecuteRes(res, `0`)
	if err != nil {
		t.Fatal(err)
	}

	res, err = tml.Execute(map[string]interface{}{"x": "xx"})
	if err != nil {
		t.Fatal(err)
	}
	err = checkExecuteRes(res, `1`)
	if err != nil {
		t.Fatal(err)
	}

	res, err = tml.Execute(map[string]interface{}{"x": []int{}})
	if err != nil {
		t.Fatal(err)
	}
	err = checkExecuteRes(res, `0`)
	if err != nil {
		t.Fatal(err)
	}

	res, err = tml.Execute(map[string]interface{}{"x": []int{0}})
	if err != nil {
		t.Fatal(err)
	}
	err = checkExecuteRes(res, `1`)
	if err != nil {
		t.Fatal(err)
	}

	res, err = tml.Execute(map[string]interface{}{"x": json.RawMessage(` [ ] `)})
	if err != nil {
		t.Fatal(err)
	}
	err = checkExecuteRes(res, `0`)
	if err != nil {
		t.Fatal(err)
	}

	res, err = tml.Execute(map[string]interface{}{"x": json.RawMessage(`[false]`)})
	if err != nil {
		t.Fatal(err)
	}
	err = checkExecuteRes(res, `1`)
	if err != nil {
		t.Fatal(err)
	}
}
