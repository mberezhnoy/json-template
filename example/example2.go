package main

import (
	"encoding/json"
	"fmt"
	"github.com/mberezhnoy/json-template"
	"time"
)

func main() {
	opt := json_template.NewOptions()
	opt.Prototype(json.RawMessage(`{
		"filters": {}
	}`))
	opt.Const("lt", json.RawMessage(`{"lt":""}`))
	opt.Func("less", func(s1, s2 interface{}) bool {
		str1, _ := s1.(string)
		str2, _ := s2.(string)
		return str1 < str2
	})
	opt.Func("addMonth", func(date interface{}) (string, error) {
		dateStr, _ := date.(string)
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return "", err
		}
		t = t.AddDate(0, 1, 0)
		return t.Format("2006-01-02"), nil
	})

	code := `date = args.from
		for less(date, args.to)
			filter = lt
			filter.lt = date
			result.filters[date]=filter
			date = addMonth(date)
		end
	`
	t, err := json_template.ParseTemplate(opt, code)
	if err != nil {
		panic(err)
	}
	res, err := t.Execute(map[string]string{
		"from": "2020-01-01",
		"to":   "2021-01-01",
	})
	if err != nil {
		panic(err)
	}
	d, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(d))
}
