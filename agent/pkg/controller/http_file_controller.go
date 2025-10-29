package controller

import (
	"io"
	"net/http"
	"strconv"

	"devops-manager/agent/pkg/service"

	"github.com/gin-gonic/gin"
)

// FileHTTPController 文件HTTP业务控制器
type FileHTTPController struct {
	fileService *service.FileService
}

// NewFileHTTPController 创建文件HTTP控制器
func NewFileHTTPController() *FileHTTPController {
	return &FileHTTPController{
		fileService: service.NewFileService("./uploads", "./downloads"),
	}
}

// UploadFile 上传文件
func (fhc *FileHTTPController) UploadFile(c *gin.Context) {
	LogHTTPRequest(c)

	// 获取上传的文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "No file uploaded")
		return
	}
	defer file.Close()

	// 读取文件内容
	data, err := io.ReadAll(file)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to read file")
		return
	}

	// 上传文件
	transfer, err := fhc.fileService.UploadFile(header.Filename, data)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	SuccessResponse(c, gin.H{
		"transfer_id": transfer.ID,
		"file_name":   transfer.FileName,
		"file_path":   transfer.FilePath,
		"size":        transfer.Size,
		"md5_hash":    transfer.MD5Hash,
		"status":      transfer.Status,
	})
}

// DownloadFile 下载文件
func (fhc *FileHTTPController) DownloadFile(c *gin.Context) {
	LogHTTPRequest(c)

	fileName := c.Param("name")
	if fileName == "" {
		ErrorResponse(c, http.StatusBadRequest, "File name is required")
		return
	}

	// 下载文件
	data, transfer, err := fhc.fileService.DownloadFile(fileName)
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	// 设置响应头
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", strconv.FormatInt(transfer.Size, 10))
	c.Header("X-MD5-Hash", transfer.MD5Hash)

	// 返回文件内容
	c.Data(http.StatusOK, "application/octet-stream", data)
}

// ListFiles 列出文件
func (fhc *FileHTTPController) ListFiles(c *gin.Context) {
	LogHTTPRequest(c)

	dir := c.DefaultQuery("dir", "upload")

	files, err := fhc.fileService.ListFiles(dir)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	SuccessResponse(c, gin.H{
		"directory": dir,
		"files":     files,
		"count":     len(files),
	})
}

// DeleteFile 删除文件
func (fhc *FileHTTPController) DeleteFile(c *gin.Context) {
	LogHTTPRequest(c)

	fileName := c.Param("name")
	if fileName == "" {
		ErrorResponse(c, http.StatusBadRequest, "File name is required")
		return
	}

	dir := c.DefaultQuery("dir", "upload")

	err := fhc.fileService.DeleteFile(fileName, dir)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	SuccessResponse(c, gin.H{
		"message":   "File deleted successfully",
		"file_name": fileName,
		"directory": dir,
	})
}

// GetFileInfo 获取文件信息
func (fhc *FileHTTPController) GetFileInfo(c *gin.Context) {
	LogHTTPRequest(c)

	fileName := c.Param("name")
	if fileName == "" {
		ErrorResponse(c, http.StatusBadRequest, "File name is required")
		return
	}

	dir := c.DefaultQuery("dir", "upload")

	info, err := fhc.fileService.GetFileInfo(fileName, dir)
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	SuccessResponse(c, gin.H{
		"file_info": info,
		"directory": dir,
	})
}
