package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"time"
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
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type DeepSeekResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func main() {
	steamAPIKey := "A0453261FC35DFAC250BFC0C1510878C"        // 填入你的 Steam API Key
	steamID := "76561198403581191"
	deepseekAPIKey := "sk-5a5ca94b00fa4e8f9115e3f2bfc72ed2"     // 填入你的 DeepSeek API Key

	player, games, err := getSteamData(steamAPIKey, steamID)
	if err != nil {
		fmt.Printf("获取Steam数据失败: %v\n", err)
		return
	}

	prompt := buildSavagePrompt(player, games)

	review, err := generateDeepSeekReview(deepseekAPIKey, prompt)
	if err != nil {
		fmt.Printf("生成锐评失败: %v\n", err)
		return
	}

	// Markdown 格式报告输出
	fmt.Printf("\n### Steam游戏库毒舌锐评报告\n")
	fmt.Printf("**玩家:** %s\n", player.PersonaName)
	fmt.Printf("![头像](%s)\n\n", player.AvatarFull) // 头像展示

	// 注册时间、游戏信息
	fmt.Printf("**注册时间:** %s\n", time.Unix(int64(player.TimeCreated), 0).Format("2006-01-02"))
	fmt.Printf("**总游戏数:** %d | **总时长:** %.1f 小时\n\n", len(games), calculateTotalHours(games))

	// 锐评部分
	fmt.Println("```")
	fmt.Println(review)
	fmt.Println("```")

	// 添加DeepSeek生成提示
	fmt.Println("\n*深度锐评来源: DeepSeek-R1生成*")
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
	var topGames []string
	for i, game := range games {
		if i >= 10 {
			break
		}
		hours := float64(game.PlaytimeForever) / 60
		topGames = append(topGames, fmt.Sprintf("%s (%.1f小时)", game.Name, hours))
	}

	return fmt.Sprintf(`请为以下Steam玩家游戏库生成毒舌幽默的锐评，要求：
1. 使用网络流行梗和幽默讽刺的语气
2. 分析游戏类型分布和游玩模式
3. 指出矛盾点和有趣现象
4. 最后给出"补货推荐"
5. 包含一个"反鸡汤总结"

玩家信息：
- 昵称: %s
- 注册时间: %s
- 总游戏数: %d
- 总游戏时长: %.1f小时
- 前10游戏: %s

请用中文生成，风格参考B站热门吐槽视频，分5-6个板块，每个板块有创意小标题。`,
		player.PersonaName,
		time.Unix(int64(player.TimeCreated), 0).Format("2006年"),
		len(games),
		calculateTotalHours(games),
		strings.Join(topGames, "，"),
	)
}

func generateDeepSeekReview(apiKey, prompt string) (string, error) {
	requestBody := DeepSeekRequest{
		Model: "deepseek-reasoner",
		Messages: []Message{
			{
				Role:    "system",
				Content: "你是一个毒舌又幽默的AI助手，擅长吐槽和反鸡汤。",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens: 2000,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.deepseek.com/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result DeepSeekResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("未获得有效响应")
	}

	return result.Choices[0].Message.Content, nil
}

func calculateTotalHours(games []SteamGame) float64 {
	total := 0
	for _, game := range games {
		total += game.PlaytimeForever
	}
	return float64(total) / 60
}
