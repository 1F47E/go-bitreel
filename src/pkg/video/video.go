package video

import (
	cfg "bytereel/pkg/config"
	"bytereel/pkg/logger"
	"fmt"
	"os/exec"
	"strings"
)

var log = logger.Log

// call ffmpeg to decode the video into frames
func ExtractFrames(filename, dir string) error {
	framesPath := dir + "/out_%08d.png"
	cmdStr := fmt.Sprintf("ffmpeg -y -i %s %s", filename, framesPath)
	cmdList := strings.Split(cmdStr, " ")
	log.Debugf("Running ffmpeg command: %s\n", cmdStr)
	cmd := exec.Command(cmdList[0], cmdList[1:]...)
	return cmd.Run()
}

// call ffmpeg to encode frames into video
func EncodeFrames() error {
	cmdStr := "ffmpeg -y -framerate 30 -i tmp/out/out_%08d.png -c:v prores -profile:v 3 -pix_fmt yuv422p10 " + cfg.PathVideoOut
	cmdList := strings.Split(cmdStr, " ")
	log.Debugf("Running ffmpeg command: %s\n", cmdStr)
	cmd := exec.Command(cmdList[0], cmdList[1:]...)
	return cmd.Run()
}
