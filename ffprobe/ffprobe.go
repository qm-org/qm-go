package ffprobe

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

func Duration(input string) (float64, error) {
	args := []string{
		"-i", input,
		"-show_entries", "format=duration",
		"-of", "csv=p=0",
	}

	cmd := exec.Command("ffprobe", args...)

	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	outs := string(out)
	outs = strings.TrimSuffix(outs, "\n") // removing the newline at the end of the output
	outs = strings.TrimSuffix(outs, "\r") // windows includes a carriage return, so we remove that too
	outs = strings.TrimSuffix(outs, "\n") // just in case there's a newline after the carriage return, because why not

	outf, err := strconv.ParseFloat(outs, 64)
	if err != nil {
		return 0, nil
	}

	return outf, nil
}

func Resolution(input string) (int, int, error) {
	args := []string{
		"-i", input,
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=p=0",
	}

	cmd := exec.Command("ffprobe", args...)

	out, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}

	outs := string(out)

	outs = strings.TrimSuffix(outs, "\n") // removing the newline at the end of the output
	outs = strings.TrimSuffix(outs, "\r") // windows includes a carriage return, so we remove that too
	outs = strings.TrimSuffix(outs, "\n") // just in case there's a newline after the carriage return, because why not
	outl := strings.Split(outs, ",")

	if len(outl) != 2 {
		return 0, 0, errors.New("parsed list is not of length 2")
	}

	width, err := strconv.Atoi(outl[0])
	if err != nil {
		return 0, 0, err
	}
	height, err := strconv.Atoi(outl[1])
	if err != nil {
		return 0, 0, err
	}

	return width, height, nil
}

func Framerate(input string) (float64, error) {
	args := []string{
		"-i", input,
		"-show_entries", "stream=r_frame_rate",
		"-select_streams", "v:0",
		"-of", "csv=p=0",
	}

	cmd := exec.Command("ffprobe", args...)

	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	outs := string(out)
	outs = strings.TrimSuffix(outs, "\n") // removing the newline at the end of the output
	outs = strings.TrimSuffix(outs, "\r") // windows includes a carriage return, so we remove that too
	outs = strings.TrimSuffix(outs, "\n") // just in case there's a newline after the carriage return, because why not
	outl := strings.Split(outs, "/")

	if len(outl) != 2 {
		return 0, errors.New("parsed list is not of length 2")
	}

	numerator, err := strconv.Atoi(outl[0])
	if err != nil {
		return 0, err
	}
	denominator, err := strconv.Atoi(outl[1])
	if err != nil {
		return 0, err
	}

	return float64(numerator) / float64(denominator), nil
}

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

	return outi
}
