package handler

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"steam-reviewer/util"
)

func HandleCheckData(c *gin.Context) {
	steamID := c.Query("steamid")
	if steamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "steamid参数必须提供"})
		return
	}

	dataPath := filepath.Join("./data", steamID+".json")
	c.JSON(http.StatusOK, gin.H{
		"exists":    util.FileExists(dataPath),
		"data_path": dataPath,
	})
}
