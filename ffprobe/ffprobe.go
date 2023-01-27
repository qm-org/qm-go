package ffprobe

import (
	"log"
	"os/exec"
	"strconv"
	"strings"
)

func FrameCount(input string) int {
	args := []string{
		"-i", input,
		"-show_entries", "stream=nb_read_packets",
		"-select_streams", "v:0",
		"-count_packets",
		"-of", "csv=p=0",
	}

	cmd := exec.Command("ffprobe", args...)

	out, err := cmd.Output()
	if err != nil {
		return 0
	}

	outs := string(out)
	outs = strings.TrimSuffix(outs, "\n") // removing the newline at the end of the output
	outs = strings.TrimSuffix(outs, "\r") // windows includes a carriage return, so we remove that too
	outs = strings.TrimSuffix(outs, "\n") // just in case there's a newline after the carriage return, because why not

	outi, err := strconv.Atoi(outs)
	if err != nil {
		return 0
	}

	return outi
}

type MediaData struct {
	Framerate float64
	Height    int
	Width     int
	Duration  float64
}

func ProbeData(input string) (MediaData, error) {
	args := []string{
		"-i", input,
		"-show_entries", "stream=width,height,r_frame_rate,duration",
		"-select_streams", "v:0",
		"-of", "csv=p=0",
	}

	cmd := exec.Command("ffprobe", args...)

	out, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}

	outs := string(out)
	if outs == "" {
		args = []string{
			"-i", input,
			"-show_entries", "stream=duration",
			"-select_streams", "a:0",
			"-of", "csv=p=0",
		}

		cmd := exec.Command("ffprobe", args...)

		out, err := cmd.Output()
		if err != nil {
			log.Fatal(err)
		}

		outs := string(out)

		outs = strings.TrimSuffix(outs, "\n") // removing the newline at the end of the output
		outs = strings.TrimSuffix(outs, "\r") // windows includes a carriage return, so we remove that too
		outs = strings.TrimSuffix(outs, "\n") // just in case there's a newline after the carriage return, because why not

		allargs := strings.Split(outs, ",")

		duration, _ := strconv.ParseFloat(allargs[0], 64)

		var outInfo MediaData

		outInfo.Duration = duration

		return outInfo, nil
	}
	outs = strings.TrimSuffix(outs, "\n") // removing the newline at the end of the output
	outs = strings.TrimSuffix(outs, "\r") // windows includes a carriage return, so we remove that too
	outs = strings.TrimSuffix(outs, "\n") // just in case there's a newline after the carriage return, because why not

	allargs := strings.Split(outs, ",")

	width, _ := strconv.Atoi(allargs[0])
	height, _ := strconv.Atoi(allargs[1])

	framerateFrac := strings.Split(allargs[2], "/")

	numerator, err := strconv.Atoi(framerateFrac[0])
	if err != nil {
		log.Fatal(err)
	}
	denominator, err := strconv.Atoi(framerateFrac[1])
	if err != nil {
		log.Fatal(err)
	}

	framerate := float64(numerator) / float64(denominator)

	duration, _ := strconv.ParseFloat(allargs[3], 64)

	var outInfo MediaData

	outInfo.Framerate = framerate
	outInfo.Height = height
	outInfo.Width = width
	outInfo.Duration = duration

	return outInfo, nil
}
