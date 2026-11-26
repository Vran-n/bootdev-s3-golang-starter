package vidhelper

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os/exec"
)

func whichRatio(number float64) string {
	switch n := math.Floor(number); n {
	case 0:
		return "9:16"
	case 1:
		return "16:9"
	default:
		return "OTHER"
	}
}

func GetVideoAspectRatio(filePath string) (string, error) {
	type ffprobeJSON struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}

	var b bytes.Buffer
	var ffjson ffprobeJSON

	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	cmd.Stdout = &b
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("Failed to run ffprobe with specified parameters: %w", err)
	}

	if err := json.Unmarshal(b.Bytes(), &ffjson); err != nil {
		return "", fmt.Errorf("Unable to unmarshal the commands output json: %w", err)
	}

	if len(ffjson.Streams) == 0 {
		fmt.Printf("%s\n", b.Bytes())
		fmt.Printf("%v\n", ffjson)
		return "", errors.New("Struct has missing metadata.")
	}

	width := ffjson.Streams[0].Width
	height := ffjson.Streams[0].Height

	var ratio float64 = float64(width) / float64(height)
	r := whichRatio(ratio)

	return r, nil
}

func ProcessVideoForFastStart(filePath string) (string, error) {
	outputPath := filePath + ".processing"
	if err := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputPath).Run(); err != nil {
		return "", fmt.Errorf("Failed to run ffmpeg with specified parameters: %w", err)
	}
	return outputPath, nil
}
