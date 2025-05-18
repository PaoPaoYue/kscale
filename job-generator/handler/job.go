package handler

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/paopaoyue/kscale/job-genrator/config"
	"github.com/paopaoyue/kscale/job-genrator/core"
	"net/http"
	"path/filepath"
	"strings"
)

func SubmitJobHandler(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File upload failed"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error opening file"})
		return
	}
	defer src.Close()

	err = core.Scheduler.SubmitJobs(strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename)), src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Jobs uploaded successfully",
	})
}

func DownloadResultHandler(c *gin.Context) {
	batchName := c.DefaultQuery("batchname", "")

	if batchName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "batchname is required"})
		return
	}

	filePath := filepath.Join(config.C.OutputFilePath, fmt.Sprintf("%s-result.csv", batchName))

	if _, err := filepath.Abs(filePath); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("File %s not found", filePath)})
		return
	}

	c.File(filePath)
}
