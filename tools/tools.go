package tools

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
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
	"hash"
	"hash/crc32"
	"hash/crc64"
	"html"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"math"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RandStr 生成随机字符串
// mode 采用按位组合：
//
//	1 -> 小写字母
//	2 -> 数字
//	4 -> 大写字母
//	8 -> 特殊字符
//
// 可组合，例如：
//
//		3 = 1 + 2 → 小写 + 数字
//		7 = 1 + 2 + 4 → 小写 + 数字 + 大写
//		15 = 1 + 2 + 4 + 8 → 所有字符
//	 textStr := RandStr(7,32)
//	 textStr := RandStr(2,32)
func RandStr(mode int, length int) string {
	lower := "abcdefghijklmnopqrstuvwxyz"
	digits := "0123456789"
	upper := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	special := "!@#$%^&*()-_=+[]{}<>?/|"

	charset := ""

	// mode & 1 → 小写
	if mode&1 > 0 {
		charset += lower
	}

	// mode & 2 → 数字
	if mode&2 > 0 {
		charset += digits
	}

	// mode & 4 → 大写
	if mode&4 > 0 {
		charset += upper
	}

	// mode & 8 → 特殊字符
	if mode&8 > 0 {
		charset += special
	}

	// 如果模式非法（用户传入0等），则默认用数字
	if charset == "" {
		charset = digits
	}

	result := make([]byte, length)

	for i := 0; i < length; i++ {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[num.Int64()]
	}

	return string(result)
}

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
// ToStr 将任意类型转换为 string，无法转换时返回 ""
// 不适用于高安全/高精度场景
func ToStr(v interface{}) string {
	switch val := v.(type) {

	// =========================
	// string
	// =========================
	case string:
		return val

	case *string:
		if val != nil {
			return *val
		}

	case []byte:
		return string(val)

	case *[]byte:
		if val != nil {
			return string(*val)
		}

	case json.RawMessage:
		return string(val)

	case *json.RawMessage:
		if val != nil {
			return string(*val)
		}

	case []rune:
		return string(val)

	// =========================
	// int
	// =========================
	case int:
		return strconv.Itoa(val)

	case *int:
		if val != nil {
			return strconv.Itoa(*val)
		}

	case int8:
		return strconv.FormatInt(int64(val), 10)

	case *int8:
		if val != nil {
			return strconv.FormatInt(int64(*val), 10)
		}

	case int16:
		return strconv.FormatInt(int64(val), 10)

	case *int16:
		if val != nil {
			return strconv.FormatInt(int64(*val), 10)
		}

	case int32:
		return strconv.FormatInt(int64(val), 10)

	case *int32:
		if val != nil {
			return strconv.FormatInt(int64(*val), 10)
		}

	case int64:
		return strconv.FormatInt(val, 10)

	case *int64:
		if val != nil {
			return strconv.FormatInt(*val, 10)
		}

	// =========================
	// uint
	// =========================
	case uint:
		return strconv.FormatUint(uint64(val), 10)

	case *uint:
		if val != nil {
			return strconv.FormatUint(uint64(*val), 10)
		}

	case uint8:
		return strconv.FormatUint(uint64(val), 10)

	case *uint8:
		if val != nil {
			return strconv.FormatUint(uint64(*val), 10)
		}

	case uint16:
		return strconv.FormatUint(uint64(val), 10)

	case *uint16:
		if val != nil {
			return strconv.FormatUint(uint64(*val), 10)
		}

	case uint32:
		return strconv.FormatUint(uint64(val), 10)

	case *uint32:
		if val != nil {
			return strconv.FormatUint(uint64(*val), 10)
		}

	case uint64:
		return strconv.FormatUint(val, 10)

	case *uint64:
		if val != nil {
			return strconv.FormatUint(*val, 10)
		}

	case uintptr:
		return strconv.FormatUint(uint64(val), 10)

	case *uintptr:
		if val != nil {
			return strconv.FormatUint(uint64(*val), 10)
		}

	// =========================
	// float
	// =========================
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 32)

	case *float32:
		if val != nil {
			return strconv.FormatFloat(float64(*val), 'f', -1, 32)
		}

	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)

	case *float64:
		if val != nil {
			return strconv.FormatFloat(*val, 'f', -1, 64)
		}

	// =========================
	// bool
	// =========================
	case bool:
		return strconv.FormatBool(val)

	case *bool:
		if val != nil {
			return strconv.FormatBool(*val)
		}

	// =========================
	// json
	// =========================
	case json.Number:
		return val.String()

	case *json.Number:
		if val != nil {
			return val.String()
		}

	// =========================
	// time
	// =========================
	case time.Time:
		return val.Format(time.RFC3339Nano)

	case *time.Time:
		if val != nil {
			return val.Format(time.RFC3339Nano)
		}

	// =========================
	// error
	// =========================
	case error:
		return val.Error()

	// =========================
	// fmt.Stringer
	// =========================
	case fmt.Stringer:
		return val.String()

	// =========================
	// string slice
	// =========================
	case []string:
		return strings.Join(val, ",")

	case *[]string:
		if val != nil {
			return strings.Join(*val, ",")
		}
	}

	// =========================
	// fallback
	// =========================
	if v == nil {
		return ""
	}

	fmt.Printf("⚡ ToStr遇到未知类型：%T -> %+v\n", v, v)

	return fmt.Sprintf("%v", v)
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
		return "", fmt.Errorf("nil value")
	}

	switch val := v.(type) {

	case string:
		return val, nil
	case *string:
		if val != nil {
			return *val, nil
		}
		return "", fmt.Errorf("nil *string")

	case []byte:
		return string(val), nil
	case *[]byte:
		if val != nil {
			return string(*val), nil
		}
		return "", fmt.Errorf("nil *[]byte")

	case json.RawMessage:
		return string(val), nil
	case *json.RawMessage:
		if val != nil {
			return string(*val), nil
		}
		return "", fmt.Errorf("nil *json.RawMessage")

	case []rune:
		return string(val), nil

	case int:
		return strconv.Itoa(val), nil
	case *int:
		if val != nil {
			return strconv.Itoa(*val), nil
		}
		return "", fmt.Errorf("nil *int")

	case int64:
		return strconv.FormatInt(val, 10), nil
	case *int64:
		if val != nil {
			return strconv.FormatInt(*val, 10), nil
		}
		return "", fmt.Errorf("nil *int64")

	case int32:
		return strconv.Itoa(int(val)), nil
	case *int32:
		if val != nil {
			return strconv.Itoa(int(*val)), nil
		}
		return "", fmt.Errorf("nil *int32")

	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64), nil
	case *float64:
		if val != nil {
			return strconv.FormatFloat(*val, 'f', -1, 64), nil
		}
		return "", fmt.Errorf("nil *float64")

	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 32), nil
	case *float32:
		if val != nil {
			return strconv.FormatFloat(float64(*val), 'f', -1, 32), nil
		}
		return "", fmt.Errorf("nil *float32")

	case bool:
		return strconv.FormatBool(val), nil
	case *bool:
		if val != nil {
			return strconv.FormatBool(*val), nil
		}
		return "", fmt.Errorf("nil *bool")

	case json.Number:
		return val.String(), nil
	case *json.Number:
		if val != nil {
			return val.String(), nil
		}
		return "", fmt.Errorf("nil *json.Number")

	case time.Time:
		return val.Format(time.RFC3339Nano), nil
	case *time.Time:
		if val != nil {
			return val.Format(time.RFC3339Nano), nil
		}
		return "", fmt.Errorf("nil *time.Time")

	case error:
		return val.Error(), nil
	case *error:
		if val != nil && *val != nil {
			return (*val).Error(), nil
		}
		return "", fmt.Errorf("nil *error")

	case fmt.Stringer:
		return val.String(), nil
	case *fmt.Stringer:
		if val != nil {
			return (*val).String(), nil
		}
		return "", fmt.Errorf("nil *fmt.Stringer")

	case []string:
		return strings.Join(val, ","), nil
	case *[]string:
		if val != nil {
			return strings.Join(*val, ","), nil
		}
		return "", fmt.Errorf("nil *[]string")

	case []any:
		b, err := json.Marshal(val)
		if err != nil {
			return "", err
		}
		return string(b), nil
		
	case map[string]interface{}:
		b, err := json.Marshal(val)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	return "", fmt.Errorf("unsupported type: %T", v)
}

// ToBytes 将任意类型转换为 []byte，无法转换时返回空切片
func ToBytes(v interface{}) []byte {
	if v == nil {
		return []byte{}
	}

	switch val := v.(type) {

	// ===== 二进制优先 =====
	case []byte:
		return val

	case *[]byte:
		if val != nil {
			return *val
		}
		return []byte{}

	case json.RawMessage:
		return val

	// ===== 字符串 =====
	case string:
		return []byte(val)

	case *string:
		if val != nil {
			return []byte(*val)
		}
		return []byte{}

	case []rune:
		return []byte(string(val))

	// ===== 标准接口 =====
	case error:
		return []byte(val.Error())

	case fmt.Stringer:
		return []byte(val.String())

	// ===== 时间 =====
	case time.Time:
		return []byte(val.Format("2006-01-02 15:04:05"))

	case *time.Time:
		if val != nil {
			return []byte(val.Format("2006-01-02 15:04:05"))
		}
		return []byte{}

	// ===== 整数 =====
	case int:
		return []byte(strconv.Itoa(val))

	case *int:
		if val != nil {
			return []byte(strconv.Itoa(*val))
		}
		return []byte{}

	case int64:
		return []byte(strconv.FormatInt(val, 10))

	case *int64:
		if val != nil {
			return []byte(strconv.FormatInt(*val, 10))
		}
		return []byte{}

	case int32:
		return []byte(strconv.Itoa(int(val)))

	case *int32:
		if val != nil {
			return []byte(strconv.Itoa(int(*val)))
		}
		return []byte{}

	// ===== 浮点（修复精度问题）=====
	case float64:
		return []byte(strconv.FormatFloat(val, 'f', -1, 64))

	case *float64:
		if val != nil {
			return []byte(strconv.FormatFloat(*val, 'f', -1, 64))
		}
		return []byte{}

	case float32:
		return []byte(strconv.FormatFloat(float64(val), 'f', -1, 32))

	case *float32:
		if val != nil {
			return []byte(strconv.FormatFloat(float64(*val), 'f', -1, 32))
		}
		return []byte{}

	// ===== bool =====
	case bool:
		return []byte(strconv.FormatBool(val))

	case *bool:
		if val != nil {
			return []byte(strconv.FormatBool(*val))
		}
		return []byte{}

	case json.Number:
		return []byte(val.String())
	}

	// ===== 反射兜底 =====
	rv := reflect.ValueOf(v)

	switch rv.Kind() {

	case reflect.Map, reflect.Struct:
		b, err := json.Marshal(v)
		if err == nil {
			return b
		}

	case reflect.Slice, reflect.Array:
		//排除 []byte
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			if b, ok := v.([]byte); ok {
				return b
			}
		}

		b, err := json.Marshal(v)
		if err == nil {
			return b
		}
	}

	// ===== 打印日志=====
	fmt.Printf("⚡ ToBytes fallback: %T -> %+v\n", v, v)

	return []byte(fmt.Sprintf("%v", v))
}

// ToBytesErr 将任意类型转换为 []byte，转换失败返回错误
func ToBytesErr(v interface{}) ([]byte, error) {
	if v == nil {
		return nil, fmt.Errorf("val is nil")
	}

	switch val := v.(type) {

	case []byte:
		return val, nil

	case *[]byte:
		if val != nil {
			return *val, nil
		}
		return nil, fmt.Errorf("*[]byte is nil")

	case string:
		return []byte(val), nil

	case *string:
		if val != nil {
			return []byte(*val), nil
		}
		return nil, fmt.Errorf("*string is nil")

	case json.RawMessage:
		return val, nil

	case error:
		return []byte(val.Error()), nil

	case fmt.Stringer:
		return []byte(val.String()), nil

	case time.Time:
		return []byte(val.Format("2006-01-02 15:04:05")), nil

	case *time.Time:
		if val != nil {
			return []byte(val.Format("2006-01-02 15:04:05")), nil
		}
		return nil, fmt.Errorf("*time.Time is nil")

	case []rune:
		return []byte(string(val)), nil

	case int:
		return []byte(strconv.Itoa(val)), nil

	case int64:
		return []byte(strconv.FormatInt(val, 10)), nil

	case int32:
		return []byte(strconv.Itoa(int(val))), nil

	case float64:
		return []byte(strconv.FormatFloat(val, 'f', -1, 64)), nil

	case float32:
		return []byte(strconv.FormatFloat(float64(val), 'f', -1, 32)), nil

	case bool:
		return []byte(strconv.FormatBool(val)), nil

	case json.Number:
		return []byte(val.String()), nil
	}

	// ===== fallback =====
	rv := reflect.ValueOf(v)

	switch rv.Kind() {

	case reflect.Map, reflect.Struct:
		return json.Marshal(v)

	case reflect.Slice, reflect.Array:
		// 排除 []byte
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			return v.([]byte), nil
		}
		return json.Marshal(v)
	default:
		return nil, fmt.Errorf("unsupported type: %T", v)
	}

}

// ToInt64 将任意类型转换为 int64，无法转换时或超出范围时返回 0
func ToInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}

	switch val := v.(type) {

	// ===== 整数 =====
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

	// ===== 无符号 =====
	case uint:
		return int64(val)
	case uint8:
		return int64(val)
	case uint16:
		return int64(val)
	case uint32:
		return int64(val)
	case uint64:
		if val <= math.MaxInt64 {
			return int64(val)
		}
		return 0

	// ===== 浮点 =====
	case float32:
		return int64(val)
	case float64:
		return int64(val)

	// ===== bool =====
	case bool:
		if val {
			return 1
		}
		return 0

	// ===== string =====
	case string:
		s := strings.TrimSpace(val)

		// 优先 int
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i
		}

		// 再 float
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return int64(f)
		}

		return 0

	// ===== []byte =====
	case []byte:
		return ToInt64(string(val))

	// ===== json.Number =====
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return i
		}
		if f, err := val.Float64(); err == nil {
			return int64(f)
		}
		return 0

	// ===== 接口 =====
	case fmt.Stringer:
		return ToInt64(val.String())

	case error:
		return 0

	case time.Time:
		return val.Unix()
	}

	// ===== 反射兜底 =====
	rv := reflect.ValueOf(v)

	switch rv.Kind() {
	case reflect.Ptr:
		if rv.IsNil() {
			return 0
		}
		return ToInt64(rv.Elem().Interface())
	}

	return 0
}

// ToInt64WithErr 将任意类型转换为 int64，无法转换或超出范围时返回错误
func ToInt64Err(v interface{}) (int64, error) {
	if v == nil {
		return 0, fmt.Errorf("val is nil")
	}

	switch val := v.(type) {

	// ===== 整数 =====
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

	// ===== 无符号 =====
	case uint:
		if uint64(val) > math.MaxInt64 {
			return 0, fmt.Errorf("uint overflow")
		}
		return int64(val), nil

	case uint64:
		if val > math.MaxInt64 {
			return 0, fmt.Errorf("uint64 overflow")
		}
		return int64(val), nil

	case uint8, uint16, uint32:
		return int64(reflect.ValueOf(val).Uint()), nil

	// ===== 浮点 =====
	case float32:
		f := float64(val)
		if f > math.MaxInt64 || f < math.MinInt64 {
			return 0, fmt.Errorf("float32 overflow")
		}
		return int64(f), nil

	case float64:
		if val > math.MaxInt64 || val < math.MinInt64 {
			return 0, fmt.Errorf("float64 overflow")
		}
		return int64(val), nil

	// ===== bool =====
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil

	// ===== string =====
	case string:
		s := strings.TrimSpace(val)

		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i, nil
		}

		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("string parse error: %v", err)
		}

		if f > math.MaxInt64 || f < math.MinInt64 {
			return 0, fmt.Errorf("float overflow")
		}

		return int64(f), nil

	// ===== []byte =====
	case []byte:
		return ToInt64Err(string(val))

	// ===== json.Number =====
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return i, nil
		}
		if f, err := val.Float64(); err == nil {
			return int64(f), nil
		}
		return 0, fmt.Errorf("json.Number parse error")

	// ===== 接口 =====
	case fmt.Stringer:
		return ToInt64Err(val.String())

	case time.Time:
		return val.Unix(), nil
	}

	// ===== 指针处理 =====
	rv := reflect.ValueOf(v)

	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return 0, fmt.Errorf("nil pointer")
		}
		return ToInt64Err(rv.Elem().Interface())
	}

	return 0, fmt.Errorf("unsupported type: %T", v)
}

// ToInt 将任意类型转换为 int，无法转换时返回 0
func ToInt(v interface{}) int {
	if v == nil {
		return 0
	}

	maxInt := int(^uint(0) >> 1)
	minInt := -maxInt - 1

	switch val := v.(type) {

	// ===== int 系列 =====
	case int:
		return val
	case int8:
		return int(val)
	case int16:
		return int(val)
	case int32:
		return int(val)
	case int64:
		if val > int64(maxInt) || val < int64(minInt) {
			return 0
		}
		return int(val)

	// ===== uint =====
	case uint:
		if val > uint(maxInt) {
			return 0
		}
		return int(val)

	case uint8, uint16, uint32:
		return int(reflect.ValueOf(val).Uint())

	case uint64:
		if val > uint64(maxInt) {
			return 0
		}
		return int(val)

	// ===== float =====
	case float32:
		f := float64(val)
		if f > float64(maxInt) || f < float64(minInt) {
			return 0
		}
		return int(f)

	case float64:
		if val > float64(maxInt) || val < float64(minInt) {
			return 0
		}
		return int(val)

	// ===== bool =====
	case bool:
		if val {
			return 1
		}
		return 0

	// ===== string =====
	case string:
		s := strings.TrimSpace(val)

		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return ToInt(i)
		}

		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return ToInt(f)
		}

		return 0

	// ===== []byte =====
	case []byte:
		return ToInt(string(val))

	// ===== json.Number =====
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return ToInt(i)
		}
		if f, err := val.Float64(); err == nil {
			return ToInt(f)
		}
		return 0

	// ===== 接口 =====
	case fmt.Stringer:
		return ToInt(val.String())

	case error:
		return 0

	case time.Time:
		return ToInt(val.Unix())
	}

	// ===== 指针处理 =====
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return 0
		}
		return ToInt(rv.Elem().Interface())
	}

	return 0
}

// ToIntErr 将任意类型转换为 int，无法转换或溢出时返回错误
func ToIntErr(v interface{}) (int, error) {
	if v == nil {
		return 0, fmt.Errorf("val is nil")
	}

	maxInt := int(^uint(0) >> 1)
	minInt := -maxInt - 1

	switch val := v.(type) {

	// ===== int =====
	case int:
		return val, nil

	case int8:
		return int(val), nil
	case int16:
		return int(val), nil
	case int32:
		return int(val), nil

	case int64:
		if val > int64(maxInt) || val < int64(minInt) {
			return 0, fmt.Errorf("int64 overflow")
		}
		return int(val), nil

	// ===== uint =====
	case uint:
		if val > uint(maxInt) {
			return 0, fmt.Errorf("uint overflow")
		}
		return int(val), nil

	case uint8, uint16, uint32:
		return int(reflect.ValueOf(val).Uint()), nil

	case uint64:
		if val > uint64(maxInt) {
			return 0, fmt.Errorf("uint64 overflow")
		}
		return int(val), nil

	// ===== float =====
	case float32:
		f := float64(val)
		if f > float64(maxInt) || f < float64(minInt) {
			return 0, fmt.Errorf("float32 overflow")
		}
		return int(f), nil

	case float64:
		if val > float64(maxInt) || val < float64(minInt) {
			return 0, fmt.Errorf("float64 overflow")
		}
		return int(val), nil

	// ===== bool =====
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil

	// ===== string =====
	case string:
		s := strings.TrimSpace(val)

		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return ToIntErr(i)
		}

		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("string parse error: %v", err)
		}

		if f > float64(maxInt) || f < float64(minInt) {
			return 0, fmt.Errorf("float overflow")
		}

		return int(f), nil

	// ===== []byte =====
	case []byte:
		return ToIntErr(string(val))

	// ===== json.Number =====
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return ToIntErr(i)
		}
		if f, err := val.Float64(); err == nil {
			return ToIntErr(f)
		}
		return 0, fmt.Errorf("json.Number parse error")

	// ===== 接口 =====
	case fmt.Stringer:
		return ToIntErr(val.String())

	case time.Time:
		return ToIntErr(val.Unix())
	}

	// ===== 指针处理 =====
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return 0, fmt.Errorf("nil pointer")
		}
		return ToIntErr(rv.Elem().Interface())
	}

	return 0, fmt.Errorf("unsupported type: %T", v)
}

// ToUint32 将任意类型转换为 uint32，无法转换或溢出时返回 0
func ToUint32(v interface{}) uint32 {
	if v == nil {
		return 0
	}

	max := uint64(^uint32(0))

	switch val := v.(type) {

	// ===== int =====
	case int:
		if val < 0 {
			return 0
		}
		return ToUint32(int64(val))

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
		if val < 0 || uint64(val) > max {
			return 0
		}
		return uint32(val)

	// ===== uint =====
	case uint:
		if uint64(val) > max {
			return 0
		}
		return uint32(val)

	case uint8:
		return uint32(val)
	case uint16:
		return uint32(val)
	case uint32:
		return val

	case uint64:
		if val > max {
			return 0
		}
		return uint32(val)

	// ===== float =====
	case float32:
		f := float64(val)
		if f < 0 || f > float64(max) {
			return 0
		}
		return uint32(f)

	case float64:
		if val < 0 || val > float64(max) {
			return 0
		}
		return uint32(val)

	// ===== bool =====
	case bool:
		if val {
			return 1
		}
		return 0

	// ===== string =====
	case string:
		s := strings.TrimSpace(val)

		if u, err := strconv.ParseUint(s, 10, 32); err == nil {
			return uint32(u)
		}

		if f, err := strconv.ParseFloat(s, 64); err == nil && f >= 0 && f <= float64(max) {
			return uint32(f)
		}

		return 0

	// ===== []byte =====
	case []byte:
		return ToUint32(string(val))

	// ===== json.Number =====
	case json.Number:
		if i, err := val.Int64(); err == nil && i >= 0 {
			return ToUint32(i)
		}
		if f, err := val.Float64(); err == nil {
			return ToUint32(f)
		}
		return 0

	// ===== 接口 =====
	case fmt.Stringer:
		return ToUint32(val.String())

	case error:
		return 0

	case time.Time:
		return ToUint32(val.Unix())
	}

	// ===== 指针 =====
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return 0
		}
		return ToUint32(rv.Elem().Interface())
	}

	return 0
}

// ToUint32Err 将任意类型转换为 uint32，无法转换或溢出时返回错误
func ToUint32Err(v interface{}) (uint32, error) {
	if v == nil {
		return 0, fmt.Errorf("val is nil")
	}

	max := uint64(^uint32(0))

	switch val := v.(type) {

	// ===== int =====
	case int:
		if val < 0 || uint64(val) > max {
			return 0, fmt.Errorf("int overflow")
		}
		return uint32(val), nil

	case int8:
		if val < 0 {
			return 0, fmt.Errorf("negative int8")
		}
		return uint32(val), nil

	case int16:
		if val < 0 {
			return 0, fmt.Errorf("negative int16")
		}
		return uint32(val), nil

	case int32:
		if val < 0 {
			return 0, fmt.Errorf("negative int32")
		}
		return uint32(val), nil

	case int64:
		if val < 0 || uint64(val) > max {
			return 0, fmt.Errorf("int64 overflow")
		}
		return uint32(val), nil

	// ===== uint =====
	case uint:
		if uint64(val) > max {
			return 0, fmt.Errorf("uint overflow")
		}
		return uint32(val), nil

	case uint8:
		return uint32(val), nil
	case uint16:
		return uint32(val), nil
	case uint32:
		return val, nil

	case uint64:
		if val > max {
			return 0, fmt.Errorf("uint64 overflow")
		}
		return uint32(val), nil

	// ===== float =====
	case float32:
		f := float64(val)
		if f < 0 || f > float64(max) {
			return 0, fmt.Errorf("float32 overflow")
		}
		return uint32(f), nil

	case float64:
		if val < 0 || val > float64(max) {
			return 0, fmt.Errorf("float64 overflow")
		}
		return uint32(val), nil

	// ===== bool =====
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil

	// ===== string =====
	case string:
		s := strings.TrimSpace(val)

		if u, err := strconv.ParseUint(s, 10, 32); err == nil {
			return uint32(u), nil
		}

		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("string parse error: %v", err)
		}

		if f < 0 || f > float64(max) {
			return 0, fmt.Errorf("float overflow")
		}

		return uint32(f), nil

	// ===== []byte =====
	case []byte:
		return ToUint32Err(string(val))

	// ===== json.Number =====
	case json.Number:
		if i, err := val.Int64(); err == nil && i >= 0 {
			return ToUint32Err(i)
		}
		if f, err := val.Float64(); err == nil {
			return ToUint32Err(f)
		}
		return 0, fmt.Errorf("json.Number parse error")

	// ===== 接口 =====
	case fmt.Stringer:
		return ToUint32Err(val.String())

	case time.Time:
		return ToUint32Err(val.Unix())
	}

	// ===== 指针 =====
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return 0, fmt.Errorf("nil pointer")
		}
		return ToUint32Err(rv.Elem().Interface())
	}

	return 0, fmt.Errorf("unsupported type: %T", v)
}

// ToFloat64 将任意类型转换为 float64，无法转换时返回 0
func ToFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}

	switch val := v.(type) {

	// ===== int =====
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

	// ===== uint =====
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

	// ===== float =====
	case float32:
		return float64(val)
	case float64:
		return val

	// ===== bool =====
	case bool:
		if val {
			return 1
		}
		return 0

	// ===== string =====
	case string:
		s := strings.TrimSpace(val)
		if s == "" {
			return 0
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0
		}
		return f

	// ===== []byte =====
	case []byte:
		return ToFloat64(string(val))

	// ===== json.Number =====
	case json.Number:
		f, err := val.Float64()
		if err != nil {
			return 0
		}
		return f

	// ===== 接口 =====
	case fmt.Stringer:
		return ToFloat64(val.String())

	case error:
		return 0

	case time.Time:
		return float64(val.Unix())

	// ===== rune slice =====
	case []rune:
		return ToFloat64(string(val))
	}

	// ===== 指针统一处理 =====
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return 0
		}
		return ToFloat64(rv.Elem().Interface())
	}

	return 0
}

// ToFloat64Err 将任意类型转换为 float64，无法转换时返回错误
func ToFloat64Err(v interface{}) (float64, error) {
	if v == nil {
		return 0, fmt.Errorf("val is nil")
	}

	switch val := v.(type) {

	// ===== int =====
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

	// ===== uint =====
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

	// ===== float =====
	case float32:
		return float64(val), nil
	case float64:
		return val, nil

	// ===== bool =====
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil

	// ===== string =====
	case string:
		s := strings.TrimSpace(val)
		if s == "" {
			return 0, fmt.Errorf("empty string")
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("string parse error: %v", err)
		}
		return f, nil

	// ===== []byte =====
	case []byte:
		return ToFloat64Err(string(val))

	// ===== json.Number =====
	case json.Number:
		return val.Float64()

	// ===== 接口 =====
	case fmt.Stringer:
		return ToFloat64Err(val.String())

	case error:
		return 0, fmt.Errorf("cannot convert error to float64")

	case time.Time:
		return float64(val.Unix()), nil

	case []rune:
		return ToFloat64Err(string(val))
	}

	// ===== 指针 =====
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return 0, fmt.Errorf("nil pointer")
		}
		return ToFloat64Err(rv.Elem().Interface())
	}

	return 0, fmt.Errorf("unsupported type: %T", v)
}

// ToFloat 将任意类型转换为 float32，无法转换时返回 0
func ToFloat(v interface{}) float32 {
	if v == nil {
		return 0
	}

	switch val := v.(type) {

	// ===== int =====
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

	// ===== uint =====
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

	// ===== float =====
	case float32:
		return val
	case float64:
		return float32(val)

	// ===== bool =====
	case bool:
		if val {
			return 1
		}
		return 0

	// ===== string =====
	case string:
		s := strings.TrimSpace(val)
		if s == "" {
			return 0
		}
		f, err := strconv.ParseFloat(s, 32)
		if err != nil {
			return 0
		}
		return float32(f)

	// ===== []byte =====
	case []byte:
		return ToFloat(string(val))

	// ===== json.Number =====
	case json.Number:
		f, err := val.Float64()
		if err != nil {
			return 0
		}
		return float32(f)

	// ===== 接口 =====
	case fmt.Stringer:
		return ToFloat(val.String())

	case error:
		return 0

	case time.Time:
		return float32(val.Unix())

	// ===== rune =====
	case []rune:
		return ToFloat(string(val))
	}

	// ===== 指针统一处理 =====
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return 0
		}
		return ToFloat(rv.Elem().Interface())
	}

	return 0
}

// ToFloatErr 将任意类型转换为 float32，无法转换时返回错误
func ToFloatErr(v interface{}) (float32, error) {
	if v == nil {
		return 0, fmt.Errorf("val is nil")
	}

	switch val := v.(type) {

	// ===== int =====
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

	// ===== uint =====
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

	// ===== float =====
	case float32:
		return val, nil
	case float64:
		return float32(val), nil

	// ===== bool =====
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil

	// ===== string =====
	case string:
		s := strings.TrimSpace(val)
		if s == "" {
			return 0, fmt.Errorf("empty string")
		}
		f, err := strconv.ParseFloat(s, 32)
		if err != nil {
			return 0, fmt.Errorf("string parse error: %v", err)
		}
		return float32(f), nil

	// ===== []byte =====
	case []byte:
		return ToFloatErr(string(val))

	// ===== json.Number =====
	case json.Number:
		f, err := val.Float64()
		if err != nil {
			return 0, err
		}
		return float32(f), nil

	// ===== 接口 =====
	case fmt.Stringer:
		return ToFloatErr(val.String())

	case error:
		return 0, fmt.Errorf("cannot convert error to float32")

	case time.Time:
		return float32(val.Unix()), nil

	case []rune:
		return ToFloatErr(string(val))
	}

	// ===== 指针 =====
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return 0, fmt.Errorf("nil pointer")
		}
		return ToFloatErr(rv.Elem().Interface())
	}

	return 0, fmt.Errorf("unsupported type: %T", v)
}

// ToUint64 将任意类型转换为 uint64，无法转换或负数时返回 0
func ToUint64(v interface{}) uint64 {
	if v == nil {
		return 0
	}

	switch val := v.(type) {

	// ===== int =====
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

	// ===== uint =====
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

	// ===== float =====
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

	// ===== bool =====
	case bool:
		if val {
			return 1
		}
		return 0

	// ===== string =====
	case string:
		s := strings.TrimSpace(val)
		if s == "" {
			return 0
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil || f < 0 {
			return 0
		}
		return uint64(f)

	// ===== []byte =====
	case []byte:
		return ToUint64(string(val))

	// ===== json.Number =====
	case json.Number:
		i, err := val.Int64()
		if err != nil || i < 0 {
			return 0
		}
		return uint64(i)

	// ===== 接口 =====
	case fmt.Stringer:
		return ToUint64(val.String())

	case error:
		return 0

	case time.Time:
		return uint64(val.Unix())

	// ===== rune =====
	case []rune:
		return ToUint64(string(val))
	}

	// ===== 指针处理 =====
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return 0
		}
		return ToUint64(rv.Elem().Interface())
	}

	return 0
}

// ToUint64Err 将任意类型转换为 uint64，无法转换或为负数时返回错误
func ToUint64Err(v interface{}) (uint64, error) {
	if v == nil {
		return 0, fmt.Errorf("val is nil")
	}

	switch val := v.(type) {

	// ===== int =====
	case int:
		if val < 0 {
			return 0, fmt.Errorf("int < 0: %d", val)
		}
		return uint64(val), nil
	case int8:
		if val < 0 {
			return 0, fmt.Errorf("int8 < 0: %d", val)
		}
		return uint64(val), nil
	case int16:
		if val < 0 {
			return 0, fmt.Errorf("int16 < 0: %d", val)
		}
		return uint64(val), nil
	case int32:
		if val < 0 {
			return 0, fmt.Errorf("int32 < 0: %d", val)
		}
		return uint64(val), nil
	case int64:
		if val < 0 {
			return 0, fmt.Errorf("int64 < 0: %d", val)
		}
		return uint64(val), nil

	// ===== uint =====
	case uint:
		return uint64(val), nil
	case uint8:
		return uint64(val), nil
	case uint16:
		return uint64(val), nil
	case uint32:
		return uint64(val), nil
	case uint64:
		return val, nil

	// ===== float =====
	case float32:
		if val < 0 {
			return 0, fmt.Errorf("float32 < 0: %f", val)
		}
		return uint64(val), nil
	case float64:
		if val < 0 {
			return 0, fmt.Errorf("float64 < 0: %f", val)
		}
		return uint64(val), nil

	// ===== bool =====
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil

	// ===== string =====
	case string:
		s := strings.TrimSpace(val)
		if s == "" {
			return 0, fmt.Errorf("empty string")
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("parse error: %v", err)
		}
		if f < 0 {
			return 0, fmt.Errorf("negative value: %f", f)
		}
		return uint64(f), nil

	// ===== []byte =====
	case []byte:
		return ToUint64Err(string(val))

	// ===== json.Number =====
	case json.Number:
		i, err := val.Int64()
		if err != nil {
			return 0, err
		}
		if i < 0 {
			return 0, fmt.Errorf("negative json.Number")
		}
		return uint64(i), nil

	// ===== 接口 =====
	case fmt.Stringer:
		return ToUint64Err(val.String())

	case error:
		return 0, fmt.Errorf("cannot convert error to uint64")

	case time.Time:
		return uint64(val.Unix()), nil

	case []rune:
		return ToUint64Err(string(val))
	}

	// ===== 指针 =====
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return 0, fmt.Errorf("nil pointer")
		}
		return ToUint64Err(rv.Elem().Interface())
	}

	return 0, fmt.Errorf("unsupported type: %T", v)
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

// ToBool 将任意类型转换为 bool，无法转换时返回 false 不适用高标准环境
func ToBool(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case *bool:
		if val != nil {
			return *val
		}

	case string:
		s := strings.TrimSpace(strings.ToLower(val))
		if s == "true" || s == "1" || s == "yes" || s == "y" || s == "on" {
			return true
		}
		return false
	case *string:
		if val != nil {
			s := strings.TrimSpace(strings.ToLower(*val))
			if s == "true" || s == "1" || s == "yes" || s == "y" || s == "on" {
				return true
			}
		}

	case []byte:
		s := strings.TrimSpace(strings.ToLower(string(val)))
		if s == "true" || s == "1" || s == "yes" || s == "y" || s == "on" {
			return true
		}
		return false
	case *([]byte):
		if val != nil {
			s := strings.TrimSpace(strings.ToLower(string(*val)))
			if s == "true" || s == "1" || s == "yes" || s == "y" || s == "on" {
				return true
			}
		}

	case int:
		return val != 0
	case *int:
		if val != nil {
			return *val != 0
		}

	case int64:
		return val != 0
	case *int64:
		if val != nil {
			return *val != 0
		}

	case int32:
		return val != 0
	case *int32:
		if val != nil {
			return *val != 0
		}

	case float64:
		return val != 0
	case *float64:
		if val != nil {
			return *val != 0
		}

	case float32:
		return val != 0
	case *float32:
		if val != nil {
			return *val != 0
		}

	case json.Number:
		f, err := val.Float64()
		if err == nil {
			return f != 0
		}

	default:
		if v == nil {
			return false
		}
		fmt.Printf("⚡ ToBool遇到未知类型：%T -> %+v\n", v, v)
	}

	return false
}

func parseBoolStr(s string) (bool, error) {
	str := strings.TrimSpace(strings.ToLower(s))

	switch str {
	case "true", "1", "yes", "y", "on":
		return true, nil
	case "false", "0", "no", "n", "off":
		return false, nil
	default:
		return false, fmt.Errorf("无法将字符串转换为bool: %s", s)
	}
}

// ToBoolErr 将任意类型转换为 bool，无法转换时返回 error（适用于严格环境）
func ToBoolErr(v interface{}) (bool, error) {
	switch val := v.(type) {
	case bool:
		return val, nil
	case *bool:
		if val != nil {
			return *val, nil
		}
		return false, fmt.Errorf("nil *bool")

	case string:
		return parseBoolStr(val)
	case *string:
		if val != nil {
			return parseBoolStr(*val)
		}
		return false, fmt.Errorf("nil *string")

	case []byte:
		return parseBoolStr(string(val))
	case *([]byte):
		if val != nil {
			return parseBoolStr(string(*val))
		}
		return false, fmt.Errorf("nil *[]byte")

	case int:
		return val != 0, nil
	case *int:
		if val != nil {
			return *val != 0, nil
		}
		return false, fmt.Errorf("nil *int")

	case int64:
		return val != 0, nil
	case *int64:
		if val != nil {
			return *val != 0, nil
		}
		return false, fmt.Errorf("nil *int64")

	case int32:
		return val != 0, nil
	case *int32:
		if val != nil {
			return *val != 0, nil
		}
		return false, fmt.Errorf("nil *int32")

	case float64:
		return val != 0, nil
	case *float64:
		if val != nil {
			return *val != 0, nil
		}
		return false, fmt.Errorf("nil *float64")

	case float32:
		return val != 0, nil
	case *float32:
		if val != nil {
			return *val != 0, nil
		}
		return false, fmt.Errorf("nil *float32")

	case json.Number:
		f, err := val.Float64()
		if err != nil {
			return false, fmt.Errorf("json.Number 转 float64 失败: %w", err)
		}
		return f != 0, nil

	default:
		if v == nil {
			return false, fmt.Errorf("nil value")
		}
		return false, fmt.Errorf("不支持的类型: %T", v)
	}
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

// IsFileDirExists 判断文件或目录是否存在 返回true表示存在
func IsFileDirExists(path string) bool {
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
		return []byte(""), err
	}

	// 统一去掉UTF-8 BOM头，调用方完全不用关心
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	return data, nil
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
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %v", err)
	}

	// 检测BOM头 0xEF 0xBB 0xBF
	// 如果有BOM说明是我们自己写出的UTF-8文件，100%确定，直接跳过
	if bytes.HasPrefix(content, []byte{0xEF, 0xBB, 0xBF}) {
		return nil
	}

	// 用chardet检测编码
	detector := chardet.NewTextDetector()
	result, err := detector.DetectBest(content[:min(len(content), 4096)])
	if err != nil {
		return fmt.Errorf("编码检测失败: %v", err)
	}

	// 如果chardet认为是UTF-8（没有BOM的UTF-8文件，非本程序产生的）
	if strings.EqualFold(result.Charset, "UTF-8") {
		return nil
	}

	if !strings.Contains(strings.ToUpper(result.Charset), "GB") {
		return fmt.Errorf("不支持的编码类型: %s", result.Charset)
	}

	fmt.Println("检测到GBK/GB2312，开始转换...")

	reader := transform.NewReader(
		bytes.NewReader(content),
		simplifiedchinese.GBK.NewDecoder(),
	)
	converted, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("GB2312 转换失败: %v", err)
	}

	// 写入时加上UTF-8 BOM头，作为"已转换"的永久标记
	bom := []byte{0xEF, 0xBB, 0xBF}
	withBom := append(bom, converted...)

	if err = os.WriteFile(filePath, withBom, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	fmt.Println("转换完成，已写入带BOM的UTF-8文件")
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
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("打开文件失败: %v", err)
	}

	//  检测BOM，是本程序写出的UTF-8文件，100%确定直接跳过
	if bytes.HasPrefix(content, []byte{0xEF, 0xBB, 0xBF}) {
		return nil
	}

	// chardet检测编码，此时面对的一定是非BOM文件
	sample := content
	if len(sample) > 10240 {
		sample = sample[:10240]
	}

	detector := chardet.NewTextDetector()
	result, err := detector.DetectBest(sample)
	if err != nil {
		return fmt.Errorf("编码检测失败: %v", err)
	}
	fmt.Println("检测到编码:", result.Charset)

	// chardet明确识别为UTF-8（外部来源的无BOM文件）
	if strings.EqualFold(result.Charset, "UTF-8") {
		return nil
	}

	// 找对应解码器
	enc := getEncoding(result.Charset)
	if enc == nil {
		return fmt.Errorf("暂不支持的编码类型: %s", result.Charset)
	}

	fmt.Printf("文件是 %s 编码，开始转换为 UTF-8...\n", result.Charset)

	reader := transform.NewReader(bytes.NewReader(content), enc.NewDecoder())
	converted, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("%s 转换失败: %v", result.Charset, err)
	}

	//  写入时加BOM，作为"已转换"的永久标记，下次调用直接第一关跳过
	bom := []byte{0xEF, 0xBB, 0xBF}
	if err = os.WriteFile(filePath, append(bom, converted...), 0644); err != nil {
		return fmt.Errorf("写入 UTF-8 文件失败: %v", err)
	}

	fmt.Println("转换完成，文件已保存为带BOM的UTF-8编码")
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

// CSVFieldMapper 自动识别和映射 CSV 表头及行数据，提供CSV字段名称直接取出字段对应的值,如果CSV 文件中每个字段是 "" 引号引起来的则需要 使用CleanStrings() 先去除首位的引号
//
// 参数:
//   - fields: []string 传入 CSV 的表头或行数据（已按逗号分割好）
//   - mapping: *map[string]int 传址 map
//   - 当 isHeader = true 时：更新表头字段和位置的映射关系
//   - 当 isHeader = false 时：通过字段名获取对应的行值
//   - isHeader: bool - true  → 表示传入的是表头，自动计算字段在 CSV 中的位置并存入 mapping - false → 表示传入的是行数据，根据 mapping 提取对应字段内容
//   - fieldName: string 当 isHeader = false 时，用于指定要取的字段名,注意如果是表头行的时候提供空则使用fields数组索引所有字段所在的位置
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
			_, exists := (*mapping)[field]
			if !exists {
				(*mapping)[field] = i
			} else {
				fmt.Printf("\033[31mError: Duplicate field found in CSV header, please reinitialize the CSV file: %s\033[0m\n", field)
			}
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

// CSVJoinFields 将 CSV 的字段数组组合为一行文本,中间使用英文的逗号连接
//
// 参数:
//
//	fields  []string - CSV 每列的数据
//	quoted  bool     - 是否在每个字段前后加双引号，例如 true -> "value"
//
// 返回:
//
//	string - 拼接后的 CSV 行数据
//
// 注意:
//  1. 如果 fields 为空，则返回空字符串
//  2. 如果 quoted 为 true，会对每个字段加上双引号
//  3. 不会自动处理字段中包含逗号或引号的情况，如需处理请自行转义
func CSVJoinFields(fields []string, quoted bool) string {
	if len(fields) == 0 {
		return ""
	}

	result := ""
	for i, field := range fields {
		if quoted {
			result += `"` + field + `"`
		} else {
			result += field
		}

		// 中间加逗号，最后一个字段不加
		if i < len(fields)-1 {
			result += ","
		}
	}

	return result
}

// CSVFieldEditStrict 在 fields 中根据 mapping 找到 fieldName 对应的位置并替换为 value。
// 严格模式：如果 mapping 缺失字段或索引超出当前 fields 长度，则返回错误（不会修改原切片）。
//
// 参数:
// - fields: CSV 行的字段切片（例如从 strings.Split(line, ",") 得到的）
// - mapping: 头->列索引映射，nil 会返回错误
// - fieldName: 要修改的字段名（应在 mapping 中存在）
// - value: 要写入的值
//
// 返回：修改后的 fields 切片（可能是同一个切片或新的切片）和错误
func CSVFieldEditStrict(fields []string, mapping map[string]int, fieldName, value string) ([]string, error) {
	if mapping == nil {
		return fields, errors.New("mapping cannot be nil")
	}
	idx, ok := mapping[fieldName]
	if !ok {
		return fields, fmt.Errorf("field %q not found in mapping", fieldName)
	}
	if idx < 0 {
		return fields, fmt.Errorf("invalid index %d for field %q", idx, fieldName)
	}
	if idx >= len(fields) {
		return fields, fmt.Errorf("index %d out of range (fields length %d) for field %q", idx, len(fields), fieldName)
	}
	// 直接修改并返回
	fields[idx] = value
	return fields, nil
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

// getFileLock 获取指定文件的锁
// 如果锁不存在则创建
func getFileLock(path string) *sync.Mutex {

	// LoadOrStore：
	// 如果 key 存在则返回已有值
	// 如果不存在则存入新值
	lock, _ := fileLocks.LoadOrStore(path, &sync.Mutex{})

	return lock.(*sync.Mutex)
}

///////////////////////////////////////////////////////////////
// 高性能线程安全写文件函数
///////////////////////////////////////////////////////////////

/*
WriteToFile 高性能线程安全写文件函数

参数说明：

filePath

	要写入的文件路径

data

	要写入的数据
	支持类型：
	    string
	    []byte

mode

	写入模式：
	    0 = 覆盖写入
	    1 = 追加到文件尾部
	    2 = 插入到文件开头

函数特性：

1 支持文件不存在自动创建
2 支持多 goroutine 并发写
3 每个文件独立锁（避免全局锁影响性能）

使用方法:

	WriteToFile("a.txt", "hello", FileAppend) //追加到尾部
	WriteToFile("a.txt", "hello", 2) //插入到开头
	WriteToFile("a.txt", "hello", 0) //覆盖写
*/
func WriteToFile(filePath string, data any, mode int) error {
	// 获取该文件对应的锁

	lock := getFileLock(filePath)

	// 加锁
	lock.Lock()

	// 函数结束时自动解锁
	defer lock.Unlock()

	// 将 data 转换为 []byte
	bytesData := ToBytes(data)

	// 根据写入模式执行不同操作
	switch mode {
	// 覆盖写入模式
	case FileOverwrite:

		/*
			os.WriteFile 行为：

			1 如果文件不存在 -> 自动创建
			2 如果文件存在 -> 清空文件内容
			3 然后写入新数据
		*/

		return os.WriteFile(filePath, bytesData, 0644)

	// 追加到文件尾部
	case FileAppend:

		/*
			打开文件参数说明：

			os.O_CREATE  文件不存在则创建
			os.O_WRONLY  只写模式
			os.O_APPEND  写入位置始终在文件末尾
		*/

		f, err := os.OpenFile(
			filePath,
			os.O_CREATE|os.O_WRONLY|os.O_APPEND,
			0644,
		)

		if err != nil {
			return err
		}

		// 函数结束自动关闭文件
		defer f.Close()

		// 写入数据
		_, err = f.Write(bytesData)

		return err

	// 插入到文件开头
	case FilePrepend:

		/*
			Go 标准库没有直接提供“头部插入”功能
			需要手动实现：

			步骤：

			1 读取旧文件内容
			2 新数据 + 旧数据 拼接
			3 覆盖写入文件
		*/

		var old []byte

		// 判断文件是否存在
		if _, err := os.Stat(filePath); err == nil {

			// 读取旧内容
			old, err = os.ReadFile(filePath)

			if err != nil {
				return err
			}
		}

		// 拼接数据
		newData := append(bytesData, old...)
		// 覆盖写入
		return os.WriteFile(filePath, newData, 0644)
	default:

		return fmt.Errorf("未知写入模式")
	}
}

/*
DomainToIP

将域名解析为 IP 地址列表（支持 IPv4 和 IPv6）

参数：

	domain  要解析的域名，例如 example.com

返回值：

	[]string  解析得到的 IP 地址列表
	bool      是否解析成功

返回规则：

	成功：
	    ["93.184.216.34", "2606:2800:220:1:248:1893:25c8:1946"], true

	失败：
	    nil, false
*/
func DomainToIP(domain string) ([]string, bool) {

	// 调用系统 DNS 解析
	ips, err := net.LookupIP(domain)

	// 如果解析失败
	if err != nil || len(ips) == 0 {
		return nil, false
	}

	// 存储最终 IP 字符串
	var results []string

	// 遍历解析结果
	for _, ip := range ips {

		// 转换为字符串
		results = append(results, ip.String())
	}

	return results, true
}

// FormatURL 自动为 URL 添加 http:// 或 https:// 前缀，默认添加https://
// 参数 urlStr: 待格式化的 URL 字符串
// 返回值: 格式化后的 URL 字符串
func FormatURL(urlStr string) string {
	// 去掉首尾空格
	urlStr = strings.TrimSpace(urlStr)
	if urlStr == "" {
		return urlStr
	}

	// 判断是否已经包含协议前缀
	if strings.HasPrefix(urlStr, "http://") || strings.HasPrefix(urlStr, "https://") {
		return urlStr // 已经有协议，直接返回
	}

	// 默认添加 https:// 前缀
	return "https://" + urlStr
}

////////////////////////////////////////////////////////////
// HashCalc
////////////////////////////////////////////////////////////

/*
HashCalc 计算 hash 值

参数：

input

	输入内容
	如果 mode=HashInputText 表示文本
	如果 mode=HashInputFile 表示文件路径

mode

	输入模式
	1 = 文本
	2 = 文件

hashType hash类型:

	HashMD5    = 1
	HashSHA1   = 2
	HashSHA224 = 3
	HashSHA256 = 4
	HashSHA384 = 5
	HashSHA512 = 6
	HashCRC32  = 7
	HashCRC64  = 8

返回：

string  hash值(hex)
bool    是否成功
*/
// CalcHash 支持 []byte 或文件路径（string）
// data 可以是 []byte 或文件路径 string
func HashCalc(data interface{}, mode int, hashType int) (string, bool) {
	var reader io.Reader
	if mode == 1 {
		switch v := data.(type) {
		case []byte:
			reader = bytes.NewReader(v) // 内存数据
		case string:
			reader = bytes.NewReader([]byte(v))
		default:
			reader = bytes.NewReader(ToBytes(v))
		}
	} else {
		file, err := os.Open(ToStr(data))
		if err != nil {
			return "", false
		}
		defer file.Close()
		reader = file // 文件
	}

	var h hash.Hash

	switch hashType {
	case HashMD5:
		h = md5.New()
	case HashSHA1:
		h = sha1.New()
	case HashSHA224:
		h = sha256.New224()
	case HashSHA256:
		h = sha256.New()
	case HashSHA384:
		h = sha512.New384()
	case HashSHA512:
		h = sha512.New()
	case HashCRC32:
		buf, _ := io.ReadAll(reader)
		v := crc32.ChecksumIEEE(buf)
		return fmt.Sprintf("%08x", v), true
	case HashCRC64:
		buf, _ := io.ReadAll(reader)
		table := crc64.MakeTable(crc64.ISO)
		v := crc64.Checksum(buf, table)
		return fmt.Sprintf("%016x", v), true
	default:
		return "", false
	}

	// 对于 hash.Hash，流式计算（支持大文件）
	_, err := io.Copy(h, reader)
	if err != nil {
		return "", false
	}

	return hex.EncodeToString(h.Sum(nil)), true
}

// CsvCleanStrings 处理字符串切片：去除首尾双引号和空格 一般用于处理csv 行分割后的数据
func CsvCleanStrings(input []string) []string {
	result := make([]string, 0, len(input))
	for i, s := range input {
		if i == 0 {
			s = strings.TrimPrefix(s, "\ufeff") //  去掉BOM字符
		}

		// 去除首尾双引号
		s = strings.Trim(s, `"`)
		// 去除首尾空格
		s = strings.TrimSpace(s)
		s = strings.ReplaceAll(s, "\t", "")
		result = append(result, s)
	}
	return result
}

// String 返回 AddressType 的字符串表示
func (a AddressType) String() string {
	switch a {
	case IPv4:
		return "IPv4"
	case IPv6:
		return "IPv6"
	case Domain:
		return "Domain"
	default:
		return "Unknown"
	}
}

// IsIPv4IPv6Domain 判断输入的字符串是域名、IPv4 还是 IPv6
// 参数 addr: 待判断的地址字符串
// 返回值: AddressType 类型 返回 Unknown 标识不是ipv4也不是ipv6也不是域名
func IsIPv4IPv6Domain(addr string) AddressType {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return Unknown
	}

	// 先尝试解析为 IP 地址
	ip := net.ParseIP(addr)
	if ip != nil {
		if ip.To4() != nil {
			return IPv4
		}
		return IPv6
	}

	// 域名正则校验（简单校验，不包含协议和路径）
	domainRegex := `^([a-zA-Z0-9-]{1,63}\.)+[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(domainRegex, addr)
	if matched {
		return Domain
	}

	return Unknown
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

		} else {                          //根本就不存在开始文本
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

// JsonGetByPathStr 根据路径获取 JSON 中的值
// 如果获取失败直接返回空
func JsonGetByPathStr(jsonData []byte, path string, rawJSON ...bool) string {
	dataRaw, err := JsonGetByPath(jsonData, path, rawJSON...)
	if err != nil {
		return ""
	}
	return ToStr(dataRaw)
}

// JsonGetByPath 根据路径获取 JSON 中的值
// jsonData 原始json数据
// path 支持多层, 使用 "." 分隔, 比如 "TcpData.NodeUUID"
// rawJSON 可选参数, 为 true 时返回未解析的原始 JSON 文本（保留原始字节）
func JsonGetByPath(jsonData []byte, path string, rawJSON ...bool) ([]byte, error) {
	if len(path) == 0 {
		return nil, errors.New("empty path")
	}

	if len(rawJSON) > 0 && rawJSON[0] {
		return JsonGetRawByPath(jsonData, path)
	}

	// 原有逻辑：顶层解析成 interface{}
	var data interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, err
	}

	parts := strings.Split(path, ".")
	curr := data

	for _, key := range parts {
		switch node := curr.(type) {
		case map[string]interface{}:
			val, ok := node[key]
			if !ok {
				return nil, fmt.Errorf("JsonGetByPath key not found: %s", key)
			}
			curr = val
		case []interface{}:
			idx, err := strconv.Atoi(key)
			if err != nil || idx < 0 || idx >= len(node) {
				return nil, fmt.Errorf("JsonGetByPath invalid array index: %s", key)
			}
			curr = node[idx]
		default:
			return nil, fmt.Errorf("JsonGetByPath unexpected type at %s", key)
		}
	}

	switch v := curr.(type) {
	case string:
		return []byte(v), nil
	case float64, bool, int, int64:
		return []byte(fmt.Sprintf("%v", v)), nil
	default:
		result, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
}

// JsonGetRawByPath 逐层用 json.RawMessage 保留原始字节，避免二次 Marshal 破坏格式
func JsonGetRawByPath(jsonData []byte, path string) ([]byte, error) {
	parts := strings.Split(path, ".")
	curr := json.RawMessage(jsonData)

	for _, key := range parts {
		// 先尝试解析为 object
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(curr, &obj); err == nil {
			val, ok := obj[key]
			if !ok {
				return nil, fmt.Errorf("JsonGetByPath key not found: %s", key)
			}
			curr = val
			continue
		}

		// 再尝试解析为 array
		var arr []json.RawMessage
		if err := json.Unmarshal(curr, &arr); err == nil {
			idx, err := strconv.Atoi(key)
			if err != nil || idx < 0 || idx >= len(arr) {
				return nil, fmt.Errorf("JsonGetByPath invalid array index: %s", key)
			}
			curr = arr[idx]
			continue
		}

		return nil, fmt.Errorf("JsonGetByPath unexpected type at %s", key)
	}

	return curr, nil
}
