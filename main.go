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
	// flags
	input, output           string
	debug                   bool
	noVideo, noAudio        bool
	preset                  int
	start, end, outDuration float64
	volume                  int
	outScale                float64
	videoBrDiv, audioBrDiv  int
	stretch                 string
	outFPS                  int
	speed                   int
	zoom                    float64
	bitrate                 int
	fadein, fadeout         float64
	stutter                 int
	vignette                float64
	corrupt                 int
	interlace               bool
	lagfun                  bool
	resample                bool
	corruptAmount           int
	// other variables
	audioBitrate  int
	corruptFilter string
)

func init() {
	pflag.CommandLine.SortFlags = false
	pflag.StringVarP(&input, "input", "i", "", "Specify the input file")
	pflag.StringVarP(&output, "output", "o", "", "Specify the output file")
	pflag.BoolVarP(&debug, "debug", "d", false, "Print out debug information")
	pflag.BoolVar(&noVideo, "no-video", false, "Produces an output with no video")
	pflag.BoolVar(&noAudio, "no-audio", false, "Produces an output with no audio")
	pflag.IntVarP(&preset, "preset", "p", 4, "Specify the quality preset")
	pflag.Float64Var(&start, "start", 0, "Specify the start time of the output")
	pflag.Float64Var(&end, "end", -1, "Specify the end time of the output, cannot be used when duration is specified")
	pflag.Float64Var(&outDuration, "duration", -1, "Specify the duration of the output, cannot be used when end is specified")
	pflag.IntVarP(&volume, "volume", "v", 0, "Specify the amount to increase or decrease the volume by, in dB")
	pflag.Float64VarP(&outScale, "scale", "s", -1, "Specify the output scale")
	pflag.IntVar(&videoBrDiv, "video-bitrate", -1, "Specify the video bitrate divisor")
	pflag.IntVar(&videoBrDiv, "vb", videoBrDiv, "Shorthand for --video-bitrate")
	pflag.IntVar(&audioBrDiv, "audio-bitrate", -1, "Specify the audio bitrate divisor")
	pflag.IntVar(&audioBrDiv, "ab", audioBrDiv, "Shorthand for --audio-bitrate")
	pflag.StringVar(&stretch, "stretch", "1:1", "Modify the existing aspect ratio")
	pflag.IntVar(&outFPS, "fps", -1, "Specify the output fps")
	pflag.IntVar(&speed, "speed", 1, "Specify the video and audio speed")
	pflag.Float64VarP(&zoom, "zoom", "z", 1, "Specify the amount to zoom in or out")
	pflag.Float64Var(&fadein, "fade-in", 0, "Fade in duration")
	pflag.Float64Var(&fadeout, "fade-out", 0, "Fade out duration")
	pflag.IntVar(&stutter, "stutter", 0, "Randomize the order of a frames")
	pflag.Float64Var(&vignette, "vignette", 0, "Specify the amount of vignette")
	pflag.IntVar(&corrupt, "corrupt", 0, "Corrupt the output")
	pflag.BoolVar(&interlace, "interlace", false, "Interlace the output")
	pflag.BoolVar(&lagfun, "lagfun", false, "Force darker pixels to update slower")
	pflag.BoolVar(&resample, "resample", false, "Blend frames together instead of dropping them")
	pflag.Parse()

	if input == "" {
		log.Fatal("No input was specified")
	}
	if output == "" {
		log.Fatal("No output was specified")
	}
	if start < 0 {
		log.Fatal("Start time cannot be negative")
	}
	if start >= end && end != -1 {
		log.Fatal("Start time cannot be greater than or equal to end time")
	}
	if outDuration != -1 && end != -1 {
		log.Fatal("Cannot specify both duration and end time")
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
		log.Print(
			input,
			output,
			debug,
			noVideo,
			noAudio,
			preset,
			start,
			end,
			outDuration,
			outScale,
			videoBrDiv,
			audioBrDiv,
			stretch,
			outFPS,
			speed,
			zoom,
			bitrate,
			fadein,
			fadeout,
			stutter,
			vignette,
			corrupt,
			interlace,
			lagfun,
			resample,
			volume,
		)
	}

	// get needed information from input video
	inputDuration, err := getDuration(input)
	if err != nil {
		log.Fatal(err)
	}
	if debug {
		log.Print("duration is ", inputDuration)
	}

	inputFPS, err := getFramerate(input)
	if err != nil {
		log.Fatal(err)
	}
	if debug {
		log.Print("fps is ", inputFPS)
	}

	inputWidth, inputHeight, err := getResolution(input)
	if err != nil {
		log.Fatal(err)
	}

	if debug {
		log.Print("resolution is ", inputWidth, " by ", inputHeight)
	}

	// set up the args for the output

	// fps and tmix
	if outFPS == -1 {
		outFPS = 24 - (3 * preset)
	}
	var fpsFilter string = "fps=" + strconv.Itoa(outFPS)
	var tmixFrames int = 0
	if resample {
		if outFPS <= int(inputFPS) {
			tmixFrames = int(inputFPS) / outFPS
			fpsFilter = "tmix=frames=" + strconv.Itoa(tmixFrames) + ":weights=1,fps=" + strconv.Itoa(outFPS)
			if debug {
				log.Print("resampling with tmix, tmix frames ", tmixFrames, " and output fps is "+strconv.Itoa(outFPS))
			}
		} else {
			log.Fatal("Cannot resample from a lower framerate to a higher framerate (output fps exceeds input fps)")
		}
	}

	if debug {
		log.Print("Output FPS is ", outFPS)
	}

	if outScale == -1 {
		outScale = 1.0 / float64(preset)
	}
	if debug {
		log.Print("Output scale is ", outScale)
	}

	// stretch calculations
	aspect := strings.Split(stretch, ":")
	aspectWidth, err := strconv.Atoi(aspect[0])
	if err != nil {
		log.Print(err)
	}
	aspectHeight, err := strconv.Atoi(aspect[1])
	if err != nil {
		log.Print(err)
	}

	if debug {
		log.Print("aspect ratio is ", aspectWidth, " by ", aspectHeight)
	}

	// calculate the output resolution and bitrate based on that
	outputWidth := int(math.Round(float64(inputWidth)*outScale*float64(aspectWidth))/2) * 2
	outputHeight := int(math.Round(float64(inputHeight)*outScale*float64(aspectHeight))/2) * 2
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
		log.Print("bitrate is ", bitrate, " which i got by doing ", outputHeight, "*", outputWidth, "*", int(math.Sqrt(float64(outFPS))), "/", preset)
	}

	// set up the ffmpeg filter for -filter_complex
	var filter strings.Builder

	if !(noVideo) {
		filter.WriteString(fpsFilter + ",scale=" + strconv.Itoa(outputWidth) + ":" + strconv.Itoa(outputHeight) + ",setsar=1:1")

		if fadein != 0 {
			filter.WriteString(",fade=t=in:d=" + strconv.FormatFloat(fadein, 'f', -1, 64))
			if debug {
				log.Print("fade in is ", fadein)
			}
		}

		if fadeout != 0 {
			filter.WriteString(",fade=t=out:d=" + strconv.FormatFloat(fadeout, 'f', -1, 64) + ":st=" + strconv.FormatFloat((inputDuration-fadeout), 'f', -1, 64))
			if debug {
				log.Print("fade out duration is ", fadeout, " start time is ", (inputDuration - fadeout))
			}
		}

		if zoom != 1 {
			filter.WriteString(",zoompan=d=1:zoom=" + strconv.FormatFloat(zoom, 'f', -1, 64) + ":fps=" + strconv.Itoa(outFPS) + ":x='iw/2-(iw/zoom/2)':y='ih/2-(ih/zoom/2)'")
			if debug {
				log.Print("zoom amount is ", zoom)
			}
		}

		if vignette != 0 {
			filter.WriteString(",vignette=PI/(5/(" + strconv.FormatFloat(vignette, 'f', -1, 64) + "/2))")
			if debug {
				log.Print("vignette amount is ", vignette, " or PI/(5/("+strconv.FormatFloat(vignette, 'f', -1, 64)+"/2))")
			}
		}

		if interlace {
			filter.WriteString(",interlace")
		}

		if speed != 1 {
			filter.WriteString(",setpts=(1/" + strconv.Itoa(speed) + ")*PTS")
			if debug {
				log.Print("speed is ", speed)
			}
		}

		if lagfun {
			filter.WriteString(",lagfun")
		}

		if stutter != 0 {
			filter.WriteString(",random=frames=" + strconv.Itoa(stutter))
			if debug {
				log.Print("stutter is ", stutter)
			}
		}
	} else {
		log.Print("no video, ignoring all video filters")
	}

	if !(noAudio) {
		if volume != 0 {
			filter.WriteString(";volume=" + strconv.Itoa(volume))
			if debug {
				log.Print("volume is ", volume)
			}
		}

		if speed != 1 {
			filter.WriteString(";atempo=" + strconv.Itoa(speed))
			if debug {
				log.Print("audio speed is ", speed)
			}
		}
	} else {
		log.Print("no audio, ignoring all audio filters")
	}

	// corruption calculations based on width and height
	if corrupt != 0 {
		corruptAmount = int(float64(outputHeight*outputWidth) / float64(bitrate) * 100000.0 / float64(corrupt*3))
		corruptFilter = "noise=" + strconv.Itoa(corruptAmount)
		if debug {
			log.Print("corrupt amount is", corruptAmount)
			log.Print("(", outputHeight, " * ", outputWidth, ")", " / 2073600 * 1000000", " / ", "(", corrupt, "* 10)")
			log.Print("corrupt filter is -bsf ", corruptFilter)
		}
	}

	// ffmpeg args
	args := []string{
		"-y",
	}
	if start != 0 {
		args = append(args, "-ss", strconv.FormatFloat(start, 'f', -1, 64))
	}
	if end != -1 {
		outDuration = end - start
	}
	if outDuration != -1 {
		args = append(args, "-t", strconv.FormatFloat(outDuration, 'f', -1, 64))
	}
	if noVideo {
		args = append(args, "-vn")
		if debug {
			log.Print("no video")
		}
	}
	if noAudio {
		args = append(args, "-an")
		if debug {
			log.Print("no audio")
		}
	}
	args = append(args,
		"-i", input,
		"-preset", "ultrafast",
		"-c:v", "libx264",
		"-b:v", strconv.Itoa(int(bitrate)),
		"-c:a", "aac",
		"-b:a", strconv.Itoa(int(audioBitrate)),
	)
	if len(filter.String()) != 0 {
		args = append(args, "-filter_complex", filter.String())
	}
	if corrupt != 0 {
		args = append(args, "-bsf", corruptFilter)
	}
	args = append(args, output)

	if debug {
		log.Print(args)
	}

	// encode
	cmd := exec.Command("ffmpeg", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if debug {
			log.Print(string(out))
		}
		log.Fatal(err)
	}
}

// all following functions move to other file when it decides to actually work
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
