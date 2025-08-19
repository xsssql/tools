package main

import (
	"fmt"
	"tools/tools"
)

func main() {

}

func Test() {
	a := "你好aa你好测试你好aa你好测试你好MM，你好HelloWord你好对"

	num, c := tools.GetTextTwoMiddle(a, "你好", "你好", 0, "a", false)
	fmt.Println(num, c)
	fmt.Println(a[num:])

	numByte, cByte := tools.GetTextTwoMiddleBytes([]byte(a), []byte("你好"), []byte("你好"), 0, []byte("a"), false)
	fmt.Println(numByte, cByte)
	fmt.Println(numByte, tools.ToStr(cByte))
	fmt.Println(a[numByte:])

	mapping := make(map[string]int)
	// 传入表头
	tools.CSVFieldMapper([]string{"id", "name", "age"}, &mapping, true, "")
	// mapping 结果: map["id"]=0, map["name"]=1, map["age"]=2

	// 传入行数据并取值
	row := []string{"1", "Tom", "18"}
	val, _ := tools.CSVFieldMapper(row, &mapping, false, "name")
	fmt.Println(val) // 输出: Tom
	head := `Sec-Ch-Ua-Arch: "x86"
Accept-Encoding: gzip
Cookie: PHPSESSION=55523704asafsdffg0
User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36
Connection: keep-alive
`
	_, respHeader, respBody := tools.HttpUrl("https://baidu.com", "POST", []byte("a=1&b=2"), "PHPSESSION=AAAAA", head, true, "http://127.0.0.1:8080", 60, 0)
	fmt.Println("=== 返回协议头 ===", respHeader)
	fmt.Println("=== 响应内容 ===", string(respBody))
}
