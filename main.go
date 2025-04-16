package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"
)

// Steam 玩家简化结构
type SteamPlayer struct {
	SteamID        string `json:"steamid"`
	PersonaName    string `json:"personaname"`
	AvatarFull     string `json:"avatarfull"`
	TimeCreated    int    `json:"timecreated"`
	LocCountryCode string `json:"loccountrycode"`
}

// Steam 游戏简化结构
type SteamGame struct {
	AppID           int    `json:"appid"`
	Name            string `json:"name"`
	PlaytimeForever int    `json:"playtime_forever"`
	Playtime2Weeks  int    `json:"playtime_2weeks"`
}

// 原始 API 响应结构
type SteamPlayerSummary struct {
	Response struct {
		Players []SteamPlayer `json:"players"`
	} `json:"response"`
}

type SteamOwnedGames struct {
	Response struct {
		GameCount int         `json:"game_count"`
		Games     []SteamGame `json:"games"`
	} `json:"response"`
}

// DeepSeek 结构
type DeepSeekRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Stream      bool      `json:"stream"`
	Temperature float64   `json:"temperature"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type DeepSeekStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

type DeepSeekStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

type ReviewData struct {
	Player      SteamPlayer `json:"player"`
	Games       []SteamGame `json:"games"`
	Review      string      `json:"review"`
	GeneratedAt time.Time   `json:"generated_at"`
}

func main() {
	// 创建数据目录
	os.MkdirAll("./data", 0755)

	http.HandleFunc("/review", handleReview)
	http.HandleFunc("/check-data", handleCheckData)
	http.Handle("/", http.FileServer(http.Dir("./public")))

	fmt.Println("启动服务：http://localhost:9010")
	http.ListenAndServe(":9010", nil)
}

var saveGroup singleflight.Group

func handleCheckData(w http.ResponseWriter, r *http.Request) {
	steamID := r.URL.Query().Get("steamid")
	if steamID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "steamid参数必须提供",
		})
		return
	}

	dataPath := filepath.Join("./data", fmt.Sprintf("%s.json", steamID))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"exists":    fileExists(dataPath),
		"data_path": dataPath,
	})
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func handleReview(w http.ResponseWriter, r *http.Request) {
	steamAPIKey := "A0453261FC35DFAC250BFC0C1510878C"
	deepseekAPIKey := "sk-5a5ca94b00fa4e8f9115e3f2bfc72ed2"

	steamID := r.URL.Query().Get("steamid")
	if steamID == "" {
		http.Error(w, "缺少steamid参数", http.StatusBadRequest)
		return
	}

	player, games, err := getSteamData(steamAPIKey, steamID)
	if err != nil {
		http.Error(w, fmt.Sprintf("获取Steam数据失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 检查是否强制刷新参数
	forceRefresh := r.URL.Query().Get("force") == "true"
	dataPath := filepath.Join("./data", fmt.Sprintf("%s.json", steamID))

	// 如果文件存在且不强制刷新，直接返回现有数据
	if !forceRefresh {
		if _, err := os.Stat(dataPath); err == nil {
			existingData, err := os.ReadFile(dataPath)
			if err == nil {
				w.Header().Set("Content-Type", "application/json")
				w.Write(existingData)
				return
			}
		}
	}

	prompt := buildSavagePrompt(player, games)
	review, err := generateDeepSeekReview(deepseekAPIKey, prompt)
	if err != nil {
		http.Error(w, fmt.Sprintf("生成锐评失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 保存数据
	reviewData := ReviewData{
		Player:      *player,
		Games:       games,
		Review:      review,
		GeneratedAt: time.Now(),
	}

	err = saveReviewData(steamID, &reviewData)
	if err != nil {
		http.Error(w, fmt.Sprintf("保存数据失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"player": player,
		"games":  games,
		"review": review,
	})

}

func generateDeepSeekReview(apiKey, prompt string) (string, error) {
	requestBody := DeepSeekRequest{
		Model: "deepseek-reasoner",
		Messages: []Message{
			{
				Role:    "system",
				Content: "你是一个既毒舌又幽默风趣的AI助手，擅长吐槽和反鸡汤。",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   8000,
		Temperature: 1.5,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("JSON编码失败: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.deepseek.com/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API返回错误: %s", string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("未获得有效响应")
	}

	return result.Choices[0].Message.Content, nil
}

func saveReviewData(steamID string, data *ReviewData) error {
	filename := filepath.Join("./data", fmt.Sprintf("%s.json", steamID))

	// 步骤1：先将数据写入临时文件
	tmpfile := filename + ".tmp"
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON编码失败: %v", err)
	}

	if err := ioutil.WriteFile(tmpfile, content, 0644); err != nil {
		return fmt.Errorf("写入临时文件失败: %v", err)
	}

	// 步骤2：原子替换（这才是保障并发安全的关键操作）
	return os.Rename(tmpfile, filename)
}

func getSteamData(apiKey, steamID string) (*SteamPlayer, []SteamGame, error) {
	client := &http.Client{}

	// 获取玩家信息
	profileURL := fmt.Sprintf("https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v2/?key=%s&steamids=%s", apiKey, steamID)
	req, err := http.NewRequest("GET", profileURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("请求Steam概要信息失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("读取玩家响应失败: %v", err)
	}

	var profile SteamPlayerSummary
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, nil, fmt.Errorf("解析玩家信息JSON失败: %v", err)
	}
	if len(profile.Response.Players) == 0 {
		return nil, nil, fmt.Errorf("未找到玩家信息")
	}
	player := profile.Response.Players[0]

	// 获取游戏信息
	gamesURL := fmt.Sprintf("https://api.steampowered.com/IPlayerService/GetOwnedGames/v1/?key=%s&steamid=%s&include_appinfo=true&include_played_free_games=true", apiKey, steamID)
	req, err = http.NewRequest("GET", gamesURL, nil)
	if err != nil {
		return &player, nil, fmt.Errorf("创建游戏请求失败: %v", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err = client.Do(req)
	if err != nil {
		return &player, nil, fmt.Errorf("请求游戏信息失败: %v", err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return &player, nil, fmt.Errorf("读取游戏响应失败: %v", err)
	}

	var games SteamOwnedGames
	if err := json.Unmarshal(body, &games); err != nil {
		return &player, nil, fmt.Errorf("解析游戏信息JSON失败: %v", err)
	}

	// 排序游戏
	sort.Slice(games.Response.Games, func(i, j int) bool {
		return games.Response.Games[i].PlaytimeForever > games.Response.Games[j].PlaytimeForever
	})

	return &player, games.Response.Games, nil
}

func buildSavagePrompt(player *SteamPlayer, games []SteamGame) string {
	// var topGames []string
	// for i, game := range games {
	// 	if i >= 10 {
	// 		break
	// 	}
	// 	hours := float64(game.PlaytimeForever) / 60
	// 	topGames = append(topGames, fmt.Sprintf("%s (%.1f小时)", game.Name, hours))
	// }
	var allGames []string
	for _, game := range games { // 移除i >= 10的限制
		hours := float64(game.PlaytimeForever) / 60
		allGames = append(allGames, fmt.Sprintf("%s (%.1f小时)", game.Name, hours))
	}

	return fmt.Sprintf(`请为以下Steam玩家游戏库生成毒舌幽默的锐评，要求：
1. 使用网络流行梗和幽默讽刺的语气
2. 分析游戏类型分布和游玩模式
3. 指出矛盾点和有趣现象
4. 最后给出游戏推荐，标题有趣一点，类似“【补货推荐】”
5. 包含一个玩家总结和彩蛋，玩家总结标题和彩蛋标题也该有趣一点，类似“反鸡汤总结”这样的。

玩家信息：
- 昵称: %s
- 注册时间: %s
- 总游戏数: %d
- 总游戏时长: %.1f小时
- 全部游戏: %s

请用中文生成，风格参考B站热门吐槽视频，分5-6个板块，每个板块有创意小标题。`,
		player.PersonaName,
		time.Unix(int64(player.TimeCreated), 0).Format("2006年"),
		len(games),
		calculateTotalHours(games),
		strings.Join(allGames, "，"),
	)
}

func calculateTotalHours(games []SteamGame) float64 {
	total := 0
	for _, game := range games {
		total += game.PlaytimeForever
	}
	return float64(total) / 60
}
