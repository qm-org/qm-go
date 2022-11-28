package main

import (
	"errors"
	"log"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

var (
	debug         bool
	interlace     bool
	input, output string
	outFPS        int
	outScale      float64
	videoBrDiv    int
	audioBrDiv    int
	preset        int
	filter        string
	bitrate       int
	audioBitrate  int
)

func init() {
	pflag.CommandLine.SortFlags = false
	pflag.StringVarP(&input, "input", "i", "", "Specify the input file")
	pflag.StringVarP(&output, "output", "o", "", "Specify the output file")
	pflag.IntVarP(&preset, "preset", "p", 4, "Specify the preset used")
	pflag.IntVarP(&outFPS, "fps", "f", -1, "Specify the output fps")
	pflag.Float64VarP(&outScale, "scale", "s", -1, "Specify the output scale")
	pflag.IntVar(&videoBrDiv, "video bitrate", -1, "Specify the video bitrate divisor")
	pflag.IntVar(&videoBrDiv, "vb", -1, "Specify the video bitrate divisor")
	pflag.IntVar(&audioBrDiv, "audio bitrate", -1, "Specify the audio bitrate divisor")
	pflag.IntVar(&audioBrDiv, "ab", -1, "Specify the audio bitrate divisor")
	pflag.BoolVar(&debug, "debug", false, "Print out debug information")
	pflag.BoolVar(&interlace, "interlace", false, "Interlace the output")
	pflag.Parse()

	if input == "" {
		log.Fatal("No input was specified")
	}
	if output == "" {
		log.Fatal("No output was specified")
	}

	_, err := os.Stat(input)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatal("Input file " + input + " does not exist")
		} else {
			log.Fatal(err)
		}
	}

}

func main() {
	if debug {
		log.Print("throwing all flags out")
		log.Print(input, output, preset, outFPS, outScale, debug)
	}

	if outFPS == -1 {
		outFPS = 24 - (3 * preset)
	}
	if debug {
		log.Print("Output FPS is", outFPS)
	}

	if outScale == -1 {
		outScale = 1.0 / float64(preset)
	}
	if debug {
		log.Print("Output scale is", outScale)
	}

	inputDuration, err := getDuration(input)
	if err != nil {
		log.Fatal(err)
	}
	if debug {
		log.Print("duration is", inputDuration)
	}

	inputFPS, err := getFramerate(input)
	if err != nil {
		log.Fatal(err)
	}
	if debug {
		log.Print("fps is", inputFPS)
	}

	inputWidth, inputHeight, err := getResolution(input)
	if err != nil {
		log.Fatal(err)
	}
	if debug {
		log.Print("resolution is", inputWidth, "by", inputHeight)
	}

	outputHeight := int(math.Round(float64(inputHeight)*outScale)/2) * 2
	outputWidth := int(math.Round(float64(inputWidth)*outScale)/2) * 2
	if videoBrDiv != -1 {
		bitrate = outputHeight * outputWidth * int(math.Sqrt(float64(outFPS))) / videoBrDiv
	} else {
		bitrate = outputHeight * outputWidth * int(math.Sqrt(float64(outFPS))) / preset
	}

	if audioBrDiv != -1 {
		audioBitrate = 80000 / audioBrDiv
	} else {
		audioBitrate = 80000 / preset
	}

	if debug {
		log.Print("bitrate is", bitrate, "which i got by doing", outputHeight, "*", outputWidth, "*", int(math.Sqrt(float64(outFPS))), "/", preset)
	}

	if interlace {
		filter = ",interlace"
	}

	args := []string{
		"-y",
		"-i", input,
		"-preset", "ultrafast",
		"-r", strconv.Itoa(int(outFPS)),
		"-c:v", "libx264",
		"-b:v", strconv.Itoa(int(bitrate)),
		"-vf", "scale=" + strconv.Itoa(outputWidth) + ":" + strconv.Itoa(outputHeight) + filter,
		"-c:a", "aac",
		"-b:a", strconv.Itoa(int(audioBitrate)),
		output,
	}

	cmd := exec.Command("ffmpeg", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if debug {
			log.Print(string(out))
		}
		log.Fatal(err)
	}
}

// move to other file when it decides to actually work
func getDuration(input string) (float64, error) {
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
	outs = strings.TrimSuffix(outs, "\n")
	outs = strings.TrimSuffix(outs, "\r")
	outs = strings.TrimSuffix(outs, "\n")

	outf, err := strconv.ParseFloat(outs, 64)
	if err != nil {
		return 0, nil
	}

	return outf, nil
}

func getFramerate(input string) (float64, error) {
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
	outs = strings.TrimSuffix(outs, "\n")
	outs = strings.TrimSuffix(outs, "\r")
	outs = strings.TrimSuffix(outs, "\n")
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

func getResolution(input string) (int, int, error) {
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

	outs = strings.TrimSuffix(outs, "\n")
	outs = strings.TrimSuffix(outs, "\r")
	outs = strings.TrimSuffix(outs, "\n")
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
