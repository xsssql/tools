package main

import (
	"fmt"
	"github.com/xsssql/tools/tools"
)

func main() {
	_, resp := tools.HttpUrl("https://baidu.com/", "POST", nil, "", "User-Agent: GoClient/1.0", true, "", 60, 0, false)
	fmt.Printf("%+v\n", resp)

}
