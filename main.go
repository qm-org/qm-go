package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
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

	outputFPSString := strconv.Itoa(*outputFPS)
	if *outputFPS == -1 {
		outputFPS = 24 - (3 * preset)
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

	outputHeight := int(float64(inputHeight) * outputScale)
	outputWidth := int(float64(inputWidth) * outputScale)
	bitrate := outputHeight / 2 * outputWidth * outputFPS / preset * 1000

	cmd := runFFmpeg(*input, *output, outputFPS, bitrate, outputHeight, outputWidth)

	fmt.Println(*input, *outputFPS, *output, bitrate, "and we have", strconv.Itoa(outputHeight), "x", strconv.Itoa(outputWidth), "and it is preset", *preset, "or", preset)

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

func runFFmpeg(input string, output string, framerate int, bitrate int, height int, width int) *exec.Cmd {
	framerateString := strconv.Itoa(framerate)
	bitrateString := strconv.Itoa(bitrate)
	heightString := strconv.Itoa(height)
	widthString := strconv.Itoa(width)
	return exec.Command(
		"ffmpeg",
		"-i", input,
		"-r", framerateString,
		"-b:v", bitrateString,
		"-vf", "scale="+widthString+":"+heightString,
		"-c:v", "libx264",
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
