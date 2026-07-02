package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os/exec"
)

type videoData struct {
	Streams []Streams `json:"streams"`
}

type Streams struct {
	Width	int	`json:"width,omitempty"`
	Height	int	`json:"height,omitempty"`
}

func getVideoAspectRatio(filePath string) (string, error) {
	var b bytes.Buffer

	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	cmd.Stdout = &b

	err := cmd.Run()
	if err != nil {
		log.Panic(err)
	}

	var videoInfo videoData
	err = json.Unmarshal(b.Bytes(), &videoInfo)
	if err != nil {
		log.Panic(err)
	}

	// an aspect ratio of 16:9 = 1.78 | 9:16 = 0.56

	var aspectRatio string
	
	width := videoInfo.Streams[0].Width
	height := videoInfo.Streams[0].Height

	fmt.Printf("width: %d, height: %d\n", width, height)
	
	ratio := float64(width) / float64(height)

	if math.Abs(ratio - (16.0/9.0)) < 0.01 {
		aspectRatio = "16:9"
	} else if math.Abs(ratio - (9.0/16.0)) < 0.01 {
		aspectRatio = "9:16"
	} else {
		aspectRatio = "other"
	}
	
	return aspectRatio, nil 
}
