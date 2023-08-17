package video

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	cfg "github.com/1F47E/go-bitreel/internal/config"
	"github.com/1F47E/go-bitreel/internal/logger"
)

// call ffmpeg to decode the video into frames
func ExtractFrames(ctx context.Context, filename, dir string) error {
	framesPath := dir + "/out_%08d.png"
	cmdStr := fmt.Sprintf("ffmpeg -y -i %s %s", filename, framesPath)
	cmdList := strings.Split(cmdStr, " ")
	logger.Log.Debugf("Running ffmpeg command: %s\n", cmdStr)
	cmd := exec.CommandContext(ctx, cmdList[0], cmdList[1:]...)
	return cmd.Run()
}

// call ffmpeg to encode frames into video
func EncodeFrames(ctx context.Context) error {
	cmdStr := "ffmpeg -y -framerate 30 -i tmp/out/out_%08d.png -c:v prores -profile:v 3 -pix_fmt yuv422p10 " + cfg.PathVideoOut
	cmdList := strings.Split(cmdStr, " ")
	logger.Log.Debugf("Running ffmpeg command: %s\n", cmdStr)
	cmd := exec.CommandContext(ctx, cmdList[0], cmdList[1:]...)
	return cmd.Run()
}
