package tools

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/saintfish/chardet"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"html"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CreateDir 创建一个目录（如果父级目录不存在也会一并创建）
func CreateDir(path string) error {
	// 使用 os.MkdirAll 自动递归创建目录
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}
	return nil
}

// ToStr 将任意类型转换为 string，无法转换时返回 "" 不适用高标准环境
func ToStr(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case *string:
		if val != nil {
			return *val
		}
	case []byte:
		return string(val)
	case *([]byte):
		if val != nil {
			return string(*val)
		}
	case int:
		return strconv.Itoa(val)
	case *int:
		if val != nil {
			return strconv.Itoa(*val)
		}
	case int64:
		return strconv.FormatInt(val, 10)
	case *int64:
		if val != nil {
			return strconv.FormatInt(*val, 10)
		}
	case int32:
		return strconv.Itoa(int(val))
	case *int32:
		if val != nil {
			return strconv.Itoa(int(*val))
		}
	case float64:
		return fmt.Sprintf("%f", val)
	case *float64:
		if val != nil {
			return fmt.Sprintf("%f", *val)
		}
	case float32:
		return fmt.Sprintf("%f", val)
	case *float32:
		if val != nil {
			return fmt.Sprintf("%f", *val)
		}
	case bool:
		return strconv.FormatBool(val)
	case *bool:
		if val != nil {
			return strconv.FormatBool(*val)
		}
	case json.Number:
		return val.String()
	default:
		if v == nil {
			return ""
		}
		fmt.Printf("⚡ ToStr遇到未知类型：%T -> %+v\n", v, v)
		return fmt.Sprintf("%v", v)
	}
	return ""
}

// ToStrErr 将任意常见类型转为 string，支持 nil 检查 转换失败返回错误
// 参数 v: 需要转换的值
// 返回 string 和 error
//
// 使用示例:
//
//	s, err := ToStrErr(123)       // "123", nil
//	s, err := ToStrErr(nil)       // "", error("值为 nil")
//	s, err := ToStrErr(true)      // "true", nil
//	s, err := ToStrErr(3.14)      // "3.140000", nil
//	s, err := ToStrErr(json.Number("99")) // "99", nil
func ToStrErr(v interface{}) (string, error) {
	if v == nil {
		return "", fmt.Errorf("值为 nil")
	}

	switch val := v.(type) {
	case string:
		return val, nil
	case *string:
		if val != nil {
			return *val, nil
		}
		return "", fmt.Errorf("string 指针为 nil")

	case int:
		return strconv.Itoa(val), nil
	case *int:
		if val != nil {
			return strconv.Itoa(*val), nil
		}
		return "", fmt.Errorf("int 指针为 nil")

	case int64:
		return strconv.FormatInt(val, 10), nil
	case *int64:
		if val != nil {
			return strconv.FormatInt(*val, 10), nil
		}
		return "", fmt.Errorf("int64 指针为 nil")

	case int32:
		return strconv.Itoa(int(val)), nil
	case *int32:
		if val != nil {
			return strconv.Itoa(int(*val)), nil
		}
		return "", fmt.Errorf("int32 指针为 nil")

	case float64:
		return fmt.Sprintf("%f", val), nil
	case *float64:
		if val != nil {
			return fmt.Sprintf("%f", *val), nil
		}
		return "", fmt.Errorf("float64 指针为 nil")

	case float32:
		return fmt.Sprintf("%f", val), nil
	case *float32:
		if val != nil {
			return fmt.Sprintf("%f", *val), nil
		}
		return "", fmt.Errorf("float32 指针为 nil")

	case bool:
		return strconv.FormatBool(val), nil
	case *bool:
		if val != nil {
			return strconv.FormatBool(*val), nil
		}
		return "", fmt.Errorf("bool 指针为 nil")

	case []byte:
		return string(val), nil
	case *[]byte:
		if val != nil {
			return string(*val), nil
		}
		return "", fmt.Errorf("[]byte 指针为 nil")

	case json.Number:
		return val.String(), nil

	default:
		return "", fmt.Errorf("未知类型: %T", v)
	}
}

// ToBytes 将任意类型转换为 []byte，无法转换时返回空切片
func ToBytes(v interface{}) []byte {
	var str string

	switch val := v.(type) {
	case string:
		str = val
	case *string:
		if val != nil {
			str = *val
		}
	case int:
		str = strconv.Itoa(val)
	case *int:
		if val != nil {
			str = strconv.Itoa(*val)
		}
	case int64:
		str = strconv.FormatInt(val, 10)
	case *int64:
		if val != nil {
			str = strconv.FormatInt(*val, 10)
		}
	case int32:
		str = strconv.Itoa(int(val))
	case *int32:
		if val != nil {
			str = strconv.Itoa(int(*val))
		}
	case float64:
		str = fmt.Sprintf("%f", val)
	case *float64:
		if val != nil {
			str = fmt.Sprintf("%f", *val)
		}
	case float32:
		str = fmt.Sprintf("%f", val)
	case *float32:
		if val != nil {
			str = fmt.Sprintf("%f", *val)
		}
	case bool:
		str = strconv.FormatBool(val)
	case *bool:
		if val != nil {
			str = strconv.FormatBool(*val)
		}
	case json.Number:
		str = val.String()
	default:
		if v == nil {
			return []byte{}
		}
		fmt.Printf("⚡ ToBytes 遇到未知类型：%T -> %+v\n", v, v)
		str = fmt.Sprintf("%v", v)
	}

	return []byte(str)
}

// ToBytesErr 将任意类型转换为 []byte，转换失败返回错误
func ToBytesErr(v interface{}) ([]byte, error) {
	if v == nil {
		return nil, fmt.Errorf("值为 nil")
	}

	var str string

	switch val := v.(type) {
	case string:
		str = val
	case *string:
		if val != nil {
			str = *val
		} else {
			return nil, fmt.Errorf("*string 为 nil")
		}
	case int:
		str = strconv.Itoa(val)
	case *int:
		if val != nil {
			str = strconv.Itoa(*val)
		} else {
			return nil, fmt.Errorf("*int 为 nil")
		}
	case int64:
		str = strconv.FormatInt(val, 10)
	case *int64:
		if val != nil {
			str = strconv.FormatInt(*val, 10)
		} else {
			return nil, fmt.Errorf("*int64 为 nil")
		}
	case int32:
		str = strconv.Itoa(int(val))
	case *int32:
		if val != nil {
			str = strconv.Itoa(int(*val))
		} else {
			return nil, fmt.Errorf("*int32 为 nil")
		}
	case float64:
		str = fmt.Sprintf("%f", val)
	case *float64:
		if val != nil {
			str = fmt.Sprintf("%f", *val)
		} else {
			return nil, fmt.Errorf("*float64 为 nil")
		}
	case float32:
		str = fmt.Sprintf("%f", val)
	case *float32:
		if val != nil {
			str = fmt.Sprintf("%f", *val)
		} else {
			return nil, fmt.Errorf("*float32 为 nil")
		}
	case bool:
		str = strconv.FormatBool(val)
	case *bool:
		if val != nil {
			str = strconv.FormatBool(*val)
		} else {
			return nil, fmt.Errorf("*bool 为 nil")
		}
	case json.Number:
		str = val.String()
	default:
		return nil, fmt.Errorf("未知类型: %T", v)
	}

	return []byte(str), nil
}

// ToInt64 将任意类型转换为 int64，无法转换时或超出范围时返回 0
func ToInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int:
		return int64(val)
	case int8:
		return int64(val)
	case int16:
		return int64(val)
	case int32:
		return int64(val)
	case int64:
		return val
	case uint:
		return int64(val)
	case uint8:
		return int64(val)
	case uint16:
		return int64(val)
	case uint32:
		return int64(val)
	case uint64:
		if val <= 1<<63-1 {
			return int64(val)
		}
		return 0
	case float32:
		return int64(val)
	case float64:
		return int64(val)
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0
		}
		return int64(f)
	case []byte:
		valData, _ := strconv.ParseInt(string(val), 10, 64)
		return valData
	default:
		return 0
	}
}

// ToInt64WithErr 将任意类型转换为 int64，无法转换或超出范围时返回错误
func ToInt64Err(v interface{}) (int64, error) {
	if v == nil {
		return 0, fmt.Errorf("值为 nil")
	}

	switch val := v.(type) {
	case int:
		return int64(val), nil
	case int8:
		return int64(val), nil
	case int16:
		return int64(val), nil
	case int32:
		return int64(val), nil
	case int64:
		return val, nil
	case uint:
		if val > math.MaxInt64 {
			return 0, fmt.Errorf("uint 值超出 int64 范围: %d", val)
		}
		return int64(val), nil
	case uint8:
		return int64(val), nil
	case uint16:
		return int64(val), nil
	case uint32:
		return int64(val), nil
	case uint64:
		if val > math.MaxInt64 {
			return 0, fmt.Errorf("uint64 值超出 int64 范围: %d", val)
		}
		return int64(val), nil
	case float32:
		if val > float32(math.MaxInt64) || val < float32(math.MinInt64) {
			return 0, fmt.Errorf("float32 值超出 int64 范围: %f", val)
		}
		return int64(val), nil
	case float64:
		if val > float64(math.MaxInt64) || val < float64(math.MinInt64) {
			return 0, fmt.Errorf("float64 值超出 int64 范围: %f", val)
		}
		return int64(val), nil
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, fmt.Errorf("字符串无法转换为 int64: %v", err)
		}
		if f > float64(math.MaxInt64) || f < float64(math.MinInt64) {
			return 0, fmt.Errorf("字符串数值超出 int64 范围: %f", f)
		}
		return int64(f), nil
	case []byte:
		s := string(val)
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("[]byte 无法转换为 int64: %v", err)
		}
		return i, nil
	default:
		return 0, fmt.Errorf("未知类型: %T", v)
	}
}

// ToInt 将任意类型转换为 int，无法转换时返回 0
func ToInt(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case int8:
		return int(val)
	case int16:
		return int(val)
	case int32:
		return int(val)
	case int64:
		return int(val) // 注意溢出风险
	case uint:
		return int(val)
	case uint8:
		return int(val)
	case uint16:
		return int(val)
	case uint32:
		return int(val)
	case uint64:
		if val <= uint64(^uint(0)) { // 判断不溢出
			return int(val)
		}
		return 0
	case float32:
		return int(val)
	case float64:
		if val > float64(^uint(0)) || val < float64(^uint(0))*-1 {
			return 0 // 溢出返回0
		}
		return int(val)
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0
		}
		return int(f)
	case []byte:
		f, err := strconv.ParseFloat(string(val), 64)
		if err != nil {
			return 0
		}
		return int(f)
	default:
		return 0
	}
}

// ToIntErr 将任意类型转换为 int，无法转换或溢出时返回错误
func ToIntErr(v interface{}) (int, error) {
	if v == nil {
		return 0, fmt.Errorf("值为 nil")
	}

	maxInt := int(^uint(0) >> 1)
	minInt := -maxInt - 1

	switch val := v.(type) {
	case int:
		return val, nil
	case int8, int16, int32, int64:
		i64 := int64(val.(int64)) // 先转换为 int64 统一处理
		if i64 > int64(maxInt) || i64 < int64(minInt) {
			return 0, fmt.Errorf("值超出 int 范围: %d", i64)
		}
		return int(i64), nil
	case uint, uint8, uint16, uint32:
		u64 := uint64(val.(uint64))
		if u64 > uint64(maxInt) {
			return 0, fmt.Errorf("值超出 int 范围: %d", u64)
		}
		return int(u64), nil
	case uint64:
		if val > uint64(maxInt) {
			return 0, fmt.Errorf("值超出 int 范围: %d", val)
		}
		return int(val), nil
	case float32:
		if val > float32(maxInt) || val < float32(minInt) {
			return 0, fmt.Errorf("float32 值超出 int 范围: %f", val)
		}
		return int(val), nil
	case float64:
		if val > float64(maxInt) || val < float64(minInt) {
			return 0, fmt.Errorf("float64 值超出 int 范围: %f", val)
		}
		return int(val), nil
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, fmt.Errorf("字符串无法转换为 int: %v", err)
		}
		if f > float64(maxInt) || f < float64(minInt) {
			return 0, fmt.Errorf("字符串数值超出 int 范围: %f", f)
		}
		return int(f), nil
	case []byte:
		s := string(val)
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("[]byte 无法转换为 int: %v", err)
		}
		if f > float64(maxInt) || f < float64(minInt) {
			return 0, fmt.Errorf("[]byte 数值超出 int 范围: %f", f)
		}
		return int(f), nil
	default:
		return 0, fmt.Errorf("未知类型: %T", v)
	}
}

// ToUint32 将任意类型转换为 uint32，无法转换或溢出时返回 0
func ToUint32(v interface{}) uint32 {
	switch val := v.(type) {
	case int:
		if val < 0 {
			return 0
		}
		return uint32(val)
	case int8:
		if val < 0 {
			return 0
		}
		return uint32(val)
	case int16:
		if val < 0 {
			return 0
		}
		return uint32(val)
	case int32:
		if val < 0 {
			return 0
		}
		return uint32(val)
	case int64:
		if val < 0 || val > int64(^uint32(0)) {
			return 0
		}
		return uint32(val)
	case uint:
		if val > uint(^uint32(0)) {
			return 0
		}
		return uint32(val)
	case uint8, uint16, uint32:
		return uint32(val.(uint64)) // 类型断言兼容
	case uint64:
		if val > uint64(^uint32(0)) {
			return 0
		}
		return uint32(val)
	case float32:
		if val < 0 || val > float32(^uint32(0)) {
			return 0
		}
		return uint32(val)
	case float64:
		if val < 0 || val > float64(^uint32(0)) {
			return 0
		}
		return uint32(val)
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil || f < 0 || f > float64(^uint32(0)) {
			return 0
		}
		return uint32(f)
	case []byte:
		f, err := strconv.ParseFloat(string(val), 64)
		if err != nil || f < 0 || f > float64(^uint32(0)) {
			return 0
		}
		return uint32(f)
	default:
		return 0
	}
}

// ToUint32Err 将任意类型转换为 uint32，无法转换或溢出时返回错误
func ToUint32Err(v interface{}) (uint32, error) {
	if v == nil {
		return 0, fmt.Errorf("值为 nil")
	}

	maxUint32 := ^uint32(0)

	switch val := v.(type) {
	case int:
		if val < 0 {
			return 0, fmt.Errorf("int 值为负数: %d", val)
		}
		if val > int(maxUint32) {
			return 0, fmt.Errorf("int 值超出 uint32 范围: %d", val)
		}
		return uint32(val), nil
	case int8, int16, int32, int64:
		i64 := int64(val.(int64))
		if i64 < 0 || i64 > int64(maxUint32) {
			return 0, fmt.Errorf("值超出 uint32 范围: %d", i64)
		}
		return uint32(i64), nil
	case uint, uint8, uint16, uint32:
		u64 := uint64(val.(uint64))
		if u64 > uint64(maxUint32) {
			return 0, fmt.Errorf("值超出 uint32 范围: %d", u64)
		}
		return uint32(u64), nil
	case uint64:
		if val > uint64(maxUint32) {
			return 0, fmt.Errorf("uint64 值超出 uint32 范围: %d", val)
		}
		return uint32(val), nil
	case float32:
		if val < 0 || val > float32(maxUint32) {
			return 0, fmt.Errorf("float32 值超出 uint32 范围: %f", val)
		}
		return uint32(val), nil
	case float64:
		if val < 0 || val > float64(maxUint32) {
			return 0, fmt.Errorf("float64 值超出 uint32 范围: %f", val)
		}
		return uint32(val), nil
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, fmt.Errorf("字符串无法转换为 uint32: %v", err)
		}
		if f < 0 || f > float64(maxUint32) {
			return 0, fmt.Errorf("字符串数值超出 uint32 范围: %f", f)
		}
		return uint32(f), nil
	case []byte:
		s := string(val)
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("[]byte 无法转换为 uint32: %v", err)
		}
		if f < 0 || f > float64(maxUint32) {
			return 0, fmt.Errorf("[]byte 数值超出 uint32 范围: %f", f)
		}
		return uint32(f), nil
	default:
		return 0, fmt.Errorf("未知类型: %T", v)
	}
}

// ToFloat64 将任意类型转换为 float64，无法转换时返回 0
func ToFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case int:
		return float64(val)
	case int8:
		return float64(val)
	case int16:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case uint:
		return float64(val)
	case uint8:
		return float64(val)
	case uint16:
		return float64(val)
	case uint32:
		return float64(val)
	case uint64:
		return float64(val)
	case float32:
		return float64(val)
	case float64:
		return val
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0
		}
		return f
	case []byte:
		f, err := strconv.ParseFloat(string(val), 64)
		if err != nil {
			return 0
		}
		return f
	default:
		return 0
	}
}

// ToFloat64Err 将任意类型转换为 float64，无法转换时返回错误
func ToFloat64Err(v interface{}) (float64, error) {
	if v == nil {
		return 0, fmt.Errorf("值为 nil")
	}

	switch val := v.(type) {
	case int:
		return float64(val), nil
	case int8:
		return float64(val), nil
	case int16:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case uint:
		return float64(val), nil
	case uint8:
		return float64(val), nil
	case uint16:
		return float64(val), nil
	case uint32:
		return float64(val), nil
	case uint64:
		return float64(val), nil
	case float32:
		return float64(val), nil
	case float64:
		return val, nil
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, fmt.Errorf("字符串无法转换为 float64: %v", err)
		}
		return f, nil
	case []byte:
		f, err := strconv.ParseFloat(string(val), 64)
		if err != nil {
			return 0, fmt.Errorf("[]byte 无法转换为 float64: %v", err)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("未知类型: %T", v)
	}
}

// ToFloat 将任意类型转换为 float32，无法转换时返回 0
func ToFloat(v interface{}) float32 {
	switch val := v.(type) {
	case int:
		return float32(val)
	case int8:
		return float32(val)
	case int16:
		return float32(val)
	case int32:
		return float32(val)
	case int64:
		return float32(val)
	case uint:
		return float32(val)
	case uint8:
		return float32(val)
	case uint16:
		return float32(val)
	case uint32:
		return float32(val)
	case uint64:
		return float32(val)
	case float32:
		return val
	case float64:
		return float32(val)
	case string:
		f, err := strconv.ParseFloat(val, 32)
		if err != nil {
			return 0
		}
		return float32(f)
	case []byte:
		f, err := strconv.ParseFloat(string(val), 32)
		if err != nil {
			return 0
		}
		return float32(f)
	default:
		return 0
	}
}

// ToFloatErr 将任意类型转换为 float32，无法转换时返回错误
func ToFloatErr(v interface{}) (float32, error) {
	if v == nil {
		return 0, fmt.Errorf("值为 nil")
	}

	switch val := v.(type) {
	case int:
		return float32(val), nil
	case int8:
		return float32(val), nil
	case int16:
		return float32(val), nil
	case int32:
		return float32(val), nil
	case int64:
		return float32(val), nil
	case uint:
		return float32(val), nil
	case uint8:
		return float32(val), nil
	case uint16:
		return float32(val), nil
	case uint32:
		return float32(val), nil
	case uint64:
		return float32(val), nil
	case float32:
		return val, nil
	case float64:
		return float32(val), nil
	case string:
		f, err := strconv.ParseFloat(val, 32)
		if err != nil {
			return 0, fmt.Errorf("字符串无法转换为 float32: %v", err)
		}
		return float32(f), nil
	case []byte:
		f, err := strconv.ParseFloat(string(val), 32)
		if err != nil {
			return 0, fmt.Errorf("[]byte 无法转换为 float32: %v", err)
		}
		return float32(f), nil
	default:
		return 0, fmt.Errorf("未知类型: %T", v)
	}
}

// ToUint64 将任意类型转换为 uint64，无法转换或负数时返回 0
func ToUint64(v interface{}) uint64 {
	switch val := v.(type) {
	case int:
		if val < 0 {
			return 0
		}
		return uint64(val)
	case int8:
		if val < 0 {
			return 0
		}
		return uint64(val)
	case int16:
		if val < 0 {
			return 0
		}
		return uint64(val)
	case int32:
		if val < 0 {
			return 0
		}
		return uint64(val)
	case int64:
		if val < 0 {
			return 0
		}
		return uint64(val)
	case uint:
		return uint64(val)
	case uint8:
		return uint64(val)
	case uint16:
		return uint64(val)
	case uint32:
		return uint64(val)
	case uint64:
		return val
	case float32:
		if val < 0 {
			return 0
		}
		return uint64(val)
	case float64:
		if val < 0 {
			return 0
		}
		return uint64(val)
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil || f < 0 {
			return 0
		}
		return uint64(f)
	case []byte:
		f, err := strconv.ParseFloat(string(val), 64)
		if err != nil || f < 0 {
			return 0
		}
		return uint64(f)
	default:
		return 0
	}
}

// ToUint64Err 将任意类型转换为 uint64，无法转换或为负数时返回错误
func ToUint64Err(v interface{}) (uint64, error) {
	if v == nil {
		return 0, fmt.Errorf("值为 nil")
	}

	switch val := v.(type) {
	case int:
		if val < 0 {
			return 0, fmt.Errorf("int 值为负数: %d", val)
		}
		return uint64(val), nil
	case int8:
		if val < 0 {
			return 0, fmt.Errorf("int8 值为负数: %d", val)
		}
		return uint64(val), nil
	case int16:
		if val < 0 {
			return 0, fmt.Errorf("int16 值为负数: %d", val)
		}
		return uint64(val), nil
	case int32:
		if val < 0 {
			return 0, fmt.Errorf("int32 值为负数: %d", val)
		}
		return uint64(val), nil
	case int64:
		if val < 0 {
			return 0, fmt.Errorf("int64 值为负数: %d", val)
		}
		return uint64(val), nil
	case uint, uint8, uint16, uint32, uint64:
		return uint64(val.(uint64)), nil
	case float32:
		if val < 0 {
			return 0, fmt.Errorf("float32 值为负数: %f", val)
		}
		return uint64(val), nil
	case float64:
		if val < 0 {
			return 0, fmt.Errorf("float64 值为负数: %f", val)
		}
		return uint64(val), nil
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, fmt.Errorf("字符串无法转换为 uint64: %v", err)
		}
		if f < 0 {
			return 0, fmt.Errorf("字符串数值为负数: %f", f)
		}
		return uint64(f), nil
	case []byte:
		f, err := strconv.ParseFloat(string(val), 64)
		if err != nil {
			return 0, fmt.Errorf("[]byte 无法转换为 uint64: %v", err)
		}
		if f < 0 {
			return 0, fmt.Errorf("[]byte 数值为负数: %f", f)
		}
		return uint64(f), nil
	default:
		return 0, fmt.Errorf("未知类型: %T", v)
	}
}

// HexStringToBytes 将十六进制字符串转换为 []byte
//
// 支持的格式:
//
//	0xA0D1, A0 D1, \A0 \D1, a0d1 等
func HexStringToBytes(s string) ([]byte, error) {
	// 去掉0x或0X前缀
	s = strings.TrimPrefix(strings.ToLower(s), "0x")

	// 去掉反斜杠
	s = strings.ReplaceAll(s, "\\", "")

	// 去掉所有空格
	s = strings.ReplaceAll(s, " ", "")

	// 如果长度为奇数，补0在前面
	if len(s)%2 != 0 {
		s = "0" + s
	}

	// 使用 encoding/hex 解码
	data, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("hex decode failed: %w", err)
	}

	return data, nil
}

// BytesToHexString 将 []byte 数据转换为十六进制字符串
//
// 参数：
//   - data: 待转换的字节数组
//   - withSpace: 是否在每个字节之间添加空格，true 表示加空格，false 表示不加
//
// 返回值：
//   - string: 转换后的十六进制字符串，字母默认大写
//
// 功能说明：
//  1. 每个字节会转换为两位十六进制字符。
//  2. 如果 withSpace 为 true，则每个字节之间用空格分隔。
//  3. 转换后的字符串方便在日志、调试、网络协议打印等场景使用。
//
// 示例：
//
//	data := []byte{0x12, 0xAB, 0x34, 0xCD}
//	// 不加空格
//	hexStr := BytesToHexString(data, false)
//	fmt.Println(hexStr) // 输出: "12AB34CD"
//
//	// 加空格
//	hexStrWithSpace := BytesToHexString(data, true)
//	fmt.Println(hexStrWithSpace) // 输出: "12 AB 34 CD"
func BytesToHexString(data []byte, withSpace bool) string {
	// 将字节数组编码为十六进制字符串（小写）
	s := hex.EncodeToString(data)

	if withSpace {
		var parts []string
		// 每两个字符为一组，对应原始字节
		for i := 0; i < len(s); i += 2 {
			// 转为大写并加入 parts 切片
			parts = append(parts, strings.ToUpper(s[i:i+2]))
		}
		// 用空格连接每个字节的十六进制表示
		return strings.Join(parts, " ")
	}

	// 不加空格，直接返回全部大写的十六进制字符串
	return strings.ToUpper(s)
}

// Base64Convert 通用 Base64 编码/解码函数
//
// 参数:
//   - data: 输入数据，可以是 string 或 []byte
//   - encode: true = 编码，false = 解码
//
// 返回值:
//   - []byte: 转换后的数据
//   - error: 转换错误信息
func Base64Convert(data interface{}, encode bool) ([]byte, error) {
	var input []byte

	// 统一转换输入为 []byte
	switch v := data.(type) {
	case string:
		input = []byte(v)
	case []byte:
		input = v
	default:
		return nil, fmt.Errorf("unsupported input type: %T", data)
	}

	if encode {
		// Base64 编码
		encoded := base64.StdEncoding.EncodeToString(input)
		return []byte(encoded), nil
	} else {
		// Base64 解码
		decoded, err := base64.StdEncoding.DecodeString(string(input))
		if err != nil {
			return nil, fmt.Errorf("base64 decode failed: %w", err)
		}
		return decoded, nil
	}
}

// GetFormatTimeByParam 根据参数截取时间
//
// now: 时间文本，可为空，如果为空则取当前时间
// param: 截取级别，可空，默认0
//
//	0=秒, 1=年, 2=月, 3=日, 4=小时, 5=分钟
func GetFormatTimeByParam(now string, param int) string {
	var t time.Time
	var err error

	// 如果输入为空，取当前时间
	if now == "" {
		t = time.Now()
	} else {
		// 尝试解析输入时间
		t, err = time.Parse("2006-01-02 15:04:05", now)
		if err != nil {
			// 解析失败用当前时间
			t = time.Now()
		}
	}

	// 根据 param 返回对应格式
	switch param {
	case 1:
		return t.Format("2006")
	case 2:
		return t.Format("2006-01")
	case 3:
		return t.Format("2006-01-02")
	case 4:
		return t.Format("2006-01-02 15")
	case 5:
		return t.Format("2006-01-02 15:04")
	default:
		return t.Format("2006-01-02 15:04:05")
	}
}

// TimeStampToStr 将时间戳转换为字符串格式
//
// ts: 时间戳
// isMilli: true 表示 ts 为13位毫秒时间戳，false 表示10位秒时间戳
func TimeStampToStr(ts int64, isMilli bool) string {
	if ts <= 0 {
		return ""
	}

	var t time.Time
	if isMilli {
		// 毫秒时间戳
		t = time.Unix(0, ts*int64(time.Millisecond))
	} else {
		// 秒时间戳
		t = time.Unix(ts, 0)
	}

	return t.Format("2006-01-02 15:04:05")
}

// TimeToTimeStamp 将时间转换为10位或13位时间戳
//
// t: 要转换的时间，可以是 time.Time、字符串或 nil/其他类型
//
//	字符串格式应为 "2025-01-02 15:04:05"
//
// isMilli: true 表示返回13位毫秒时间戳，false 表示10位秒时间戳
func TimeToTimeStamp(t interface{}, isMilli bool) int64 {
	var tt time.Time

	switch v := t.(type) {
	case time.Time:
		tt = v
	case string:
		parsed, err := time.Parse("2006-01-02 15:04:05", v)
		if err != nil {
			// 解析失败用当前时间
			tt = time.Now()
		} else {
			tt = parsed
		}
	default:
		tt = time.Now()
	}

	if isMilli {
		return tt.UnixNano() / int64(time.Millisecond)
	}
	return tt.Unix()
}

// GetRunPath 获取当前cmd终端目录或当前程序运行目录
//
// 如果参数1 填写 true 则获取cmd终端目录，填写false 则返回程序所在目录
func GetRunPath(cmd bool) string {
	wd, err := os.Getwd() // 获取当前工作目录
	if cmd == true {
		if err == nil {
			return wd
		}

	}

	// 获取可执行文件的绝对路径
	execPath, err := os.Executable()
	if err != nil {
		fmt.Println("获取可执行文件路径失败:", err)
		return wd
	}

	// 取出目录部分
	execDir := filepath.Dir(execPath)

	return execDir
}

// FileExists 判断文件是否存在 返回true表示存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	// 其他错误视为“未知状态”，此处也返回 false 更安全
	return false
}

// DeleteFileOrDir 删除文件或整个目录（包含内容）
func DeleteFileOrDir(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		return fmt.Errorf("删除失败: %w", err)
	}
	return nil
}

// GetLeftOfUnderscore 取文本左边
// 参数:
//   - s: 参数1 原始文本
//   - keyWord: 参数2 如原来的文本是 abc-bbb 参数2提供"-"则返回abc，如果没有 "-", 则返回原始字符串
func GetLeftOfUnderscore(s string, keyWord string) string {
	idx := strings.Index(s, keyWord)
	if idx == -1 {
		return s
	}
	return s[:idx]
}

// GetRightOfSeparator 取文本右边
//
// 参数:
//   - s: 原始文本
//   - keyWord: 分隔符，如原文本是 "abc-bbb"，提供 "-" 则返回 "bbb"
//     如果找不到 keyWord，则返回原始字符串
func GetRightOfSeparator(s string, keyWord string) string {
	idx := strings.Index(s, keyWord)
	if idx == -1 {
		return s
	}
	return s[idx+len(keyWord):]
}

// GetMiddleOfSeparator 取文本中间
//
// 参数:
//   - s: 原始文本
//   - left: 左分隔符
//   - right: 右分隔符
//
// 示例:
//
//	s = "abc-123-xyz", left = "-", right = "-" -> 返回 "123"
func GetMiddleOfSeparator(s, left, right string) string {
	startIdx := strings.Index(s, left)
	if startIdx == -1 {
		return ""
	}
	startIdx += len(left)

	endIdx := strings.Index(s[startIdx:], right)
	if endIdx == -1 {
		return ""
	}

	return s[startIdx : startIdx+endIdx]
}

// CopyDirOrFile 拷贝文件或目录
//
// 参数:
//   - src: 源路径，可以是文件或目录
//   - dst: 目标路径
func CopyDirOrFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("获取源信息失败: %w", err)
	}

	if info.IsDir() {
		// 如果是目录，递归拷贝
		return copyDir(src, dst)
	} else {
		// 如果是文件，直接拷贝文件
		return copyFile(src, dst)
	}
}

// copyDir 递归拷贝目录
func copyDir(srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("读取目录失败: %w", err)
	}

	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile 拷贝单个文件
func copyFile(srcFile, dstFile string) error {
	src, err := os.Open(srcFile)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer src.Close()

	// 创建目标文件
	dst, err := os.Create(dstFile)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		return fmt.Errorf("拷贝文件失败: %w", err)
	}

	// 保持原文件权限
	info, err := os.Stat(srcFile)
	if err == nil {
		os.Chmod(dstFile, info.Mode())
	}

	return nil
}

// ReadFile 读取指定路径的文件内容，返回字节切片和错误信息
func ReadFile(filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// WriteBytesToFile 将 byte 数据写入到指定文件
func WriteBytesToFile(filePath string, data []byte) error {
	// 使用 os.WriteFile 写入文件，权限设置为 0644（可读写）
	err := os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}
	return nil
}

// CSVStringsToLine 将字符串切片转换为一行 CSV 格式的 []byte
//
// 参数：
//   - fields: 要转换的字符串切片
//   - lineBreak: 是否在末尾添加换行符（\r\n）
//
// 处理规则：
//  1. 自动用逗号连接
//  2. 移除英文逗号（避免干扰分隔）
//  3. 移除双引号（避免错位）
//  4. 移除换行符（\r \n）
func CSVStringsToLine(fields []string, lineBreak bool) []byte {
	for i, str := range fields {
		// 不允许英文逗号
		str = strings.ReplaceAll(str, ",", "")
		fields[i] = str
	}

	// 拼接成一行
	combined := strings.Join(fields, ",")

	// 清理不允许的字符
	combined = strings.ReplaceAll(combined, `"`, "")
	combined = strings.ReplaceAll(combined, "\r", "")
	combined = strings.ReplaceAll(combined, "\n", "")

	// 转成字节，并根据需要添加换行
	if lineBreak {
		return []byte(combined + "\r\n")
	}
	return []byte(combined)
}

// ListAllFilesByModTime 遍历目录下的所有文件，并按修改时间升序排序，支持通配符/后缀名过滤
//
// 参数1: root 要遍历的目录
// 参数2: patterns 过滤规则列表，可以是后缀(.csv)，也可以是通配符(*.csv, aaa*.*)
func ListAllFilesByModTime(root string, patterns []string) ([]string, error) {
	var files []FileInfoExts

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			log.Printf("Warning: failed to access %q: %v\n", path, walkErr)
			return nil // 忽略错误，继续遍历
		}

		if d.IsDir() {
			return nil
		}

		// 如果设置了过滤条件
		if len(patterns) > 0 {
			match := false
			filename := filepath.Base(path)

			for _, p := range patterns {
				// 如果是单纯的扩展名（如 ".csv"）
				if strings.HasPrefix(p, ".") {
					if strings.EqualFold(filepath.Ext(filename), p) {
						match = true
						break
					}
					continue
				}

				// 通配符匹配 (*.csv, aaa*.* 等)
				ok, _ := filepath.Match(p, filename)
				if ok {
					match = true
					break
				}
			}

			if !match {
				return nil
			}
		}

		info, err := d.Info()
		if err != nil {
			log.Printf("Warning: unable to stat file %q: %v\n", path, err)
			return nil
		}

		files = append(files, FileInfoExts{
			Path:    path,
			ModTime: info.ModTime(),
		})

		// 打点日志（每扫描5万文件提示一次）
		if len(files)%50000 == 0 {
			log.Printf("Scanned %d files...\n", len(files))
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 按修改时间升序排序
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime.Before(files[j].ModTime)
	})

	// 提取路径列表
	sortedPaths := make([]string, len(files))
	for i, f := range files {
		sortedPaths[i] = f.Path
	}

	return sortedPaths, nil
}

// FilterFileName 根据关键字过滤文件名
// 如果 keywords 为空不需要过滤返回 true,返回true 表示包含全部关键字,返回false 表示不存在特定关键字
func FilterFileName(fullPath string, keywords []string) bool {
	filename := filepath.Base(fullPath)
	ok := true
	if len(keywords) == 0 {

		return ok
	}

	for _, kw := range keywords {
		if strings.Contains(filename, kw) {
			if ok != false {
				ok = true
			}
		} else {
			ok = false
		}
	}

	return ok // 不包含关键字
}

// GB2312ToUtf8 检测文件编码并将 GB2312/GBK 文件转换为 UTF-8
//
// 一般用于判断CSV文件编码是否为UTF-8，如果是GB2312则转换为UTF-8
func GB2312ToUtf8(filePath string) error {
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	// 读取前4KB用于编码检测
	buf := make([]byte, 4096)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return fmt.Errorf("读取文件失败: %v", err)
	}
	sample := buf[:n]

	// 使用 chardet 检测编码
	detector := chardet.NewTextDetector()
	result, err := detector.DetectBest(sample)
	if err != nil {
		return fmt.Errorf("编码检测失败: %v", err)
	}

	//fmt.Println("检测到编码:", result.Charset)

	// 如果是 UTF-8，就不转换
	if strings.EqualFold(result.Charset, "UTF-8") {
		//fmt.Println("文件是 UTF-8 编码, 无需转换")
		return nil
	}

	// 目前只处理 GBK/GB2312 → UTF-8，其他编码类型暂不支持
	if !strings.Contains(result.Charset, "GB") {
		return fmt.Errorf("不支持的编码类型: %s", result.Charset)
	}

	fmt.Println("文件是 GBK/GB2312 编码，开始转换为 UTF-8...")

	// 重新读取整个文件（从头开始）
	file.Seek(0, io.SeekStart)
	reader := transform.NewReader(file, simplifiedchinese.GBK.NewDecoder())

	// 读取转换后的 UTF-8 内容
	converted, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("GB2312 转换失败: %v", err)
	}

	// 覆盖写入 UTF-8 编码文件
	err = os.WriteFile(filePath, converted, 0644)
	if err != nil {
		return fmt.Errorf("写入 UTF-8 文件失败: %v", err)
	}

	fmt.Println("转换完成，文件已保存为 UTF-8 编码")
	return nil
}

/*
FileToUTF8 将文件转为 UTF-8 编码，支持多种常见编码

简体中文：GB2312 / GBK（simplifiedchinese.GBK）

繁体中文：Big5（traditionalchinese.Big5）

日文：Shift-JIS（japanese.ShiftJIS）

韩文：EUC-KR（korean.EUCKR）

西欧：ISO-8859-1（Latin1，charmap.ISO8859_1）

Windows 系列：Windows-1252（charmap.Windows1252）
*/
func FileToUTF8(filePath string) error {
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	// 读取前4KB用于编码检测
	buf := make([]byte, 4096)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return fmt.Errorf("读取文件失败: %v", err)
	}
	sample := buf[:n]

	// 检测文件编码
	detector := chardet.NewTextDetector()
	result, err := detector.DetectBest(sample)
	if err != nil {
		return fmt.Errorf("编码检测失败: %v", err)
	}
	fmt.Println("检测到编码:", result.Charset)

	// 如果是 UTF-8 就直接返回
	if strings.EqualFold(result.Charset, "UTF-8") {
		fmt.Println("文件已是 UTF-8，无需转换")
		return nil
	}

	// 找到对应的编码
	enc := getEncoding(result.Charset)
	if enc == nil {
		return fmt.Errorf("暂不支持的编码类型: %s", result.Charset)
	}

	fmt.Printf("文件是 %s 编码，开始转换为 UTF-8...\n", result.Charset)

	// 从头读取
	file.Seek(0, io.SeekStart)
	reader := transform.NewReader(file, enc.NewDecoder())

	// 转换为 UTF-8
	converted, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("%s 转换失败: %v", result.Charset, err)
	}

	// 覆盖写回 UTF-8
	err = os.WriteFile(filePath, converted, 0644)
	if err != nil {
		return fmt.Errorf("写入 UTF-8 文件失败: %v", err)
	}

	fmt.Println("转换完成，文件已保存为 UTF-8 编码")
	return nil
}

// getEncoding 根据检测结果返回对应编码
func getEncoding(charset string) encoding.Encoding {
	cs := strings.ToUpper(charset)
	switch {
	case strings.Contains(cs, "GB"):
		return simplifiedchinese.GBK
	case strings.Contains(cs, "BIG5"):
		return traditionalchinese.Big5
	case strings.Contains(cs, "SHIFT_JIS"), strings.Contains(cs, "SJIS"):
		return japanese.ShiftJIS
	case strings.Contains(cs, "EUC-KR"):
		return korean.EUCKR
	case strings.Contains(cs, "ISO-8859-1"):
		return charmap.ISO8859_1
	case strings.Contains(cs, "WINDOWS-1252"):
		return charmap.Windows1252
	default:
		return nil
	}
}

// EncodeConvert 通用编码转换函数
//
// 功能说明：
//
//	将输入的数据（字符串或字节切片）从源编码转换为 UTF-8 编码。
//	支持自动检测源编码，也可手动指定源编码。
//	常见支持的编码包括：UTF-8、UTF-16LE/BE、GBK/GB2312、BIG5、Shift-JIS、EUC-JP、EUC-KR、ISO-8859-1~16、Windows-125x 等。
//
// 参数说明：
//   - data: 待转换的数据，类型可以是 string 或 []byte。
//   - targetEncoding: 手动指定源编码名称（如 "GBK", "ISO-8859-1"），仅在 autoDetect=false 时生效。
//   - autoDetect: 是否自动检测源编码，true 表示自动检测，false 表示使用 targetEncoding 指定的编码。
//
// 返回值：
//   - []byte: 转换后的 UTF-8 编码字节切片。
//   - error: 转换失败或输入类型不支持时返回错误。
//
// 使用示例：
//
//	gbkData := []byte{0xC4, 0xE3, 0xBA, 0xC3} // "你好" GBK
//	result, err := EncodeConvert(gbkData, "GBK", true)
//	if err != nil {
//	    fmt.Println("Error:", err)
//	} else {
//	    fmt.Println("Result:", string(result))
//	}
func EncodeConvert(data interface{}, targetEncoding string, autoDetect bool) ([]byte, error) {
	var raw []byte
	switch v := data.(type) {
	case string:
		raw = []byte(v)
	case []byte:
		raw = v
	default:
		return nil, fmt.Errorf("unsupported input type: %T", data)
	}

	var srcEnc encoding.Encoding
	if autoDetect {
		detector := chardet.NewTextDetector()
		result, err := detector.DetectBest(raw)
		if err != nil {
			return nil, fmt.Errorf("encoding detection failed: %w", err)
		}
		srcEnc = detectEncodingByName(result.Charset)
	} else {
		srcEnc = detectEncodingByName(targetEncoding)
	}

	if srcEnc == nil {
		return nil, fmt.Errorf("unsupported encoding: %s", targetEncoding)
	}

	reader := transform.NewReader(strings.NewReader(string(raw)), srcEnc.NewDecoder())
	converted, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("encoding conversion failed: %w", err)
	}

	return converted, nil
}

// 支持更多常见编码
func detectEncodingByName(name string) encoding.Encoding {
	switch strings.ToUpper(name) {
	case "UTF-8":
		return encoding.Nop
	case "UTF-16LE":
		return unicode.UTF16(unicode.LittleEndian, unicode.ExpectBOM)
	case "UTF-16BE":
		return unicode.UTF16(unicode.BigEndian, unicode.ExpectBOM)
	case "GBK", "GB2312":
		return simplifiedchinese.GBK
	case "BIG5":
		return traditionalchinese.Big5
	case "SHIFT_JIS", "SJIS":
		return japanese.ShiftJIS
	case "EUC-JP":
		return japanese.EUCJP
	case "EUC-KR":
		return korean.EUCKR
	case "ISO-8859-1":
		return charmap.ISO8859_1
	case "ISO-8859-2":
		return charmap.ISO8859_2
	case "ISO-8859-3":
		return charmap.ISO8859_3
	case "ISO-8859-4":
		return charmap.ISO8859_4
	case "ISO-8859-5":
		return charmap.ISO8859_5
	case "ISO-8859-6":
		return charmap.ISO8859_6
	case "ISO-8859-7":
		return charmap.ISO8859_7
	case "ISO-8859-8":
		return charmap.ISO8859_8
	case "ISO-8859-9":
		return charmap.ISO8859_9
	case "WINDOWS-1250":
		return charmap.Windows1250
	case "WINDOWS-1251":
		return charmap.Windows1251
	case "WINDOWS-1252":
		return charmap.Windows1252
	case "WINDOWS-1253":
		return charmap.Windows1253
	case "WINDOWS-1254":
		return charmap.Windows1254
	case "WINDOWS-1255":
		return charmap.Windows1255
	case "WINDOWS-1256":
		return charmap.Windows1256
	case "WINDOWS-1257":
		return charmap.Windows1257
	case "WINDOWS-1258":
		return charmap.Windows1258
	default:
		return nil
	}
}

// HTMLConvert 通用 HTML 编码/解码函数
//
// 功能：
//   - 当 encode=true：对文本进行 HTML 实体编码（如 `<` -> `&lt;`, `&` -> `&amp;` 等）
//   - 当 encode=false：对 HTML 实体进行还原（如 `&lt;` -> `<`, `&#34;` -> `"` 等）
//
// 参数：
//   - data: 输入数据（string 或 []byte）
//   - encode: true=编码，false=解码
//
// 返回：
//   - []byte: 转换后的结果（UTF-8）
//   - error : 输入类型不支持等错误
//
// 说明：
//   - 使用标准库 html 包的 EscapeString / UnescapeString，支持常见命名实体和数字实体
//   - 适用于一般文本内容的 HTML 安全编码/解码（非针对 URL、JS、CSS 的上下文）
//
// 示例：
//
//	enc, _ := HTMLConvert("<div title=\"A&B\">", true)   // => "&lt;div title=&quot;A&amp;B&quot;&gt;"
//	dec, _ := HTMLConvert(enc, false)                     // => "<div title="A&B">"
func HTMLConvert(data interface{}, encode bool) ([]byte, error) {
	var in string
	switch v := data.(type) {
	case string:
		in = v
	case []byte:
		in = string(v)
	default:
		return nil, fmt.Errorf("unsupported input type: %T (only string or []byte)", data)
	}

	if encode {
		return []byte(html.EscapeString(in)), nil
	}
	return []byte(html.UnescapeString(in)), nil
}

// CSVFieldMapper 自动识别和映射 CSV 表头及行数据，提供CSV字段名称直接去出字段对应的值,如果CSV 文件中每个字段是 "" 引号引起来的则需要 使用CleanStrings() 先去除首位的引号
//
// 参数:
//   - fields: []string 传入 CSV 的表头或行数据（已按逗号分割好）
//   - mapping: *map[string]int 传址 map
//   - 当 isHeader = true 时：更新表头字段和位置的映射关系
//   - 当 isHeader = false 时：通过字段名获取对应的行值
//   - isHeader: bool
//   - true  → 表示传入的是表头，自动计算字段在 CSV 中的位置并存入 mapping
//   - false → 表示传入的是行数据，根据 mapping 提取对应字段内容
//   - fieldName: string
//   - 当 isHeader = false 时，用于指定要取的字段名
//
// 返回:
//   - string: 如果是表头模式，返回 ""；如果是行数据模式，返回对应字段名的实际值
//   - error : 如果字段不存在或索引超界，则返回错误
//
// 示例:
//
//	mapping := make(map[string]int)
//	// 传入表头
//	CSVFieldMapper([]string{"id", "name", "age"}, &mapping, true, "")
//	// mapping 结果: map["id"]=0, map["name"]=1, map["age"]=2
//
//	// 传入行数据并取值
//	row := []string{"1", "Tom", "18"}
//	val, _ := CSVFieldMapper(row, &mapping, false, "name")
//	fmt.Println(val) // 输出: Tom
func CSVFieldMapper(fields []string, mapping *map[string]int, isHeader bool, fieldName string) (string, error) {
	if mapping == nil {
		return "", errors.New("mapping cannot be nil")
	}

	if isHeader {
		// 构建表头索引映射
		for i, field := range fields {
			(*mapping)[field] = i
		}
		return "", nil
	}

	// 提取csv字段所在表中的位置
	idx, ok := (*mapping)[fieldName]
	if !ok {
		return "", fmt.Errorf("field %s not found in mapping", fieldName)
	}
	if idx < 0 || idx >= len(fields) {
		return "", fmt.Errorf("field %s index out of range", fieldName)
	}

	return fields[idx], nil
}

// URLEncodeDecode 进行 URL 编码或解码
//
// 参数1：原始字符串
//
// 参数2：操作类型，true 表示编码encode，false 表示解码Decode
//
// 返回：处理后的字符串和错误
//
// 示例：
//
//	encoded, _ := URLEncodeDecode("a b&c", true)
//	fmt.Println(encoded) // 输出: a+b%26c
//
//	decoded, _ := URLEncodeDecode("a+b%26c", false)
//	fmt.Println(decoded) // 输出: a b&c
func URLEncodeDecode(input string, encode bool) (string, error) {
	if encode {
		return url.QueryEscape(input), nil
	}
	return url.QueryUnescape(input)
}

// WriteToStingFile 将 []string数组写到文件，一般用于写出CSV文件比较方便,CSV文件每行组装后直接放到[]string 数组里面,最后统一写出
//
// 参数1 文件路径
//
// 参数2 写出的[]string类型数组数据
//
// 参数3 是否添加换行符,添加写 true
func WriteToStingFile(fileName string, data []string, AddNewline bool) {
	file, err := os.Create(fileName)

	if err != nil {
		fmt.Println("写到文件错误:"+fileName, err)
	}
	defer file.Close()

	for _, temp := range data {
		if AddNewline == true {
			temp = temp + "\r\n"
		}

		_, err = file.Write([]byte(temp))
		if err != nil {
			fmt.Println("写出文件数据错误:", err)
		}

	}

}

// WriteToByteFile 将字节切片写入到指定文件
//
// 参数：
//   - fileName: 输出文件路径
//   - data: 要写入的字节数据
//
// 功能：
//  1. 如果文件不存在则创建；存在则覆盖
//  2. 将 []byte 一次性写入文件
//  3. 写入失败时打印错误信息
func WriteToByteFile(fileName string, data []byte) {
	// 创建/覆盖文件
	file, err := os.Create(fileName)
	if err != nil {
		fmt.Println("创建文件失败:", fileName, err)
		return
	}
	defer file.Close()

	// 写入数据
	_, err = file.Write(data)
	if err != nil {
		fmt.Println("写入文件数据失败:", err)
	}
}

// CleanStrings 处理字符串切片：去除首尾双引号和空格 一般用于处理csv 行分割后的数据
func CleanStrings(input []string) []string {
	result := make([]string, 0, len(input))
	for _, s := range input {
		// 去除首尾双引号
		s = strings.Trim(s, `"`)
		// 去除首尾空格
		s = strings.TrimSpace(s)
		s = strings.ReplaceAll(s, "\t", "")
		result = append(result, s)
	}
	return result
}

// GetTextTwoMiddle 取两段文本中间
// 参数：
//   - sourceText       参数1 原始文本
//   - startText        参数2 开始文本,如果为空 则取右边的文本
//   - endText          参数3 结束文本,如果为空 则取左边的文本
//   - startPosition    参数4 始位置,如果从0开始则填写0或-1,如果此参数大于0 则先跳到某个特定位置后再取参数1 和参数2中间的文本，注意如果该值大于0 且参数5不等于空则先调到开始位置后 然后再查找参数5所在的位置 然后再取参数1 和参数2中间的文本
//   - offset           参数5 从某文本后开始查找,就是先找到这个文本 然后再取 参数1 和参数3 中间的文本
//   - fallbackToSource 参数6 如果未找到是否返回原始文本 如果填写 true 未找到则返回原始文本
//
// 返回：
//   - int 		返回值1 如果找到返回 结束文本 最后一个字符所在的位置,可以用于下一轮循环查找,注意即使参数6 填写真未找到的情况该值还是会返回 -1
//   - string 	返回值2: 返回找到的实际文本,如果参数6 传入 true 未找到的情况会返回原始文本
//
// 示例1：
//
//	a := "你好aa你好测试你好aa你好测试你好MM，你好HelloWord你好对"
//	num, c := tools.GetTextTwoMiddle(a, "你好", "你好", 0, "MM", false)
//	fmt.Println(num, c)//输出 72 HelloWord
//
// //原因是它先找到MM 然后再取MM 后面的 你好 和你好中间的文本
//
// 示例2：
//
//	a := "你好aa你好测试你好aa你好测试你好MM，你好HelloWord你好对"
//	num, c := tools.GetTextTwoMiddle(a, "你好", "", 0, "MM", false)
//	fmt.Println(num, c)//输出57 HelloWord你好对
func GetTextTwoMiddle(sourceText string, startText string, endText string, startPosition int, offset string, fallbackToSource bool) (int, string) {
	sourceTextTemp := sourceText
	backEndPos := 0
	if startPosition == -1 {
		startPosition = 0
	}

	if startPosition > 0 { //开始位置大于0 先截取文本右边
		if len(sourceText) >= startPosition {
			backEndPos = backEndPos + startPosition //取右边的时候需要开始计算偏移位置
			sourceText = sourceText[startPosition:] //取出开始位置的文本右边
		} else {
			if fallbackToSource == true { //填写true 未找到返回原始文本
				return -1, sourceTextTemp
			}
			return -1, ""
		}

	}

	if offset != "" { //计算从某文本后开始查找
		pos := strings.Index(sourceText, offset) //查找是否存在 参数5 从某文本后开始查找
		if pos != -1 {
			if len(sourceText) >= pos+len(offset) {
				backEndPos = backEndPos + pos + len(offset) //取右边的时候需要开始计算偏移位置
				sourceText = sourceText[pos+len(offset):]   //取出找到某文本后 右边的文本,但此处要加上 从某文本后开始查找 长度
			} else {
				if fallbackToSource == true { //填写true 未找到返回原始文本
					return -1, sourceTextTemp
				}
				return -1, ""
			}

		} else {
			if fallbackToSource == true { //填写true 未找到返回原始文本
				return -1, sourceTextTemp
			}
			return -1, ""
		}
	}

	if startText != "" {
		startPos := strings.Index(sourceText, startText) //查找是否存在 参数1 开始文本
		if startPos != -1 {
			if len(sourceText) >= startPos+len(startText) {
				backEndPos = backEndPos + startPos + len(startText) //取右边的时候需要开始计算偏移位置
				sourceText = sourceText[startPos+len(startText):]   //取出找到 开始文本 右边的文本,但此处要加上 开始文本的 长度
			} else {
				if fallbackToSource == true { //填写true 未找到返回原始文本
					return -1, sourceTextTemp
				}
				return -1, ""
			}

		} else { //根本就不存在开始文本
			if fallbackToSource == true { //填写true 未找到返回原始文本
				return -1, sourceTextTemp
			}
			return -1, ""
		}

	}

	if endText != "" {
		endPos := strings.Index(sourceText, endText) //查找是否存在 参数2 结束文本
		if endPos != -1 {
			if endPos >= 0 {
				sourceText = sourceText[:endPos]
				backEndPos = backEndPos + len(sourceText) + len(endText) //取右边的时候需要开始计算偏移位置

			} else {
				if fallbackToSource == true { //填写true 未找到返回原始文本
					return -1, sourceTextTemp
				}
				return -1, ""
			}
		} else {
			if fallbackToSource == true { //填写true 未找到返回原始文本
				return -1, sourceTextTemp
			}
			return -1, ""
		}

	}

	return backEndPos, sourceText
}

// GetTextTwoMiddleBytes 取两段文本中间 (byte 版本)
//
// 参数：
//   - sourceText       参数1 原始文本
//   - startText        参数2 开始文本,如果为空 则取右边的文本
//   - endText          参数3 结束文本,如果为空 则取左边的文本
//   - startPosition    参数4 开始位置,如果从0开始则填写0或-1,如果此参数大于0 则先跳到某个特定位置后再取参数1 和参数2中间的文本，注意如果该值大于0 且参数5不等于空则先调到开始位置后 然后再查找参数5所在的位置 然后再取参数1 和参数2中间的文本
//   - offset           参数5 从某文本后开始查找,就是先找到这个文本 然后再取 参数1 和参数3 中间的文本
//   - fallbackToSource 参数6 如果未找到是否返回原始文本 如果填写 true 未找到则返回原始文本
//
// 返回：
//   - int    返回值1 如果找到返回 结束文本最后一个字符所在的位置,可以用于下一轮循环查找,注意即使参数6 填写真未找到的情况该值还是会返回 -1
//   - []byte 返回值2 返回找到的实际文本,如果参数6 传入 true 未找到的情况会返回原始文本
//
// 示例1：
//
//	a := "你好aa你好测试你好aa你好测试你好MM，你好HelloWord你好对"
//	numByte, cByte := tools.GetTextTwoMiddleBytes([]byte(a), []byte("你好"), []byte("你好"), 0, []byte("a"), false)
//	fmt.Println(numByte, cByte)//输出 [230 181 139 232 175 149]  ="测试"
//	fmt.Println(numByte, tools.ToStr(cByte))//输出 "测试"
func GetTextTwoMiddleBytes(sourceText, startText, endText []byte, startPosition int, offset []byte, fallbackToSource bool) (int, []byte) {
	sourceTextTemp := sourceText
	backEndPos := 0
	if startPosition == -1 {
		startPosition = 0
	}

	if startPosition > 0 { //开始位置大于0 先截取文本右边
		if len(sourceText) >= startPosition {
			backEndPos = backEndPos + startPosition
			sourceText = sourceText[startPosition:]
		} else {
			if fallbackToSource {
				return -1, sourceTextTemp
			}
			return -1, nil
		}
	}

	if len(offset) > 0 { //计算从某文本后开始查找
		pos := bytes.Index(sourceText, offset)
		if pos != -1 {
			if len(sourceText) >= pos+len(offset) {
				backEndPos = backEndPos + pos + len(offset)
				sourceText = sourceText[pos+len(offset):]
			} else {
				if fallbackToSource {
					return -1, sourceTextTemp
				}
				return -1, nil
			}
		} else {
			if fallbackToSource {
				return -1, sourceTextTemp
			}
			return -1, nil
		}
	}

	if len(startText) > 0 {
		startPos := bytes.Index(sourceText, startText)
		if startPos != -1 {
			if len(sourceText) >= startPos+len(startText) {
				backEndPos = backEndPos + startPos + len(startText)
				sourceText = sourceText[startPos+len(startText):]
			} else {
				if fallbackToSource {
					return -1, sourceTextTemp
				}
				return -1, nil
			}
		} else {
			if fallbackToSource {
				return -1, sourceTextTemp
			}
			return -1, nil
		}
	}

	if len(endText) > 0 {
		endPos := bytes.Index(sourceText, endText)
		if endPos != -1 {
			if endPos >= 0 {
				sourceText = sourceText[:endPos]
				backEndPos = backEndPos + len(sourceText) + len(endText)
			} else {
				if fallbackToSource {
					return -1, sourceTextTemp
				}
				return -1, nil
			}
		} else {
			if fallbackToSource {
				return -1, sourceTextTemp
			}
			return -1, nil
		}
	}

	return backEndPos, sourceText
}
