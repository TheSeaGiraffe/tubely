package main

import (
	"fmt"
	"os/exec"
)

func processVideoForFastStart(filePath string) (string, error) {
	outFilePath := filePath + ".processing"
	args := []string{
		"-i", filePath,
		"-c", "copy",
		"-movflags", "faststart",
		"-f", "mp4",
		outFilePath,
	}
	cmd := exec.Command("ffmpeg", args...)
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("Error running command: %w", err)
	}

	return outFilePath, nil
}
