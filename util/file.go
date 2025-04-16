package util

import (
	"encoding/json"
	"os"
	"path/filepath"
	"steam-reviewer/model"
)

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func SaveReviewData(steamID string, data *model.ReviewData) error {
	path := filepath.Join("./data", steamID+".json")
	tmp := path + ".tmp"

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, bytes, 0644); err != nil {
		return err
	}

	return os.Rename(tmp, path)
}
