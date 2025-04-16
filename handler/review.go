package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"steam-reviewer/model"
	"steam-reviewer/service"
	"steam-reviewer/util"
)

func HandleReview(c *gin.Context) {
	steamID := c.Query("steamid")
	if steamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少steamid参数"})
		return
	}

	force := c.Query("force") == "true"
	dataPath := filepath.Join("./data", steamID+".json")

	// 若不强制刷新且文件存在
	if !force && util.FileExists(dataPath) {
		c.File(dataPath)
		return
	}

	steamKey := os.Getenv("STEAM_API_KEY")
	player, games, err := service.GetSteamData(steamKey, steamID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	prompt := util.BuildSavagePrompt(player, games)
	review, err := service.GenerateReview(prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reviewData := model.ReviewData{
		Player:      *player,
		Games:       games,
		Review:      review,
		GeneratedAt: time.Now(),
	}

	if err := util.SaveReviewData(steamID, &reviewData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, reviewData)
}
