package tools // 枚举当前目录下全部文件
import "time"

type FileInfoExts struct {
	Path    string
	ModTime time.Time
}
