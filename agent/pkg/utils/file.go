package utils

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FileInfo 文件信息结构
type FileInfo struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	ModTime int64  `json:"mod_time"`
	IsDir   bool   `json:"is_dir"`
	MD5Hash string `json:"md5_hash,omitempty"`
}

// GetFileInfo 获取文件信息
func GetFileInfo(path string) (*FileInfo, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	info := &FileInfo{
		Path:    path,
		Size:    stat.Size(),
		Mode:    stat.Mode().String(),
		ModTime: stat.ModTime().Unix(),
		IsDir:   stat.IsDir(),
	}

	// 如果是文件，计算MD5
	if !stat.IsDir() {
		hash, err := calculateMD5(path)
		if err == nil {
			info.MD5Hash = hash
		}
	}

	return info, nil
}

// calculateMD5 计算文件MD5值
func calculateMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// EnsureDir 确保目录存在
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

// CopyFile 复制文件
func CopyFile(src, dst string) error {
	// 确保目标目录存在
	if err := EnsureDir(filepath.Dir(dst)); err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
