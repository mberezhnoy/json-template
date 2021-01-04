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
	err = json.Unmarshal([]byte(expect), v2)
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
	checkExecuteRes(res, `{"x":null}`)

	res, err = tml.Execute(`{"x":2}`)
	if err != nil {
		t.Fatal(err)
	}
	checkExecuteRes(res, `{"x":null}`)

	res, err = tml.Execute(json.RawMessage(`{"x":[1,2,3]}`))
	if err != nil {
		t.Fatal(err)
	}
	checkExecuteRes(res, `{"x":[1,2,3]`)

	res, err = tml.Execute(map[string]string{"x": "y"})
	if err != nil {
		t.Fatal(err)
	}
	checkExecuteRes(res, `{"x":"y"}`)

	var args struct {
		X int `json:"x"`
	}
	args.X = 123
	res, err = tml.Execute(args)
	if err != nil {
		t.Fatal(err)
	}
	checkExecuteRes(res, `{"x":123}`)
}
