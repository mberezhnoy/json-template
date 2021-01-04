package main

import (
	"encoding/json"
	"fmt"
	"github.org/mberezhnoy/json-template"
)

func main() {
	tml := `
	result = %%{
			"obj":{}, 
			"arr":[], 
			"info": "test template"
		}%%
	for k v in args
		result.obj[k] = v
		result.arr[] = v
	end
	`
	t, err := json_template.ParseTemplate(nil, tml)
	if err != nil {
		panic(err)
	}
	res, err := t.Execute(map[string]string{"k1": "v1", "k2": "v2"})
	if err != nil {
		panic(err)
	}
	d, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(d))
}
