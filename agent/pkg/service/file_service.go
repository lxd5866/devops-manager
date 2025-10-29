package service

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"devops-manager/agent/pkg/utils"
)

// FileService 文件传输服务
type FileService struct {
	uploadDir   string
	downloadDir string
	mutex       sync.RWMutex
}

// FileTransfer 文件传输记录
type FileTransfer struct {
	ID       string `json:"id"`
	FileName string `json:"file_name"`
	FilePath string `json:"file_path"`
	Size     int64  `json:"size"`
	MD5Hash  string `json:"md5_hash"`
	Status   string `json:"status"` // uploading, completed, failed
	Progress int    `json:"progress"`
	ErrorMsg string `json:"error_msg,omitempty"`
}

// NewFileService 创建文件服务
func NewFileService(uploadDir, downloadDir string) *FileService {
	// 确保目录存在
	utils.EnsureDir(uploadDir)
	utils.EnsureDir(downloadDir)

	return &FileService{
		uploadDir:   uploadDir,
		downloadDir: downloadDir,
	}
}

// UploadFile 上传文件
func (fs *FileService) UploadFile(fileName string, data []byte) (*FileTransfer, error) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	filePath := filepath.Join(fs.uploadDir, fileName)

	// 写入文件
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	// 计算MD5
	hash := md5.Sum(data)
	md5Hash := fmt.Sprintf("%x", hash)

	transfer := &FileTransfer{
		ID:       generateTransferID(),
		FileName: fileName,
		FilePath: filePath,
		Size:     int64(len(data)),
		MD5Hash:  md5Hash,
		Status:   "completed",
		Progress: 100,
	}

	return transfer, nil
}

// DownloadFile 下载文件
func (fs *FileService) DownloadFile(fileName string) ([]byte, *FileTransfer, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	filePath := filepath.Join(fs.downloadDir, fileName)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("file not found: %s", fileName)
	}

	// 读取文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file: %w", err)
	}

	// 计算MD5
	hash := md5.Sum(data)
	md5Hash := fmt.Sprintf("%x", hash)

	transfer := &FileTransfer{
		ID:       generateTransferID(),
		FileName: fileName,
		FilePath: filePath,
		Size:     int64(len(data)),
		MD5Hash:  md5Hash,
		Status:   "completed",
		Progress: 100,
	}

	return data, transfer, nil
}

// ListFiles 列出文件
func (fs *FileService) ListFiles(dir string) ([]*utils.FileInfo, error) {
	var targetDir string
	switch dir {
	case "upload":
		targetDir = fs.uploadDir
	case "download":
		targetDir = fs.downloadDir
	default:
		return nil, fmt.Errorf("invalid directory: %s", dir)
	}

	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []*utils.FileInfo
	for _, entry := range entries {
		filePath := filepath.Join(targetDir, entry.Name())
		fileInfo, err := utils.GetFileInfo(filePath)
		if err != nil {
			continue // 跳过错误的文件
		}
		files = append(files, fileInfo)
	}

	return files, nil
}

// DeleteFile 删除文件
func (fs *FileService) DeleteFile(fileName, dir string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	var targetDir string
	switch dir {
	case "upload":
		targetDir = fs.uploadDir
	case "download":
		targetDir = fs.downloadDir
	default:
		return fmt.Errorf("invalid directory: %s", dir)
	}

	filePath := filepath.Join(targetDir, fileName)

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// GetFileInfo 获取文件信息
func (fs *FileService) GetFileInfo(fileName, dir string) (*utils.FileInfo, error) {
	var targetDir string
	switch dir {
	case "upload":
		targetDir = fs.uploadDir
	case "download":
		targetDir = fs.downloadDir
	default:
		return nil, fmt.Errorf("invalid directory: %s", dir)
	}

	filePath := filepath.Join(targetDir, fileName)
	return utils.GetFileInfo(filePath)
}

// VerifyFile 验证文件完整性
func (fs *FileService) VerifyFile(fileName, dir, expectedMD5 string) (bool, error) {
	fileInfo, err := fs.GetFileInfo(fileName, dir)
	if err != nil {
		return false, err
	}

	return fileInfo.MD5Hash == expectedMD5, nil
}

// generateTransferID 生成传输ID
func generateTransferID() string {
	return fmt.Sprintf("transfer-%d", time.Now().UnixNano())
}
