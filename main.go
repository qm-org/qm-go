/*
Quality Muncher Go (QM:GO) - a program to make videos lower in quality
Copyright (C) 2022 Quality Muncher Organization

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package main

import (
	"errors"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/flopp/go-findfont"
	"github.com/spf13/pflag"
)

var (
	// flags
	input, output             string
	debug                     bool
	loglevel                  string
	noVideo, noAudio          bool
	replaceAudio              string
	preset                    int
	start, end, outDuration   float64
	volume                    int
	outScale                  float64
	videoBrDiv, audioBrDiv    int
	stretch                   string
	outFPS                    int
	speed                     float64
	zoom                      float64
	bitrate                   int
	fadein, fadeout           float64
	stutter                   int
	vignette                  float64
	corrupt                   int
	interlace                 bool
	lagfun                    bool
	resample                  bool
	corruptAmount             int
	text, textFont, textColor string
	textposx, textposy        int
	fontSize                  float64

	// other variables
	audioBitrate  int
	corruptFilter string
)

func init() {
	pflag.CommandLine.SortFlags = false
	pflag.StringVarP(&input, "input", "i", "", "Specify the input file")
	pflag.StringVarP(&output, "output", "o", "", "Specify the output file")
	pflag.BoolVarP(&debug, "debug", "d", false, "Print out debug information")
	pflag.StringVar(&loglevel, "loglevel", "error", "Specify the log level for ffmpeg")
	pflag.BoolVar(&noVideo, "no-video", false, "Produces an output with no video")
	pflag.BoolVar(&noAudio, "no-audio", false, "Produces an output with no audio")
	pflag.StringVar(&replaceAudio, "replace-audio", "", "Replace the audio with the specified file")
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
	pflag.Float64Var(&speed, "speed", 1.0, "Specify the video and audio speed")
	pflag.Float64VarP(&zoom, "zoom", "z", 1, "Specify the amount to zoom in or out")
	pflag.Float64Var(&fadein, "fade-in", 0, "Fade in duration")
	pflag.Float64Var(&fadeout, "fade-out", 0, "Fade out duration")
	pflag.IntVar(&stutter, "stutter", 0, "Randomize the order of a frames")
	pflag.Float64Var(&vignette, "vignette", 0, "Specify the amount of vignette")
	pflag.IntVar(&corrupt, "corrupt", 0, "Corrupt the output")
	pflag.BoolVar(&interlace, "interlace", false, "Interlace the output")
	pflag.BoolVar(&lagfun, "lagfun", false, "Force darker pixels to update slower")
	pflag.BoolVar(&resample, "resample", false, "Blend frames together instead of dropping them")
	pflag.StringVarP(&text, "text", "t", "", "Text to add (if empty, no text)")
	pflag.StringVar(&textFont, "text-font", "arial", "Text to add (if empty, no text)")
	pflag.StringVar(&textColor, "text-color", "white", "Text color")
	pflag.IntVar(&textposx, "text-pos-x", 50, "horizontal position of text, 0 is far left, 100 is far right")
	pflag.IntVar(&textposy, "text-pos-y", 90, "vertical position of text, 0 is top, 100 is bottom")
	pflag.Float64Var(&fontSize, "font-size", 12, "Font size (scales with output width")
	pflag.Parse()

	if input == "" {
		log.Fatal("No input was specified")
	}
	if output == "" {
		if debug {
			log.Println("No output was specified, using input name plus (Quality Munched)")
		}
		output = strings.TrimSuffix(input, filepath.Ext(input)) + " (Quality Munched)" + ".mp4"
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
			text,
			textFont,
			textColor,
			textposx,
			textposy,
			fontSize,
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

	// if NOT using --no-video, set add the specified video filters to filter
	if !(noVideo) {

		if speed != 1 {
			filter.WriteString("setpts=(1/" + strconv.FormatFloat(speed, 'f', -1, 64) + ")*PTS,")
			if debug {
				log.Print("speed is ", speed)
			}
		}

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

		if text != "" {
			fontPath, err := findfont.Find(textFont + ".ttf")
			if err != nil {
				panic(err)
			}
			if err := os.MkdirAll("temp", os.ModePerm); err != nil {
				log.Fatal(err)
			}
			input, err := ioutil.ReadFile(fontPath)
			if err != nil {
				log.Print(err)
			}
			err = ioutil.WriteFile("temp/font.ttf", input, 0644)
			if err != nil {
				log.Print("Error creating", "temp/font.ttf")
				log.Print(err)
				return
			}
			log.Print(fontPath)
			filter.WriteString(",drawtext=fontfile='temp/font.ttf':text='" + text + "':fontcolor=" + textColor + ":borderw=(" + strconv.FormatFloat(fontSize*float64(outputWidth/100), 'f', -1, 64) + "/12):fontsize=" + strconv.FormatFloat(fontSize*float64(outputWidth/100), 'f', -1, 64) + ":x=(w-(tw))*(" + strconv.Itoa(textposx) + "/100):y=(h-(th))*(" + strconv.Itoa(textposy) + "/100)")
			if debug {
				log.Print("text is ", text)
				log.Print(",drawtext=fontfile='temp/font.ttf':text='" + text + "':fontcolor=" + textColor + ":borderw=(" + string(outputWidth/len(text)*2) + "/12):fontsize=" + strconv.Itoa(outputWidth/len(text)*2) + ":x=(w-(tw))*(" + strconv.Itoa(textposx) + "/100):y=(h-(th))*(" + strconv.Itoa(textposy) + "/100)")
			}
		}

		if interlace {
			filter.WriteString(",interlace")
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

	// if NOT using --no-audio, set add the specified audio filters to filter
	if !(noAudio) {
		if volume != 0 {
			filter.WriteString(";volume=" + strconv.Itoa(volume))
			if debug {
				log.Print("volume is ", volume)
			}
		}

		if speed != 1 {
			filter.WriteString(";atempo=" + strconv.FormatFloat(speed, 'f', -1, 64))
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
		"-y", // forces overwrite of existing file, if one does exist
	}
	if start != 0 {
		args = append(args, "-ss", strconv.FormatFloat(start, 'f', -1, 64)) // -ss is the start time
	}
	if end != -1 {
		outDuration = end - start
	}
	if outDuration != -1 {
		args = append(args, "-t", strconv.FormatFloat(outDuration, 'f', -1, 64)) // -t sets the duration
	}
	if noVideo {
		args = append(args, "-vn") // removes video
		if debug {
			log.Print("no video")
		}
	}
	if noAudio {
		args = append(args, "-an") // removes audio
		if debug {
			log.Print("no audio")
		}
	}
	args = append(args,
		// "-stats_period", "0.1",
		"-loglevel", loglevel,
		"-i", input,
	)
	if replaceAudio != "" {
		args = append(args, "-i", replaceAudio)
		args = append(args, "-map", "0:v:0")
		args = append(args, "-map", "1:a:0")
	}
	args = append(args,
		"-preset", "ultrafast",
		"-shortest",
		"-c:v", "libx264",
		"-b:v", strconv.Itoa(int(bitrate)),
		"-c:a", "aac",
		"-b:a", strconv.Itoa(int(audioBitrate)),
	)
	if len(filter.String()) != 0 { // if filter is not empty, add the flag for setting the complex filter to the ffmpeg args
		args = append(args, "-filter_complex", filter.String())
	}
	if corrupt != 0 { // if corrupt isn't default, add the corrupt filter to the ffmpeg args
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

// all following functions should be moved to another file, at least when i figure out how to make it work
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
	outs = strings.TrimSuffix(outs, "\n") // removing the newline at the end of the output
	outs = strings.TrimSuffix(outs, "\r") // windows includes a carriage return, so we remove that too
	outs = strings.TrimSuffix(outs, "\n") // just in case there's a newline after the carriage return, because why not

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
