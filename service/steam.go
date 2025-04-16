package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"steam-reviewer/model"
)

func GetSteamData(apiKey, steamID string) (*model.SteamPlayer, []model.SteamGame, error) {
	client := &http.Client{}

	// 玩家信息
	profileURL := fmt.Sprintf("https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v2/?key=%s&steamids=%s", apiKey, steamID)
	resp, err := client.Get(profileURL)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var profile model.SteamPlayerSummary
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, nil, err
	}
	if len(profile.Response.Players) == 0 {
		return nil, nil, fmt.Errorf("未找到玩家信息")
	}
	player := profile.Response.Players[0]

	// 游戏信息
	gameURL := fmt.Sprintf("https://api.steampowered.com/IPlayerService/GetOwnedGames/v1/?key=%s&steamid=%s&include_appinfo=true&include_played_free_games=true", apiKey, steamID)
	resp, err = client.Get(gameURL)
	if err != nil {
		return &player, nil, err
	}
	defer resp.Body.Close()

	body, _ = io.ReadAll(resp.Body)
	var games model.SteamOwnedGames
	if err := json.Unmarshal(body, &games); err != nil {
		return &player, nil, err
	}

	sort.Slice(games.Response.Games, func(i, j int) bool {
		return games.Response.Games[i].PlaytimeForever > games.Response.Games[j].PlaytimeForever
	})

	return &player, games.Response.Games, nil
}
