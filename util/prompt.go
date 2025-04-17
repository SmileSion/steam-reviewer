package util

import (
	"fmt"
	"strings"
	"time"

	"steam-reviewer/model"
)

func BuildSavagePrompt(player *model.SteamPlayer, games []model.SteamGame) string {
	var allGames []string
	total := 0
	for _, g := range games {
		hours := float64(g.PlaytimeForever) / 60
		allGames = append(allGames, fmt.Sprintf("%s (%.1f小时)", g.Name, hours))
		total += g.PlaytimeForever
	}

	return fmt.Sprintf(`请为以下Steam玩家游戏库生成毒舌幽默的锐评，要求：
1. 使用网络流行梗和幽默讽刺的语气
2. 分析游戏类型分布和游玩模式
3. 指出矛盾点和有趣现象
4. 最后给出游戏推荐，标题有趣一点，类似“【补货推荐】”
5. 包含一个玩家总结和彩蛋，玩家总结标题和彩蛋标题也该有趣一点，类似“反鸡汤总结”这样的。
不要生成markdown格式的！我需要直接展示在前端，可以生成html样式的内容，不需要背景颜色块，不同模块用横杆区分即可，文字颜色和大小可以多样。

玩家信息：
- 昵称: %s
- 注册时间: %s
- 总游戏数: %d
- 总游戏时长: %.1f小时
- 全部游戏: %s`,
		player.PersonaName,
		time.Unix(int64(player.TimeCreated), 0).Format("2006年"),
		len(games),
		float64(total)/60,
		strings.Join(allGames, "，"),
	)
}
