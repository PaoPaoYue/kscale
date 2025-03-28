package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/paopaoyue/kscale/job-genrator/core"
	"net/http"
	"path/filepath"
	"strings"
)

func SubmitJobHandler(c *gin.Context) {
	if core.Scheduler.Active {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job scheduler is already active"})
		return
	}

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error submitting jobs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Jobs uploaded successfully",
	})
}
