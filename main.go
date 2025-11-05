package main

import (
	"encoding/json"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gogf/gf/v2/os/glog"
)

type MusicInfo struct {
	IsPlaying  bool   `json:"is_playing"`
	TrackName  string `json:"track_name"`
	ArtistName string `json:"artist_name"`
	AlbumName  string `json:"album_name"`
	AlbumCover string `json:"album_cover"`
	Progress   string `json:"progress"` // 新增进度字段
}

func main() {
	musicInfo, err := getAppleMusicInfo()
	if err != nil {
		glog.Error(nil, err)
		return
	}

	// 转换为 JSON 并输出
	jsonData, err := json.Marshal(musicInfo)
	if err != nil {
		glog.Error(nil, "转换 JSON 失败:", err)
		return
	}
	fmt.Println(string(jsonData))
}

func getAppleMusicInfo() (*MusicInfo, error) {
	// 修改基本信息脚本，使用特殊分隔符避免逗号冲突
	infoScript := `
	tell application "Music"
		if player state is playing then
			set currentTrack to current track
			set trackProgress to (player position / (duration of currentTrack)) * 100
			set trackName to name of currentTrack
			set artistName to artist of currentTrack
			set albumName to album of currentTrack
			return trackName & "|||" & artistName & "|||" & albumName & "|||" & trackProgress
		else
			return ""
		end if
	end tell
	`

	// 获取专辑封面
	artworkScript := `
	tell application "Music"
		try
			if player state is not stopped then
				set currentTrack to current track
				tell artwork 1 of currentTrack
					if format is JPEG picture then
						set imgFormat to ".jpg"
					else
						set imgFormat to ".png"
					end if
				end tell
				
				set tempPath to (POSIX path of (path to temporary items)) & "temp" & imgFormat
				set rawData to raw data of artwork 1 of currentTrack
				
				try
					set fileRef to (open for access POSIX file tempPath with write permission)
					write rawData to fileRef starting at 0
					close access fileRef
					
					set coverData to (do shell script "base64 < " & quoted form of tempPath)
					do shell script "rm " & quoted form of tempPath
					return coverData
				on error errMsg
					log errMsg
					try
						close access fileRef
					end try
					return ""
				end try
			end if
		on error
			return ""
		end try
	end tell
	`

	cmd := exec.Command("osascript", "-e", infoScript)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("执行 AppleScript 出错: %v", err)
	}

	result := strings.TrimSpace(string(output))

	// 如果没有播放，返回空信息
	if result == "" {
		return &MusicInfo{IsPlaying: false}, nil
	}

	// 使用特殊分隔符 ||| 进行分割
	parts := strings.Split(result, "|||")

	if len(parts) < 4 || parts[0] == "" {
		return &MusicInfo{IsPlaying: false}, nil
	}

	// 解析进度百分比
	progress := 0.0
	if progressStr := strings.TrimSpace(parts[3]); progressStr != "" {
		if p, err := strconv.ParseFloat(progressStr, 64); err == nil {
			progress = p
		}
	}

	// 获取专辑封面
	cmd = exec.Command("osascript", "-e", artworkScript)
	coverData, err := cmd.Output()
	if err != nil {
		fmt.Printf("获取专辑封面出错: %v\n", err)
		coverData = []byte("")
	}

	coverDataStr := strings.TrimSpace(string(coverData))

	musicInfo := &MusicInfo{
		IsPlaying:  true,
		TrackName:  strings.TrimSpace(parts[0]),
		ArtistName: strings.TrimSpace(parts[1]),
		AlbumName:  strings.TrimSpace(parts[2]),
		AlbumCover: coverDataStr,
		Progress:   fmt.Sprintf("%.2f", progress),
	}

	return musicInfo, nil
}
