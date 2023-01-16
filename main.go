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
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"qm-go/ffprobe"
	"qm-go/utils"

	"github.com/flopp/go-findfont"
	"github.com/spf13/pflag"
)

var (
	// flags
	output                    string
	inputs                    []string
	debug                     bool
	progbarLength             int
	loopNum                   int
	loglevel                  string
	updateSpeed               float64
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
	formattingCodes           bool

	// other variables
	audioBitrate           int
	corruptFilter          string
	unspecifiedProgbarSize bool
	bitrate                int
)

func init() {
	pflag.CommandLine.SortFlags = false
	pflag.StringSliceVarP(&inputs, "input", "i", []string{""}, "Specify the input file(s)")
	pflag.StringVarP(&output, "output", "o", "", "Specify the output file")
	pflag.BoolVarP(&debug, "debug", "d", false, "Print out debug information")
	pflag.IntVar(&progbarLength, "progress-bar", -1, "Length of progress bar, defaults based on terminal width")
	pflag.IntVar(&loopNum, "loop", 1, "Number of time to compress the input. ONLY USED FOR IMAGES.")
	pflag.StringVar(&loglevel, "loglevel", "error", "Specify the log level for ffmpeg")
	pflag.Float64Var(&updateSpeed, "update-speed", 0.0167, "Specify the speed at which stats will be updated")
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
	pflag.BoolVar(&formattingCodes, "formatting-codes", false, "Print out ANSI escape/formatting codes")
	pflag.Parse()

	// check for invalid arguments
	if inputs[0] == "" {
		log.Fatal("No input was specified")
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

	// set some bools for making sure certain variables can be overwritten later
	if progbarLength == -1 {
		unspecifiedProgbarSize = true
	} else {
		unspecifiedProgbarSize = false
	}
}

func main() {
	// throw out all flags if debug is enabled
	if debug {
		log.Print("throwing all flags out")
		log.Println(
			inputs,
			output,
			debug,
			progbarLength,
			loopNum,
			loglevel,
			updateSpeed,
			noVideo,
			noAudio,
			preset,
			start,
			end,
			outDuration,
			volume,
			outScale,
			videoBrDiv,
			audioBrDiv,
			stretch,
			outFPS,
			speed,
			zoom,
			fadein,
			fadeout,
			stutter,
			vignette,
			corrupt,
			interlace,
			lagfun,
			resample,
			text,
			textFont,
			textColor,
			textposx,
			textposy,
			fontSize,
		)
	}

	for i, input := range inputs {
		// check if input file exists
		_, err := os.Stat(input)
		if err != nil {
			if os.IsNotExist(err) {
				log.Println("\033[4m\033[31mFatal Error\033[24m: input file " + input + " does not exist\033[0m")
				continue
			} else {
				fmt.Println("\033[4m\033[38;2;254;165;0mWarning\033[24m: Input file " + input + " might not exist\033[0m")
			}
		}

		// set the progressbar length to 0 if it isn't explicitly set, preventing it from spilling over to the next line when called for the first time
		if unspecifiedProgbarSize {
			progbarLength = 0
		}

		if debug {
			log.Println("input: " + input)
			log.Println("input #: " + strconv.Itoa(i))
		}

		inputData, _ := ffprobe.ProbeData(input)

		outExt := ".mp4"

		// check if image or video
		isImage := false
		if inputData.Duration < 0.1 {
			if ffprobe.FrameCount(input) == 1 {
				log.Print("duration: ", inputData.Duration)
				isImage = true
				outExt = ".jpg"
			} else {
				if start >= inputData.Duration {
					log.Fatal("Start time cannot be greater than or equal to input duration")
				}
			}
		} else {
			if start >= inputData.Duration {
				log.Fatal("Start time cannot be greater than or equal to input duration")
			}
		}

		// set the output file name if it isn't explicitly set or if there are multiple inputs
		if len(inputs) > 1 {
			output = strings.TrimSuffix(input, filepath.Ext(input)) + " (Quality Munched)" + outExt
		}
		if output == "" {
			if debug {
				log.Println("No output was specified, using input name plus (Quality Munched)")
			}
			output = strings.TrimSuffix(input, filepath.Ext(input)) + " (Quality Munched)" + outExt
		}

		// check if output file exists
		_, outExistErr := os.Stat(output)
		if outExistErr == nil {
			if debug {
				log.Print("output file already exists")
			}
			var confirm string
			fmt.Println("\033[4m\033[31mWarning\033[24m: The output file\033[91m", output, "\033[31malready exists! Overwrite? [Y/N]\033[0m")
			fmt.Scanln(&confirm)
			if confirm != "Y" && confirm != "y" {
				log.Fatal("Aborted by user - output file already exists")
			}
		}

		startTime := time.Now()
		if isImage {
			if debug {
				log.Println("input is an image")
			}
			imageMunch(input, inputData)
		} else {
			videoMunch(input, inputData)
		}

		// check if output file exists
		_, outErr := os.Stat(output)
		if outErr != nil {
			if os.IsNotExist(outErr) {
				log.Fatal("\033[4m\033[31mFatal Error\033[24m: something went wrong when making the output file!\033[0m")
			} else {
				log.Fatal(err)
			}
		} else {
			fmt.Println("\033[92mFinished encoding\033[32m", output, "\033[92min", utils.TrimTime(utils.FormatTime(time.Since(startTime).Seconds())), "\033[0m")
		}
	}
}

func videoMunch(input string, inputData ffprobe.MediaData) {
	// get input resolution
	if debug {
		log.Print("resolution is ", inputData.Width, " by ", inputData.Height)
	}

	// fps and tmix (frame resampling) filters/calculations
	if outFPS == -1 {
		outFPS = 24 - (3 * preset)
	}
	var fpsFilter string = "fps=" + strconv.Itoa(outFPS)
	var tmixFrames int = 0
	if resample {
		if outFPS <= int(inputData.Framerate) {
			tmixFrames = int(inputData.Framerate) / outFPS
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
	outputWidth := int(math.Round(float64(inputData.Width)*outScale*float64(aspectWidth))/2) * 2
	outputHeight := int(math.Round(float64(inputData.Height)*outScale*float64(aspectHeight))/2) * 2
	if videoBrDiv != -1 {
		bitrate = outputHeight * outputWidth * int(math.Sqrt(float64(outFPS))) / videoBrDiv
	} else {
		bitrate = outputHeight * outputWidth * int(math.Sqrt(float64(outFPS))) / preset
	}

	// calculate the audio bitrate
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
			filter.WriteString(",fade=t=out:d=" + strconv.FormatFloat(fadeout, 'f', -1, 64) + ":st=" + strconv.FormatFloat((inputData.Duration-fadeout), 'f', -1, 64))
			if debug {
				log.Print("fade out duration is ", fadeout, " start time is ", (inputData.Duration - fadeout))
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
				log.Fatal("\033[4m\033[31mFatal Error\033[24m: unable to create temp/font.ttf")
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

	// find what the duration of the output should be for the progress bar, % completion, and ETA in stats
	var realOutputDuration float64
	if outDuration >= inputData.Duration || outDuration == -1 {
		realOutputDuration = (inputData.Duration - start) / speed // if the output duration is longer than the input duration, set the output duration to the input duration times speed
	} else {
		realOutputDuration = outDuration / speed // if the output duration is shorter than the input duration, set the output duration to the output duration times speed
	}

	// if not using --no-audio, set add the specified audio filters to filter
	if !(noAudio) {
		if volume != 0 {
			filter.WriteString(";volume=" + strconv.Itoa(volume))
			if debug {
				log.Print("volume is ", volume)
			}
		}

		// is speed is not 1, set the audio speed to the specified speed
		if speed != 1 {
			if replaceAudio != "" {
				filter.WriteString(";[1]atempo=" + strconv.FormatFloat(speed, 'f', -1, 64)) // if the audio is being replaced, use audio from second input
			} else {
				filter.WriteString(";[0]atempo=" + strconv.FormatFloat(speed, 'f', -1, 64)) // first input (video)
			}

			if debug {
				log.Print("audio speed is ", speed)
			}
		}
	} else {
		log.Print("no audio, ignoring all audio filters")
	}

	// corruption calculations based on width and height
	if corrupt != 0 {
		// amount of corruption is based on the bitrate of the video, the amount of corruption, and the size of the video
		corruptAmount = int(float64(outputHeight*outputWidth) / float64(bitrate) * 100000.0 / float64(corrupt*3))
		corruptFilter = "noise=" + strconv.Itoa(corruptAmount)

		if debug {
			log.Print("corrupt amount is", corruptAmount)
			log.Print("(", outputHeight, " * ", outputWidth, ")", " / 2073600 * 1000000", " / ", "(", corrupt, "* 10)")
			log.Print("corrupt filter is -bsf ", corruptFilter)
		}
	}

	// staring ffmpeg args
	args := []string{
		"-y", // forces overwrite of existing file, if one does exist
		"-loglevel", loglevel,
		"-hide_banner",
		"-progress", "-",
		"-stats_period", strconv.FormatFloat(updateSpeed, 'f', -1, 64),
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

	if noVideo { // remove video if noVideo is true
		args = append(args, "-vn")
		if debug {
			log.Print("no video")
		}
	}

	// remove audio if noAudio is true
	if noAudio {
		args = append(args, "-an") // removes audio
		if debug {
			log.Print("no audio")
		}
	}

	args = append(args,
		"-i", input,
	)

	// if replaceAudio is specified, add the second input to the ffmpeg args to replace the audio of the output
	if replaceAudio != "" {
		args = append(args, "-i", replaceAudio)
		args = append(args, "-map", "0:v:0")
		args = append(args, "-map", "1:a:0")
		if debug {
			log.Print("replacing audio")
		}
	}

	// more always-used args
	args = append(args,
		"-preset", "ultrafast",
		"-shortest",
		"-c:v", "libx264",
		"-b:v", strconv.Itoa(int(bitrate)),
		"-c:a", "aac",
		"-b:a", strconv.Itoa(int(audioBitrate)),
	)

	// if any filters are being used, add them
	if len(filter.String()) != 0 {
		args = append(args, "-filter_complex", filter.String())
	}

	// if corruption is specified, add the corrupt filter to the ffmpeg args
	if corrupt != 0 {
		args = append(args, "-bsf", corruptFilter)
	}

	args = append(args, output) // add the output file to the ffmpeg args

	if debug {
		log.Print(args)
	}

	fmt.Println("\033[94mEncoding file\033[36m", input, "\033[94mto\033[36m", output, "\033[0m") // print the input and output file

	// start ffmpeg for encoding
	cmd := exec.Command("ffmpeg", args...)
	stderr, _ := cmd.StdoutPipe()
	cmd.Start()

	// variables for progress bar and stats
	scannerTextAccum := " "
	eta := 0.0
	currentFrame := 0
	fullTime := ""                // time as a string
	oldFrame := 0                 // the frame number from the last time that the average fps over the last second was calculated
	avgFramerate := " "           // the average framerate over the entire video
	lastSecAvgFramerate := " "    // the average framerate over the last second
	startTime := time.Now()       // the time that the video started encoding
	changeStartTime := time.Now() // the time that the last change in the one second framerate was made
	var currentTotalTime float64

	// if the progress bar length is not set, set it to the length of the longest possible progress bar
	if unspecifiedProgbarSize {
		progbarLength = utils.ProgbarSize(len(" " + strconv.FormatFloat((currentTotalTime*100/realOutputDuration), 'f', 1, 64) + "%" + " time: " + utils.TrimTime(fullTime) + " ETA: " + utils.TrimTime(utils.FormatTime(eta)) + " fps: " + avgFramerate + " fp1s: " + lastSecAvgFramerate))
	}
	if debug {
		log.Print("progbarLength is", progbarLength)
	}

	// print the progress bar, completely unfilled
	fmt.Println(utils.ProgressBar(0.0, realOutputDuration, progbarLength))

	// start the progress bar updater until the video is done encoding
	scanner := bufio.NewScanner(stderr)
	scanner.Split(bufio.ScanRunes)
	for scanner.Scan() {
		// accumulate the text from the scanner
		scannerTextAccum += scanner.Text()

		// if the scanner text is a newline or carriage return, process the accumulated text
		if scanner.Text() == "\r" || scanner.Text() == "\n" {
			// if the accumulated text contains the time, process it and print the progress bar and % completion
			if strings.Contains(scannerTextAccum, "time=") {
				// time variables
				fullTime = strings.Split(strings.Split(scannerTextAccum, "\n")[0], "=")[1]
				hour, _ := strconv.Atoi(strings.Split(fullTime, ":")[0])
				min, _ := strconv.Atoi(strings.Split(fullTime, ":")[1])
				sec, _ := strconv.Atoi(strings.Split(strings.Split(fullTime, ":")[2], ".")[0])
				milisec, _ := strconv.ParseFloat("."+strings.Split(fullTime, ".")[1], 64)
				fullTime = strings.Split(fullTime, ".")[0] + strconv.FormatFloat(milisec, 'f', 1, 64)[1:] + "s"

				// calculate estimated time remaining
				eta = getETA(startTime, currentTotalTime, realOutputDuration)

				// if the progress bar length is not set, set it to the length of the longest possible progress bar
				if unspecifiedProgbarSize {
					progbarLength = utils.ProgbarSize(len(" " + strconv.FormatFloat((currentTotalTime*100/realOutputDuration), 'f', 1, 64) + "%" + " time: " + utils.TrimTime(fullTime) + " ETA: " + utils.TrimTime(utils.FormatTime(eta)) + " fps: " + avgFramerate + " fp1s: " + lastSecAvgFramerate))
				}
				currentTotalTime = float64(hour*3600+min*60+sec) + milisec

				// if the progress bar length is greater than 0, print the progress bar
				if progbarLength > 0 {
					fmt.Print("\033[1A", utils.ProgressBar(currentTotalTime, realOutputDuration, progbarLength))
				} else {
					fmt.Print("\033[1A\033[0J")
				}

				// print the percentage complete
				fmt.Print(" ", strconv.FormatFloat((currentTotalTime*100/realOutputDuration), 'f', 1, 64), "%")
			}

			// if the accumulated text contains the frame, process it
			if strings.Contains(scannerTextAccum, "frame=") {
				currentFrame, _ = strconv.Atoi(strings.Split(strings.Split(scannerTextAccum, "\n")[0], "=")[1])
				avgFramerate = strconv.FormatFloat(float64(currentFrame)/time.Since(startTime).Seconds(), 'f', 1, 64)

				// if it's been one second since the last fps over one second update, update it again
				if time.Since(changeStartTime).Seconds() >= 1 {
					lastSecAvgFramerate = strconv.FormatFloat(float64(currentFrame-oldFrame)/time.Since(changeStartTime).Seconds(), 'f', 1, 64)
					oldFrame = currentFrame
					changeStartTime = time.Now()
				}
			}

			// if the accumulated text contains the speed, print the time, eta, fps, and fps over the last second
			if strings.Contains(scannerTextAccum, "speed=") {
				fmt.Print(" time: ", utils.TrimTime(fullTime))
				fmt.Print(" ETA: ", utils.TrimTime(utils.FormatTime(eta)))
				fmt.Print(" fps: ", avgFramerate)
				fmt.Print("\033[0J") // this overwrites any previous text that was printed, even if unlikely
				fmt.Print(" fp1s: ", lastSecAvgFramerate)
				fmt.Print("\n")
			}

			// reset the accumulated text
			scannerTextAccum = ""
		}
	}

	cmd.Wait()

	// if the progress bar length is greater than 0, print the progress bar one last time at 100%
	if progbarLength > 0 {
		fmt.Print("\033[1A\033[0J", utils.ProgressBar(realOutputDuration, realOutputDuration, progbarLength))
	} else {
		fmt.Print("\033[1A\033[0J")
	}

	// print the percentage complete (100% by now), time, ETA (hopfully 0s), fps, and fps over the last second
	fmt.Print(
		" 100.0%",
		" time: ", utils.TrimTime(fullTime),
		" ETA: ", utils.TrimTime(utils.FormatTime(eta)),
		" fps: ", avgFramerate,
		" fp1s: ", lastSecAvgFramerate,
		"\n",
	)
}

func imageMunch(input string, inputData ffprobe.MediaData) {
	if debug {
		log.Print("resolution is ", inputData.Width, " by ", inputData.Height)
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

	// calculate the output resolution
	outputWidth := int(math.Round(float64(inputData.Width)*outScale*float64(aspectWidth))/2) * 2
	outputHeight := int(math.Round(float64(inputData.Height)*outScale*float64(aspectHeight))/2) * 2

	// set up the ffmpeg filter for -filter_complex
	var filter strings.Builder

	filter.WriteString("scale=" + strconv.Itoa(outputWidth) + ":" + strconv.Itoa(outputHeight) + ",setsar=1:1")

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
			log.Fatal("\033[4m\033[31mFatal Error\033[24m: unable to create temp/font.ttf")
		}
		log.Print(fontPath)
		filter.WriteString(",drawtext=fontfile='temp/font.ttf':text='" + text + "':fontcolor=" + textColor + ":borderw=(" + strconv.FormatFloat(fontSize*float64(outputWidth/100), 'f', -1, 64) + "/12):fontsize=" + strconv.FormatFloat(fontSize*float64(outputWidth/100), 'f', -1, 64) + ":x=(w-(tw))*(" + strconv.Itoa(textposx) + "/100):y=(h-(th))*(" + strconv.Itoa(textposy) + "/100)")
		if debug {
			log.Print("text is ", text)
			log.Print(",drawtext=fontfile='temp/font.ttf':text='" + text + "':fontcolor=" + textColor + ":borderw=(" + strconv.Itoa(outputWidth/len(text)*2) + "/12):fontsize=" + strconv.Itoa(outputWidth/len(text)*2) + ":x=(w-(tw))*(" + strconv.Itoa(textposx) + "/100):y=(h-(th))*(" + strconv.Itoa(textposy) + "/100)")
		}
	}

	// staring ffmpeg args
	args := []string{
		"-y", // forces overwrite of existing file, if one does exist
		"-loglevel", loglevel,
		"-hide_banner",
		"-progress", "-",
		"-stats_period", strconv.FormatFloat(updateSpeed, 'f', -1, 64),
		"-i", input,
		"-c:v", "mjpeg",
		"-q:v", "31",
		"-frames:v", "1",
	}

	// if any filters are being used, add them
	if len(filter.String()) != 0 {
		args = append(args, "-filter_complex", filter.String())
	}

	args = append(args, output) // add the output file to the ffmpeg args

	if debug {
		log.Print(args)
	}

	fmt.Println("\033[94mEncoding file\033[36m", input, "\033[94mto\033[36m", output, "\033[0m") // print the input and output file

	// start ffmpeg for encoding
	cmd := exec.Command("ffmpeg", args...)
	cmd.Start()
	cmd.Wait()

	var newOutput string
	oldOutput := output

	startTime := time.Now()
	eta := 0.0

	if loopNum > 1 {
		if err := os.MkdirAll("temp", os.ModePerm); err != nil {
			log.Fatal(err)
		}

		newOutput = "temp/loop1.jpg"
		cmd = exec.Command(
			"ffmpeg",
			"-y", // forces overwrite of existing file, if one does exist
			"-loglevel", loglevel,
			"-hide_banner",
			"-progress", "-",
			"-stats_period", strconv.FormatFloat(updateSpeed, 'f', -1, 64),
			"-i", output,
			"-c:v", "mjpeg",
			"-q:v", "31",
			"-frames:v", "1",
			newOutput,
		)
		cmd.Start()
		cmd.Wait()

		eta = getETA(startTime, 1, 1)

		// if the progress bar length is not set, set it to the length of the longest possible progress bar
		if unspecifiedProgbarSize {
			progbarLength = utils.ProgbarSize(len(" " + (strconv.FormatFloat((1*100/float64(loopNum)), 'f', 1, 64) + "%" + " ETA: " + utils.TrimTime(utils.FormatTime(eta)))))
		}

		// if the progress bar length is greater than 0, print the progress bar
		fmt.Println()
		if progbarLength > 0 {
			fmt.Print("\033[1A", utils.ProgressBar(1, float64(loopNum), progbarLength))
		} else {
			fmt.Print("\033[1A\033[0J")
		}

		// print the percentage complete
		fmt.Println(" ", strconv.FormatFloat((1*100/float64(loopNum)), 'f', 1, 64)+"%"+" ETA: "+utils.TrimTime(utils.FormatTime(eta))+"\033[0J")

		if debug {
			log.Print("libwebp:")
			log.Print("compression level: " + strconv.Itoa(int(float64(1/float64(preset))*7.0)-1))
			log.Print("quality: " + strconv.Itoa(((preset)*12)+16))
			log.Print("libx264:")
			log.Print("crf: " + strconv.Itoa(int(float64(preset)*(51.0/7.0))))
			log.Print("mjpeg:")
			log.Print("q:v: " + strconv.Itoa(int(float64(preset)*3.0)+10))
		}

		for i := 2; i < loopNum-1; i++ {
			oldOutput = newOutput
			newOutput = "temp/loop" + strconv.Itoa(i) + ".png"
			cmd = exec.Command(
				"ffmpeg",
				"-y", // forces overwrite of existing file, if one does exist
				"-loglevel", loglevel,
				"-hide_banner",
				"-progress", "-",
				"-stats_period", strconv.FormatFloat(updateSpeed, 'f', -1, 64),
				"-i", oldOutput,
				"-c:v", "libwebp",
				"-compression_level", strconv.Itoa(int(float64(1/float64(preset))*7.0)-1),
				"-quality", strconv.Itoa(((preset)*12)+16),
				"-frames:v", "1",
				newOutput,
			)
			cmd.Start()
			cmd.Wait()
			os.Remove(oldOutput)

			eta = getETA(startTime, float64(i), float64(loopNum))
			if unspecifiedProgbarSize {
				progbarLength = utils.ProgbarSize(len(" " + (strconv.FormatFloat((float64(i)*100/float64(loopNum)), 'f', 1, 64) + "%" + " ETA: " + utils.TrimTime(utils.FormatTime(eta)))))
			}

			// if the progress bar length is greater than 0, print the progress bar
			if progbarLength > 0 {
				fmt.Print("\033[1A", utils.ProgressBar(float64(i), float64(loopNum), progbarLength))
			} else {
				fmt.Print("\033[1A\033[0J")
			}

			// print the percentage complete
			fmt.Println(" ", strconv.FormatFloat((float64(i)*100/float64(loopNum)), 'f', 1, 64)+"%"+" ETA: "+utils.TrimTime(utils.FormatTime(eta))+"\033[0J")

			i++
			oldOutput = newOutput
			newOutput = "temp/loop" + strconv.Itoa(i) + ".png"
			cmd = exec.Command(
				"ffmpeg",
				"-y", // forces overwrite of existing file, if one does exist
				"-loglevel", loglevel,
				"-hide_banner",
				"-progress", "-",
				"-stats_period", strconv.FormatFloat(updateSpeed, 'f', -1, 64),
				"-i", oldOutput,
				"-c:v", "libx264",
				"-crf", strconv.Itoa(int(float64(preset)*(51.0/7.0))),
				"-frames:v", "1",
				newOutput,
			)
			cmd.Start()
			cmd.Wait()
			os.Remove(oldOutput)

			eta = getETA(startTime, float64(i), float64(loopNum))
			if unspecifiedProgbarSize {
				progbarLength = utils.ProgbarSize(len(" " + (strconv.FormatFloat((float64(i)*100/float64(loopNum)), 'f', 1, 64) + "%" + " ETA: " + utils.TrimTime(utils.FormatTime(eta)))))
			}

			// if the progress bar length is greater than 0, print the progress bar
			if progbarLength > 0 {
				fmt.Print("\033[1A", utils.ProgressBar(float64(i), float64(loopNum), progbarLength))
			} else {
				fmt.Print("\033[1A\033[0J")
			}

			// print the percentage complete
			fmt.Println(" ", strconv.FormatFloat((float64(i)*100/float64(loopNum)), 'f', 1, 64)+"%"+" ETA: "+utils.TrimTime(utils.FormatTime(eta))+"\033[0J")

			i++
			oldOutput = newOutput
			newOutput = "temp/loop" + strconv.Itoa(i) + ".jpg"
			cmd = exec.Command(
				"ffmpeg",
				"-y", // forces overwrite of existing file, if one does exist
				"-loglevel", loglevel,
				"-hide_banner",
				"-progress", "-",
				"-stats_period", strconv.FormatFloat(updateSpeed, 'f', -1, 64),
				"-i", oldOutput,
				"-c:v", "mjpeg",
				"-q:v", strconv.Itoa(int(float64(preset)*3.0)+10),
				"-frames:v", "1",
				newOutput,
			)
			cmd.Start()
			cmd.Wait()
			os.Remove(oldOutput)

			eta = getETA(startTime, float64(i), float64(loopNum))
			if unspecifiedProgbarSize {
				progbarLength = utils.ProgbarSize(len(" " + (strconv.FormatFloat((float64(i)*100/float64(loopNum)), 'f', 1, 64) + "%" + " ETA: " + utils.TrimTime(utils.FormatTime(eta)))))
			}

			// if the progress bar length is greater than 0, print the progress bar
			if progbarLength > 0 {
				fmt.Print("\033[1A", utils.ProgressBar(float64(i), float64(loopNum), progbarLength))
			} else {
				fmt.Print("\033[1A\033[0J")
			}

			// print the percentage complete
			fmt.Println(" ", strconv.FormatFloat((float64(i)*100/float64(loopNum)), 'f', 1, 64)+"%"+" ETA: "+utils.TrimTime(utils.FormatTime(eta))+"\033[0J")
		}

		oldOutput = newOutput
		cmd = exec.Command(
			"ffmpeg",
			"-y", // forces overwrite of existing file, if one does exist
			"-loglevel", loglevel,
			"-hide_banner",
			"-progress", "-",
			"-stats_period", strconv.FormatFloat(updateSpeed, 'f', -1, 64),
			"-i", oldOutput,
			"-c:v", "mjpeg",
			"-q:v", "31",
			"-frames:v", "1",
			output,
		)
		cmd.Start()
		cmd.Wait()
		os.Remove(oldOutput)

		eta = getETA(startTime, float64(loopNum), float64(loopNum))
		if unspecifiedProgbarSize {
			progbarLength = utils.ProgbarSize(len(" " + (strconv.FormatFloat((float64(loopNum)*100/float64(loopNum)), 'f', 1, 64) + "%" + " ETA: " + utils.TrimTime(utils.FormatTime(eta)))))
		}

		// if the progress bar length is greater than 0, print the progress bar
		if progbarLength > 0 {
			fmt.Print("\033[1A", utils.ProgressBar(float64(loopNum), float64(loopNum), progbarLength))
		} else {
			fmt.Print("\033[1A\033[0J")
		}

		// print the percentage complete
		fmt.Println(" ", strconv.FormatFloat((float64(loopNum)*100/float64(loopNum)), 'f', 1, 64)+"%"+" ETA: "+utils.TrimTime(utils.FormatTime(eta))+"\033[0J")
	}
}

func getETA(startingTime time.Time, current float64, total float64) float64 {
	return time.Since(startingTime).Seconds() * (total - current) / current
}
