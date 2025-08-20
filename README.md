# tools

用法:

先拉取模块

`go get github.com/xsssql/tools/tools`

然后添加模块:
```
package main

import (
"fmt"
"github.com/xsssql/tools/tools"
)

func main() {
Test()
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

```
go语言开发常用函数，HTTP/https请求，通用类型转换

#支持以下常用功能

HttpUrl HTTP请求网页函数，支持HTTP2/HTTP1.1，下载文件默认最大支持200M

// CreateDir 创建一个目录（如果父级目录不存在也会一并创建）

// ToStr 将任意类型转换为 string，无法转换时返回 "" 不适用高标准环境

//ToStrErr 将任意常见类型转为 string，支持 nil 检查 转换失败返回错误

// ToBytes 将任意类型转换为 []byte，无法转换时返回空切片

// ToBytesErr 将任意类型转换为 []byte，转换失败返回错误

// ToInt64 将任意类型转换为 int64，无法转换时或超出范围时返回 0

// ToInt64WithErr 将任意类型转换为 int64，无法转换或超出范围时返回错误

// ToInt 将任意类型转换为 int，无法转换时返回 0

// ToIntErr 将任意类型转换为 int，无法转换或溢出时返回错误

// ToUint32 将任意类型转换为 uint32，无法转换或溢出时返回 0

// ToUint32Err 将任意类型转换为 uint32，无法转换或溢出时返回错误

// ToFloat64 将任意类型转换为 float64，无法转换时返回 0

// ToFloat64Err 将任意类型转换为 float64，无法转换时返回错误

// ToFloat 将任意类型转换为 float32，无法转换时返回 0

// ToFloatErr 将任意类型转换为 float32，无法转换时返回错误

// ToUint64 将任意类型转换为 uint64，无法转换或负数时返回 0

// ToUint64Err 将任意类型转换为 uint64，无法转换或为负数时返回错误

// HexStringToBytes 将十六进制字符串转换为 []byte

// BytesToHexString 将 []byte 转换为十六进制字符串，默认大写，每字节之间可加空格

// Base64Convert 通用 Base64 编码/解码函数

// GetFormatTimeByParam 根据参数截取时间

// TimeStampToStr 将时间戳转换为字符串格式

// TimeToTimeStamp 将时间转换为10位或13位时间戳

// GetRunPath 获取当前cmd终端目录或当前程序运行目录

// FileExists 判断文件是否存在 返回true表示存在

// DeleteFileOrDir 删除文件或整个目录（包含内容）

// GetLeftOfUnderscore 取文本左边

// GetRightOfSeparator 取文本右边

// GetMiddleOfSeparator 取文本中间

// CopyDirOrFile 拷贝文件或目录

// ReadFile 读取指定路径的文件内容，返回字节切片和错误信息

// WriteBytesToFile 将 byte 数据写入到指定文件

// CSVStringsToLine 将字符串切片转换为一行 CSV 格式的 []byte

// ListAllFilesByModTime 遍历目录下的所有文件，并按修改时间升序排序，支持通配符/后缀名过滤

// FilterFileName 根据关键字过滤文件名

// GB2312ToUtf8 检测文件编码并将 GB2312/GBK 文件转换为 UTF-8

//FileToUTF8 将文件转为 UTF-8 编码，支持多种常见编码
支持简体中文：GB2312 / GBK（simplifiedchinese.GBK）

繁体中文：Big5（traditionalchinese.Big5）

日文：Shift-JIS（japanese.ShiftJIS）
韩文：EUC-KR（korean.EUCKR）

西欧：ISO-8859-1（Latin1，charmap.ISO8859_1）

Windows 系列：Windows-1252（charmap.Windows1252）

// EncodeConvert 通用编码转换函数
// 功能说明：
//
//	将输入的数据（字符串或字节切片）从源编码转换为 UTF-8 编码。
//	支持自动检测源编码，也可手动指定源编码。
//	常见支持的编码包括：UTF-8、UTF-16LE/BE、GBK/GB2312、BIG5、Shift-JIS、EUC-JP、EUC-KR、ISO-8859-1~16、Windows-125x 等。

// HTMLConvert 通用 HTML 编码/解码函数

// CSVFieldMapper 自动识别和映射 CSV 表头及行数据，提供CSV字段名称直接去出字段对应的值,如果CSV 文件中每个字段是 "" 引号引起来的则需要 使用CleanStrings() 先去除首位的引号


// URLEncodeDecode 进行 URL 编码或解码

// WriteToStingFile 将 []string数组写到文件，一般用于写出CSV文件比较方便,CSV文件每行组装后直接放到[]string 数组里面,最后统一写出

// WriteToByteFile 将字节切片写入到指定文件

// CleanStrings 处理字符串切片：去除首尾双引号和空格 一般用于处理csv 行分割后的数据

// GetTextTwoMiddle 取两段文本中间

// 示例2：
//
//	a := "你好aa你好测试你好aa你好测试你好MM，你好HelloWord你好对"

//	num, c := tools.GetTextTwoMiddle(a, "你好", "", 0, "MM", false)

//	fmt.Println(num, c)//输出57 HelloWord你好对


// GetTextTwoMiddleBytes 取两段文本中间 (byte 版本)

// 示例1：
//
//	a := "你好aa你好测试你好aa你好测试你好MM，你好HelloWord你好对"

//	numByte, cByte := tools.GetTextTwoMiddleBytes([]byte(a), []byte("你好"), []byte("你好"), 0, []byte("a"), false)

//	fmt.Println(numByte, cByte)//输出 [230 181 139 232 175 149]  ="测试"

//	fmt.Println(numByte, tools.ToStr(cByte))//输出 "测试"