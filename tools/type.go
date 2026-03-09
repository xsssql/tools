package tools // 枚举当前目录下全部文件
import (
	"sync"
	"time"
)

type FileInfoExts struct {
	Path    string
	ModTime time.Time
}

var fileLocks sync.Map

const (
	FileOverwrite = 0 // 覆盖写
	FileAppend    = 1 // 追加到尾部
	FilePrepend   = 2 // 插入到开头
)

////////////////////////////////////////////////////////////
// 输入模式
////////////////////////////////////////////////////////////

const (
	HashInputText = 1
	HashInputFile = 2
)

////////////////////////////////////////////////////////////
// Hash 类型
////////////////////////////////////////////////////////////

const (
	HashMD5    = 1
	HashSHA1   = 2
	HashSHA224 = 3
	HashSHA256 = 4
	HashSHA384 = 5
	HashSHA512 = 6
	HashCRC32  = 7
	HashCRC64  = 8
)

// AddressType 用于返回地址类型
type AddressType int

const (
	Unknown AddressType = iota // 未知类型
	IPv4                       // IPv4 地址
	IPv6                       // IPv6 地址
	Domain                     // 域名
)
