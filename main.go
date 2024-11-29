package main

import (
	"encoding/json"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	"os/exec"
	"strings"

	"github.com/gogf/gf/v2/os/glog"
)

type MusicInfo struct {
	IsPlaying  bool   `json:"is_playing"`
	TrackName  string `json:"track_name"`
	ArtistName string `json:"artist_name"`
	AlbumName  string `json:"album_name"`
	AlbumCover string `json:"album_cover"` // 将存储 base64 编码的图片数据
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
	// 首先获取基本信息
	infoScript := `
	tell application "Music"
		if player state is playing then
			set currentTrack to current track
			return {name of currentTrack, artist of currentTrack, album of currentTrack}
		else
			return {"", "", ""}
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
	
	// 解析基本信息
	result = strings.TrimPrefix(result, "{")
	result = strings.TrimSuffix(result, "}")
	parts := strings.Split(result, ", ")
	
	if len(parts) < 3 || parts[0] == "" {
		return &MusicInfo{IsPlaying: false}, nil
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
		TrackName:  strings.Trim(parts[0], "\""),
		ArtistName: strings.Trim(parts[1], "\""),
		AlbumName:  strings.Trim(parts[2], "\""),
		AlbumCover: coverDataStr,
	}

	return musicInfo, nil
}
