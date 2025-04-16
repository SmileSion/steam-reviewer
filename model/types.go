package model

import "time"

type SteamPlayer struct {
	SteamID        string `json:"steamid"`
	PersonaName    string `json:"personaname"`
	AvatarFull     string `json:"avatarfull"`
	TimeCreated    int    `json:"timecreated"`
	LocCountryCode string `json:"loccountrycode"`
}

type SteamGame struct {
	AppID           int    `json:"appid"`
	Name            string `json:"name"`
	PlaytimeForever int    `json:"playtime_forever"`
}

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

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type DeepSeekRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Stream      bool      `json:"stream"`
	Temperature float64   `json:"temperature"`
}

type ReviewData struct {
	Player      SteamPlayer `json:"player"`
	Games       []SteamGame `json:"games"`
	Review      string      `json:"review"`
	GeneratedAt time.Time   `json:"generated_at"`
}
