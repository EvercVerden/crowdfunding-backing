package util

import (
	"path/filepath"
	"strconv"
	"time"
)

// GenerateUniqueFilename 生成唯一的文件名
func GenerateUniqueFilename(originalFilename string) string {
	ext := filepath.Ext(originalFilename)
	name := filepath.Base(originalFilename)
	name = name[:len(name)-len(ext)]

	timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)
	return name + "_" + timestamp + ext
}
