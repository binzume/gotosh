package main

import (
	"fmt"

	"github.com/binzume/gotosh/jqjson"
)

func main() {
	data := `{"users":[{"name":"Alice","active":true},{"name":"Bob","active":false}]}`

	names, err := jqjson.Filter(data, `.users[] | select(.active) | .name`, "-r")
	fmt.Println("active users:", names)
	fmt.Println("filter error:", err)

	typ, err := jqjson.Type(data)
	fmt.Println("type:", typ)
	fmt.Println("type error:", err)

	name, err := jqjson.Get(data, `.users[0].name`)
	fmt.Println("first user:", name)
	fmt.Println("get error:", err)

	empty, err := jqjson.Build(`{"users":[{"items":[]}]}`, "-c")
	fmt.Println("empty array:", empty, err)
	withString, err := jqjson.AppendString(empty, `.users[0].items`, "Alice")
	fmt.Println("after string append:", withString, err)
	withObject, err := jqjson.Append(withString, `.users[0].items`, `{"name":"Bob"}`)
	fmt.Println("after JSON append:", withObject, err)
}
