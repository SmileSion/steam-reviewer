package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"steam-reviewer/model"
	"steam-reviewer/service"
	"steam-reviewer/util"

	"github.com/gin-gonic/gin"
)

func HandleReview(c *gin.Context) {
	steamID := c.Query("steamid")
	if steamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少steamid参数"})
		return
	}

	force := c.Query("force") == "true"
	dataPath := filepath.Join("./data", steamID+".json")

	if !force && util.FileExists(dataPath) {
		c.File(dataPath)
		return
	}

	// 设置流式响应头
	c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.WriteHeader(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.String(http.StatusInternalServerError, "Streaming unsupported!")
		return
	}

	// 启动一个心跳协程，避免连接超时（每10秒发送一个空格）
	stopHeartbeat := make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.Writer.Write([]byte(" ")) // 空格保持连接
				flusher.Flush()
			case <-stopHeartbeat:
				return
			}
		}
	}()

	// 执行主逻辑（获取数据并生成）
	steamKey := os.Getenv("STEAM_API_KEY")
	player, games, err := service.GetSteamData(steamKey, steamID)
	if err != nil {
		close(stopHeartbeat)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	prompt := util.BuildSavagePrompt(player, games)
	review, err := service.GenerateReview(prompt)
	if err != nil {
		close(stopHeartbeat)
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
		close(stopHeartbeat)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败: " + err.Error()})
		return
	}

	// 停止心跳协程
	close(stopHeartbeat)
	// 最后一次性输出最终结果（返回完整JSON）
	jsonBytes, err := json.Marshal(reviewData)
	if err != nil {
		c.String(http.StatusInternalServerError, "JSON编码失败")
		return
	}

	c.Writer.Write(jsonBytes)
	flusher.Flush()
}
