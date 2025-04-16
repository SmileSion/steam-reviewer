package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"steam-reviewer/model"
)

func GenerateReview(prompt string) (string, error) {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")

	request := model.DeepSeekRequest{
		Model: "deepseek-reasoner",
		Messages: []model.Message{
			{Role: "system", Content: "你是一个既毒舌又幽默风趣的AI助手，擅长吐槽和反鸡汤。"},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   8000,
		Temperature: 1.5,
	}

	data, _ := json.Marshal(request)
	req, _ := http.NewRequest("POST", "https://api.deepseek.com/chat/completions", bytes.NewBuffer(data))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API请求失败: %s", string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Choices[0].Message.Content, nil
}
