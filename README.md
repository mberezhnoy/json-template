# json-template

Package json-template implements templates for generating json output.

## example
```go
package main

import (
	"encoding/json"
	"fmt"
	"github.com/mberezhnoy/json-template"
)

func main() {
	code := `
		result = %%{"data":[]}%%
		for key val in args
			obj = %%{}%%
			obj.name = key
			obj.value = val
			result.data[] = obj
		end
	`
	t, err := json_template.ParseTemplate(nil, code)
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
```
output:
```json
{
  "data": [
    {
      "name": "k1",
      "value": "v1"
    },
    {
      "name": "k2",
      "value": "v2"
    }
  ]
}
```

For more examples see `example` directory.

## template language spec

#### Actions
- **set variable**
```
varName = *pipeline*
```
- **json set**
```
varName[*key1Pipeline*][*key2Pipeline*] = *pipeline*
varName.key1.key2 = *pipeline*
```
for array access keyPipeline should be representable as integer. 
Negative pipeline values mean access from end of the array.
If pipeline values outside of array length then array will be extended.

examples:
```
template: result=["a", "b", "c"] result[1] = "d"  
output: ["a", "d", "c"]
 
template: result=["a", "b", "c"] result[-1] = "d"  
output: ["a", "b", "d"]
 
template: result=["a", "b", "c"] result[5] = "d"  
template: output: ["a", "b", "c", null, null, "d"]
 
template: result=["a", "b", "c"] result[-5] = "d"  
output: ["d", null "a", "b", "c"] 
```
 
- **array append**
```
varName[] = *pipeline*
varName[*key1Pipeline*][*key2Pipeline*][] = *pipeline*
varName.key1.key2[] = *pipeline*
```
- **if**
```
if *pipeline* *actions* end
if *pipeline* *actions* else *actions* end     
```
- **for**
```
if *pipeline* *actions* end
```
- **foreach**
```
for keyVarName in *pipeline* *actions* end
for keyVarName valueVarName in *pipeline* *actions* end
for _ valueVarName in *pipeline* *actions* end
```

#### Pipeline
- **string value**
```
"abc"
"ab\"cd"
```
- **numeric value**
```
123
-345
12.34
```
- **object value**
```
%%{"key":"val"}%%

%xx%[
    "ab%%cd",
    123
]%xx%
```
- **const or var access**
```
varName
```
- **json get**
```
constName[*key1Pipeline*][*key2Pipeline*]
varName.key1.key2
```

- **function call**
```
fnName(*pipeline1*, *pipeline2*)
```

- **string template**
```
.templateName(*pipeline*)
```

#### build in functions
- **sum**
- **eq**
- **or**
- **and**
- **not**

## User defined functions
example:
```go
opt := json_template.NewOptions()
err := opt.Func("less", func(a, b int) bool {
    return a < b
})
if err != nil {
    panic(err)
}
code := `result = args[0]
if less(args[0], args[1]) result = args[1] end`
t, _ := json_template.ParseTemplate(opt, code)

fmt.Println( t.Execute([]int{1,2}))
fmt.Println( t.Execute([]int{5,3}))
fmt.Println( t.Execute([]interface{}{5,"X"}))
```
output:
```
2 <nil>
5 <nil>
<nil> [2:4] incorect arg type, expect: int got: string
```

`opt.Func(fnName, fn)` - may return error if provide incorrect function name or function with unacceptable signature

`fn` should be declared as `func(....) SomeType` or `func(....) (SomeType, error)`. 
In second case if it return error then `t.Execute` will stopped and returned this error.

## String template
example:
```go
opt := json_template.NewOptions()
opt.StringTemplate("title", "Title For {{.}}")
code := `for _ v in args
    result[v]=.title(v)
end`
t, err := json_template.ParseTemplate(opt, code)
if err != nil {
    panic(err)
}
res, err := t.Execute([]string{"a","b"})
if err != nil {
    panic(err)
}
d, _ := json.MarshalIndent(res, "", "  ")
fmt.Println(string(d))
``` 
output:
```json
{
  "a": "Title For a",
  "b": "Title For b"
}
```

`opt.StringTemplate(name, tml)` - create `text/template` and allow call it inside json template

`opt.StringFunctions(funcMap)` - https://golang.org/pkg/text/template/#Template.Funcs

## Object Declaration
example:
```go
code := `
result = %%{"query":{
    "bool": {
        "filter":[]
    } 
}}%%
for k v in args
    f = %%{"term":{}}%%
    f.term[k] = v
    result.query.bool.filter[] = f
end`
t, err := json_template.ParseTemplate(nil, code)
```
it can be overwritten as
```go
opt := json_template.NewOptions()
opt.Prototype(json.RawMessage(`{"query":{
    "bool": {
        "filter":[]
    } 
}}`))
opt.Const("term", json.RawMessage(`{"term":{}}`))

code := `for k v in args
    f = term
    f.term[k] = v
    result.query.bool.filter[] = f
end`
t, err := json_template.ParseTemplate(opt, code)
```  

`opt.Prototype(someJson)` - result will init with this value

`opt.Const(name, someJson)` - add const for use in template

This allow move object declaration outside of template, and keep in template just code for json manipulation. 
