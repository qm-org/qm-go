package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"math"
	"os/exec"
	"strconv"
	"strings"
)

func main() {
	fmt.Println("Quality Munchera")
	input := flag.String("i", "", "Input file")
	output := flag.String("o", "", "Output file")
	outputFPS := flag.Int("fps", -1, "Output framerate")
	outputScaleString := flag.String("s", "", "Amount to upscale by")
	preset := flag.Int("p", 1, "Preset, 0-7, higher being worse")

	flag.Parse()

	presetString := strconv.Itoa(*preset)

	if *outputFPS == -1 {
		*outputFPS = 24 - (3 * *preset)
	}

	var outputScale float64
	if *outputScaleString == "" {
		outputScale = fracToFloat("1/" + presetString)
	} else {
		outputScale = fracToFloat(*outputScaleString)
	}

	if *output == "" {
		log.Println("Output file not specified")
	}

	if *input != "" {
		fmt.Println("input found " + *input)
	} else {
		fmt.Println("No input lmao")
	}
	if *output != "" {
		fmt.Println("output found " + *output)
	} else {
		fmt.Println("No output lmao")
	}

	fmt.Println()

	inputDuration := getDuration(*input)
	fmt.Println("duration is", inputDuration)

	inputFPS := getFramerate(*input)
	fmt.Println("fps is", inputFPS)

	inputHeight := getDimension("height", *input)
	fmt.Println("height is", inputHeight)

	inputWidth := getDimension("width", *input)
	fmt.Println("width is", inputWidth)

	outputHeight := int(math.Round(float64(inputHeight)*outputScale)/2) * 2
	outputWidth := int(math.Round(float64(inputWidth)*outputScale)/2) * 2
	bitrate := 2 * (outputHeight / 2 * outputWidth * int(math.Sqrt(float64(*outputFPS))) / *preset)
	audioBitrate := 80000 / *preset
	fmt.Println("bitrate is", bitrate, "which i got by doing", outputHeight, "/ 2 *", outputWidth, "*", *outputFPS, "/", *preset)

	cmd := runFFmpeg(*input, *output, *outputFPS, bitrate, audioBitrate, outputWidth, outputHeight)

	fmt.Println(*input, *output, *outputFPS, bitrate, audioBitrate, outputWidth, outputHeight)

	cmd.Run()
}

func getDuration(in string) float64 {
	var out bytes.Buffer
	var stderr bytes.Buffer

	cmdArgs := []string{
		"-i", in,
		"-show_entries", "format=duration",
		"-of", "csv=p=0",
	}

	cmdt := exec.Command("ffprobe", cmdArgs...)
	cmdt.Stdout = &out
	cmdt.Stderr = &stderr

	if err := cmdt.Run(); err != nil {
		log.Println(err)
	}

	stria := out.String()
	stria = strings.TrimSuffix(stria, "\r\n")
	striafloat, erro := strconv.ParseFloat(stria, 64)

	if erro != nil {
		log.Println(erro)
	}

	return striafloat
}

func getFramerate(in string) float64 {
	var out bytes.Buffer
	var stderr bytes.Buffer

	cmdArgs := []string{
		"-i", in,
		"-show_entries", "stream=r_frame_rate",
		"-select_streams", "v:0",
		"-of", "csv=p=0",
	}

	cmdt := exec.Command("ffprobe", cmdArgs...)
	cmdt.Stdout = &out
	cmdt.Stderr = &stderr

	if err := cmdt.Run(); err != nil {
		log.Println(err)
	}

	stria := out.String()
	stria = strings.TrimSuffix(stria, "\r\n")
	numden := strings.Split(stria, "/")
	numer, numerror := strconv.Atoi(numden[0])
	denom, denerror := strconv.Atoi(numden[1])
	striafloat := float64(numer) / float64(denom)

	if numerror != nil {
		log.Println(numerror)
	}
	if denerror != nil {
		log.Println(denerror)
	}

	return striafloat
}

func getDimension(axis string, in string) int {
	var out bytes.Buffer
	var stderr bytes.Buffer

	cmdArgs := []string{
		"-i", in,
		"-select_streams", "v:0",
		"-show_entries", "stream=" + axis,
		"-of", "csv=p=0",
	}

	cmdt := exec.Command("ffprobe", cmdArgs...)
	cmdt.Stdout = &out
	cmdt.Stderr = &stderr

	if err := cmdt.Run(); err != nil {
		log.Println(err)
	}

	stria := out.String()
	stria = strings.TrimSuffix(stria, "\r\n")
	strint, strerror := strconv.Atoi(stria)

	if strerror != nil {
		log.Println(strerror)
	}

	return strint
}

func runFFmpeg(input string, output string, framerate int, bitrate int, audioBitrate int, width int, height int) *exec.Cmd {
	framerateString := strconv.Itoa(framerate)
	bitrateString := strconv.Itoa(bitrate)
	audioBitrateString := strconv.Itoa(audioBitrate)
	widthString := strconv.Itoa(width)
	heightString := strconv.Itoa(height)
	return exec.Command(
		"ffmpeg",
		"-i", input,
		"-preset", "ultrafast",
		"-c:v", "libx264",
		"-b:v", bitrateString,
		"-vf", "scale="+widthString+":"+heightString,
		"-r", framerateString,
		"-c:a", "aac",
		"-b:a", audioBitrateString,
		output,
	)
}

func fracToFloat(in string) float64 {
	numden := strings.Split(in, "/")
	numer, numerror := strconv.Atoi(numden[0])
	denom, denerror := strconv.Atoi(numden[1])
	striafloat := float64(numer) / float64(denom)

	if numerror != nil {
		log.Println(numerror)
	}
	if denerror != nil {
		log.Println(denerror)
	}

	return striafloat
}
