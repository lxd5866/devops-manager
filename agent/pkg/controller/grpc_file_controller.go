package controller

import (
	"log"

	"devops-manager/agent/pkg/service"
)

// FileGRPCController 文件gRPC业务控制器
type FileGRPCController struct {
	fileService *service.FileService
}

// NewFileGRPCController 创建文件gRPC控制器
func NewFileGRPCController() *FileGRPCController {
	return &FileGRPCController{
		fileService: service.NewFileService("./uploads", "./downloads"),
	}
}

// UploadFile 上传文件（内部方法）
func (fgc *FileGRPCController) UploadFile(fileName string, data []byte) (*service.FileTransfer, error) {
	LogGRPCRequest("UploadFile", fileName)

	transfer, err := fgc.fileService.UploadFile(fileName, data)
	if err != nil {
		LogGRPCResponse("UploadFile", false, err.Error())
		return nil, err
	}

	LogGRPCResponse("UploadFile", true, "File uploaded successfully")
	log.Printf("File uploaded: %s (%d bytes)", fileName, len(data))

	return transfer, nil
}

// DownloadFile 下载文件（内部方法）
func (fgc *FileGRPCController) DownloadFile(fileName string) ([]byte, *service.FileTransfer, error) {
	LogGRPCRequest("DownloadFile", fileName)

	data, transfer, err := fgc.fileService.DownloadFile(fileName)
	if err != nil {
		LogGRPCResponse("DownloadFile", false, err.Error())
		return nil, nil, err
	}

	LogGRPCResponse("DownloadFile", true, "File downloaded successfully")
	log.Printf("File downloaded: %s (%d bytes)", fileName, len(data))

	return data, transfer, nil
}

// ListFiles 列出文件（内部方法）
func (fgc *FileGRPCController) ListFiles(dir string) (interface{}, error) {
	LogGRPCRequest("ListFiles", dir)

	files, err := fgc.fileService.ListFiles(dir)
	if err != nil {
		LogGRPCResponse("ListFiles", false, err.Error())
		return nil, err
	}

	LogGRPCResponse("ListFiles", true, "Files listed successfully")
	log.Printf("Listed %d files in directory: %s", len(files), dir)

	return files, nil
}

// DeleteFile 删除文件（内部方法）
func (fgc *FileGRPCController) DeleteFile(fileName, dir string) error {
	LogGRPCRequest("DeleteFile", fileName)

	err := fgc.fileService.DeleteFile(fileName, dir)
	if err != nil {
		LogGRPCResponse("DeleteFile", false, err.Error())
		return err
	}

	LogGRPCResponse("DeleteFile", true, "File deleted successfully")
	log.Printf("File deleted: %s from %s", fileName, dir)

	return nil
}

// GetFileInfo 获取文件信息（内部方法）
func (fgc *FileGRPCController) GetFileInfo(fileName, dir string) (interface{}, error) {
	LogGRPCRequest("GetFileInfo", fileName)

	info, err := fgc.fileService.GetFileInfo(fileName, dir)
	if err != nil {
		LogGRPCResponse("GetFileInfo", false, err.Error())
		return nil, err
	}

	LogGRPCResponse("GetFileInfo", true, "File info retrieved successfully")
	log.Printf("File info retrieved: %s", fileName)

	return info, nil
}

// VerifyFile 验证文件完整性（内部方法）
func (fgc *FileGRPCController) VerifyFile(fileName, dir, expectedMD5 string) (bool, error) {
	LogGRPCRequest("VerifyFile", fileName)

	isValid, err := fgc.fileService.VerifyFile(fileName, dir, expectedMD5)
	if err != nil {
		LogGRPCResponse("VerifyFile", false, err.Error())
		return false, err
	}

	LogGRPCResponse("VerifyFile", isValid, "File verification completed")
	log.Printf("File verification: %s - %t", fileName, isValid)

	return isValid, nil
}
