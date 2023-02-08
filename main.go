/*
Quality Muncher Go (QM:GO) - a program to worsen the quality of media files
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
	overwrite                 bool
	progbarLength             int
	imagePasses               int
	loglevel                  string
	updateSpeed               float64
	noVideo, noAudio          bool
	replaceAudio              string
	preset                    int
	start, end, outDuration   float64
	volume                    int
	earrape                   bool
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
	fry                       int
	interlace                 bool
	lagfun                    bool
	resample                  bool
	text, textFont, textColor string
	textposx, textposy        int
	fontSize                  float64

	// other variables
	unspecifiedProgbarSize bool
)

func init() {
	pflag.CommandLine.SortFlags = false
	pflag.StringSliceVarP(&inputs, "input", "i", []string{""}, "Specify the input file(s)")
	pflag.StringVarP(&output, "output", "o", "", "Specify the output file")
	pflag.BoolVarP(&debug, "debug", "d", false, "Print out debug information")
	pflag.BoolVarP(&overwrite, "overwrite", "y", false, "Overwrite the output file if it exists instead of prompting for confirmation")
	pflag.IntVar(&progbarLength, "progress-bar", -1, "Length of progress bar, defaults based on terminal width")
	pflag.IntVar(&imagePasses, "loop", 1, "Number of time to compress the input. ONLY USED FOR IMAGES.")
	pflag.StringVar(&loglevel, "loglevel", "error", "Specify the log level for ffmpeg")
	pflag.Float64Var(&updateSpeed, "update-speed", 0.0167, "Specify the speed at which stats will be updated")
	pflag.BoolVar(&noVideo, "no-video", false, "Produces an output with no video")
	pflag.BoolVar(&noAudio, "no-audio", false, "Produces an output with no audio")
	pflag.StringVar(&replaceAudio, "replace-audio", "", "Replace the audio with the specified file")
	pflag.IntVarP(&preset, "preset", "p", 4, "Specify the quality preset (1-7, higher = worse)")
	pflag.Float64Var(&start, "start", 0, "Specify the start time of the output")
	pflag.Float64Var(&end, "end", -1, "Specify the end time of the output, cannot be used when duration is specified")
	pflag.Float64Var(&outDuration, "duration", -1, "Specify the duration of the output, cannot be used when end is specified")
	pflag.IntVarP(&volume, "volume", "v", 0, "Specify the amount to increase or decrease the volume by, in dB")
	pflag.BoolVar(&earrape, "earrape", false, "Heavily and extremely distort the audio (aka earrape). BE WARNED: VOLUME WILL BE SUBSTANTIALLY INCREASED.")
	pflag.Float64VarP(&outScale, "scale", "s", -1, "Specify the output scale")
	pflag.IntVar(&videoBrDiv, "video-bitrate", -1, "Specify the video bitrate divisor (higher = worse)")
	pflag.IntVar(&videoBrDiv, "vb", videoBrDiv, "Shorthand for --video-bitrate")
	pflag.IntVar(&audioBrDiv, "audio-bitrate", -1, "Specify the audio bitrate divisor (higher = worse)")
	pflag.IntVar(&audioBrDiv, "ab", audioBrDiv, "Shorthand for --audio-bitrate")
	pflag.StringVar(&stretch, "stretch", "1:1", "Modify the existing aspect ratio")
	pflag.IntVar(&outFPS, "fps", -1, "Specify the output fps (lower = worse)")
	pflag.Float64Var(&speed, "speed", 1.0, "Specify the video and audio speed")
	pflag.Float64VarP(&zoom, "zoom", "z", 1, "Specify the amount to zoom in or out")
	pflag.Float64Var(&fadein, "fade-in", 0, "Fade in duration")
	pflag.Float64Var(&fadeout, "fade-out", 0, "Fade out duration")
	pflag.IntVar(&stutter, "stutter", 0, "Randomize the order of a frames (higher = more stutter)")
	pflag.Float64Var(&vignette, "vignette", 0, "Specify the amount of vignette")
	pflag.IntVar(&corrupt, "corrupt", 0, "Corrupt the output (1-10, higher = worse)")
	pflag.IntVar(&fry, "deep-fry", 0, "Deep-fry the output (1-10, higher = worse)")
	pflag.BoolVar(&interlace, "interlace", false, "Interlace the output")
	pflag.BoolVar(&lagfun, "lagfun", false, "Force darker pixels to update slower")
	pflag.BoolVar(&resample, "resample", false, "Blend frames together instead of dropping them")
	pflag.StringVarP(&text, "text", "t", "", "Text to add (if empty, no text)")
	pflag.StringVar(&textFont, "text-font", "arial", "Text to add (if empty, no text)")
	pflag.StringVar(&textColor, "text-color", "white", "Text color")
	pflag.IntVar(&textposx, "text-pos-x", 50, "horizontal position of text (0 is far left, 100 is far right)")
	pflag.IntVar(&textposy, "text-pos-y", 90, "vertical position of text (0 is top, 100 is bottom)")
	pflag.Float64Var(&fontSize, "font-size", 12, "Font size (scales with output width)")
	pflag.Parse()

	// check for invalid input
	if inputs[0] == "" {
		log.Fatal("No input was specified")
	}
	// negative start time would give an output with no video so throw an error
	if start < 0 {
		log.Fatal("Start time cannot be negative")
	}
	// if start time is greater than or equal to end time, output length would be 0 so throw an error
	if start >= end && end != -1 {
		log.Fatal("Start time cannot be greater than or equal to end time")
	}
	if outDuration != -1 && end != -1 {
		log.Fatal("Cannot specify both duration and end time")
	}

	// make sure that we know when the progress bar length is unspecified so we can set it automatically
	if progbarLength == -1 {
		unspecifiedProgbarSize = true
	} else {
		unspecifiedProgbarSize = false
	}
}

func main() {
	programStartTime := time.Now()
	// throw out all flags if debug is enabled
	if debug {
		log.Println("throwing all flags out")
		log.Println(
			inputs,
			output,
			debug,
			progbarLength,
			imagePasses,
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

	// loop for each provided input because we support queueing multiple inputs
	for i, input := range inputs {
		// check if input file exists
		_, err := os.Stat(input)
		// if it doesn't exist, skip this input and go to the next one
		if err != nil {
			if os.IsNotExist(err) {
				log.Println("\033[4m\033[31mError\033[24m: input file \033[91m" + input + " \033[31mdoes not exist\033[0m")
				continue
			} else {
				// the input file exists but can't be accessed for some reason
				fmt.Println("\033[4m\033[38;2;250;169;30mWarning\033[24m: Input file \033[38;2;250;182;37m" + input + " \033[38;2;250;169;30mmight not be accessible\033[0m")
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

		// i have no clue how this works but i remember writing it and it works so i'm not touching it
		// maybe i should stop using triple negatives in my variable names/code
		renderVideo := !noVideo
		renderAudio := !noAudio
		if !noVideo {
			renderVideo = Stream(input, "v:0")
		}
		if len(string(replaceAudio)) == 0 {
			if !noAudio {
				renderAudio = Stream(input, "a:0")
			}
		}

		// get input data: width, height, duration, and framerate
		inputData, _ := ffprobe.ProbeData(input)

		// assume that the input is a video unless proven otherwise
		isImage := false
		outExt := ".mp4"
		if renderAudio && !renderVideo {
			outExt = ".mp3"
		}

		if !renderVideo && !renderAudio {
			log.Println("\033[4m\033[mError\033[24m: Cannot encode video without audio or video streams\033[0m")
			continue
		}

		// if the duration is less than 1.0 seconds, check the frame count\
		// only check the frame count when needed because it's slow
		if inputData.Duration < 1.0 && renderVideo {
			if ffprobe.FrameCount(input) == 1 { // if there's only one frame, it's an image
				log.Print("duration: ", inputData.Duration)
				isImage = true
				outExt = ".jpg"
			}
		}

		// set the output file name if it isn't explicitly set or if there are multiple inputs
		if len(inputs) > 1 {
			output = strings.TrimSuffix(input, filepath.Ext(input)) + " (Quality Munched)" + outExt
		}
		if output == "" {
			output = strings.TrimSuffix(input, filepath.Ext(input)) + " (Quality Munched)" + outExt
			if debug {
				log.Println("No output was specified, using input name plus (Quality Munched)")
				log.Println("output: " + output)
			}
		} else {
			if !strings.Contains(output, ":") {
				output = filepath.Dir(input) + "/" + output
			}
		}

		// check if output file already exists
		_, outExistErr := os.Stat(output)
		if outExistErr == nil {
			if debug {
				log.Print("output file already exists")
			}
			var confirm string
			if !overwrite {
				fmt.Println("\033[4m\033[38;2;250;169;30mWarning\033[24m: The output file\033[38;2;250;182;37m", output, "\033[38;2;250;169;30malready exists! Overwrite? [Y/N]\033[0m")
				fmt.Scanln(&confirm) // get user input, confirming that they want to overwrite the output file
				if confirm != "Y" && confirm != "y" {
					log.Println("Aborted by user - output file already exists")
					continue
				}
			}
		}

		startTime := time.Now()
		if isImage {
			if debug {
				log.Println("input is an image")
			}
			imageMunch(input, inputData, i+1, len(inputs)) // encode the image
		} else {
			if start >= inputData.Duration {
				log.Fatal("Start time cannot be greater than or equal to input duration")
			}
			videoMunch(input, inputData, i+1, len(inputs), renderVideo, renderAudio) // encode the video
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
	log.Println("\033[92mTotal time elapsed:", utils.TrimTime(utils.FormatTime(time.Since(programStartTime).Seconds()))+"\033[0m")
}

func videoMunch(input string, inputData ffprobe.MediaData, inNum int, totalNum int, renderVideo bool, renderAudio bool) {
	if !renderVideo {
		inputData.Width = 1
		inputData.Height = 1
		inputData.Framerate = 1.0
	}
	// get input resolution
	if debug {
		log.Print("resolution is ", inputData.Width, " by ", inputData.Height)
	}

	// fps and tmix (frame resampling) filters/calculations
	if outFPS == -1 {
		outFPS = 24 - (3 * preset)
	}
	if debug {
		log.Print("Output FPS is ", outFPS)
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

	// if the output scale isn't explicitly set, calculate it using the given preset
	if outScale == -1 {
		outScale = 1.0 / float64(preset)
	}
	if debug {
		log.Print("Output scale is ", outScale)
	}

	// calculate the output resolution
	outputWidth, outputHeight := newResolution(inputData.Width, inputData.Height)

	var bitrate int
	// calculate the video bitrate
	if videoBrDiv != -1 {
		bitrate = outputHeight * outputWidth * int(math.Sqrt(float64(outFPS))) / videoBrDiv
	} else {
		bitrate = outputHeight * outputWidth * int(math.Sqrt(float64(outFPS))) / preset
	}

	var audioBitrate int
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
	if renderVideo {
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
			filter.WriteString(makeTextFilter(outputWidth, text, textFont, fontSize, textColor, textposx, textposy))
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

		if fry != 0 {
			filter.WriteString("," + "eq=saturation=" + strconv.FormatFloat(float64(fry)*0.15+0.85, 'f', -1, 64) + ":contrast=" + strconv.Itoa(fry) + ",unsharp=5:5:1.25:5:5:" + strconv.FormatFloat(float64(fry)/6.66, 'f', -1, 64) + ",noise=alls=" + strconv.Itoa(fry*5) + ":allf=t")
			if debug {
				log.Print("fry is ", ","+"eq=saturation="+strconv.FormatFloat(float64(fry)*0.15+0.85, 'f', -1, 64)+":contrast="+strconv.Itoa(fry)+",unsharp=5:5:1.25:5:5:"+strconv.FormatFloat(float64(fry)/6.66, 'f', -1, 64)+",noise=alls="+strconv.Itoa(fry*5)+":allf=t")
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
	if renderAudio {
		if earrape {
			filter.WriteString(";aeval=sgn(val(5)):c=same")
			if debug {
				log.Print("earrape is true")
			}
		}

		if volume != 0 {
			if earrape {
				filter.WriteString(",volume=" + strconv.Itoa(volume) + "dB")
			} else {
				filter.WriteString(";volume=" + strconv.Itoa(volume) + "dB")
			}
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

	var corruptAmount int
	var corruptFilter string
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

	if start != 0 { // if start is specified
		args = append(args, "-ss", strconv.FormatFloat(start, 'f', -1, 64)) // -ss is the start time
	}

	if end != -1 { // if end is specified
		outDuration = end - start
	}

	if outDuration != -1 { // if the duration is specified
		args = append(args, "-t", strconv.FormatFloat(outDuration, 'f', -1, 64)) // -t sets the duration
	}

	// remove video if the user wants no video
	if !renderVideo {
		args = append(args, "-vn")
		if debug {
			log.Print("no video")
		}
	}

	// remove audio if noAudio is true
	if !renderAudio {
		args = append(args, "-an") // removes audio
		if debug {
			log.Print("no audio")
		}
	}

	// add the input to the ffmpeg args
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
	if renderVideo {
		args = append(args,
			"-preset", "ultrafast",
			"-shortest",
			"-c:v", "libx264",
			"-b:v", strconv.Itoa(int(bitrate)),
			"-c:a", "aac",
			"-b:a", strconv.Itoa(int(audioBitrate)),
		)
	} else {
		args = append(args,
			"-shortest",
			"-b:v", strconv.Itoa(int(bitrate)),
			"-c:a", "libmp3lame",
			"-b:a", strconv.Itoa(int(audioBitrate)),
		)
	}

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

	// the [x/y] thing when encoding multiple files
	var encodingFileOutOf string
	if totalNum != 1 {
		encodingFileOutOf = "[" + strconv.Itoa(inNum) + "/" + strconv.Itoa(totalNum) + "] "
	}

	fmt.Println("\033[94m"+encodingFileOutOf+"Encoding file\033[36m", input, "\033[94mto\033[36m", output+"\033[0m") // print the input and output file

	// start ffmpeg for encoding
	cmd := exec.Command("ffmpeg", args...)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	cmd.Start()

	// variables for progress bar and stats
	scannerTextAccum := " "
	scannerrorTextAccum := " "
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
	scanner := bufio.NewScanner(stdout)
	scannerror := bufio.NewScanner(stderr)
	scanner.Split(bufio.ScanRunes)
	scannerror.Split(bufio.ScanRunes)
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

	for scannerror.Scan() {
		scannerrorTextAccum += scannerror.Text()
	}

	cmd.Wait()

	// if the progress bar length is greater than 0, print the progress bar one last time at 100%
	if progbarLength > 0 {
		fmt.Print("\033[1A\033[0J", utils.ProgressBar(realOutputDuration, realOutputDuration, progbarLength))
	} else {
		fmt.Print("\033[1A\033[0J")
	}

	// print the percentage complete (100% by now), time, ETA (hopfully 0s), fps, and fps over the last second

	if len(scannerrorTextAccum) > 1 {
		log.Print("\n\n\033[31m\033[4mPossible FFmpeg Error:\033[24m\033[31m", scannerrorTextAccum, "\033[0m")
	} else {
		fmt.Print(
			" 100.0%",
			" time: ", utils.TrimTime(fullTime),
			" ETA: ", utils.TrimTime(utils.FormatTime(eta)),
			" fps: ", avgFramerate,
			" fp1s: ", lastSecAvgFramerate,
			"\n",
		)
	}

}

func imageMunch(input string, inputData ffprobe.MediaData, inNum int, totalNum int) {
	if debug {
		log.Print("resolution is ", inputData.Width, " by ", inputData.Height)
	}

	if outScale == -1 {
		outScale = 1.0 / float64(preset)
	}
	if debug {
		log.Print("Output scale is ", outScale)
	}

	outputWidth, outputHeight := newResolution(inputData.Width, inputData.Height)

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
		filter.WriteString(makeTextFilter(outputWidth, text, textFont, fontSize, textColor, textposx, textposy))
	}

	if fry != 0 {
		filter.WriteString("," + "eq=saturation=" + strconv.FormatFloat(float64(fry)*0.15+0.85, 'f', -1, 64) + ":contrast=" + strconv.Itoa(fry) + ",unsharp=5:5:1.25:5:5:" + strconv.FormatFloat(float64(fry)/6.66, 'f', -1, 64) + ",noise=alls=" + strconv.Itoa(fry*5) + ":allf=t")
		if debug {
			log.Print("fry is ", ","+"eq=saturation="+strconv.FormatFloat(float64(fry)*0.15+0.85, 'f', -1, 64)+":contrast="+strconv.Itoa(fry)+",unsharp=5:5:1.25:5:5:"+strconv.FormatFloat(float64(fry)/6.66, 'f', -1, 64)+",noise=alls="+strconv.Itoa(fry*5)+":allf=t")
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

	// the [x/y] thing when encoding multiple files
	var encodingFileOutOf string
	if totalNum != 1 {
		encodingFileOutOf = "[" + strconv.Itoa(inNum) + "/" + strconv.Itoa(totalNum) + "] "
	}

	fmt.Println("\033[94m"+encodingFileOutOf+"Encoding file\033[36m", input, "\033[94mto\033[36m", output+"\033[0m") // print the input and output file

	// start ffmpeg for encoding
	cmd := exec.Command("ffmpeg", args...)
	cmd.Start()
	cmd.Wait()

	var newOutput string

	startTime := time.Now()
	eta := 0.0

	if imagePasses > 1 {
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
			progbarLength = utils.ProgbarSize(len(" " + (strconv.FormatFloat((1*100/float64(imagePasses)), 'f', 1, 64) + "%" + " ETA: " + utils.TrimTime(utils.FormatTime(eta)))))
		}

		// if the progress bar length is greater than 0, print the progress bar
		if progbarLength > 0 {
			fmt.Print(utils.ProgressBar(1, float64(imagePasses), progbarLength))
		} else {
			fmt.Print("\033[0J")
		}

		// print the percentage complete
		fmt.Println(" ", strconv.FormatFloat((1*100/float64(imagePasses)), 'f', 1, 64)+"%"+" ETA: "+utils.TrimTime(utils.FormatTime(eta))+"\033[0J")

		if debug {
			log.Print("libwebp:")
			log.Print("compression level: " + strconv.Itoa(int(float64(1/float64(preset))*7.0)-1))
			log.Print("quality: " + strconv.Itoa(((preset)*12)+16))
			log.Print("libx264:")
			log.Print("crf: " + strconv.Itoa(int(float64(preset)*(51.0/7.0))))
			log.Print("mjpeg:")
			log.Print("q:v: " + strconv.Itoa(int(float64(preset)*3.0)+10))
		}

		var oldOutput string

		for i := 2; i < imagePasses-1; i++ {
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

			eta = getETA(startTime, float64(i), float64(imagePasses))
			if unspecifiedProgbarSize {
				progbarLength = utils.ProgbarSize(len(" " + (strconv.FormatFloat((float64(i)*100/float64(imagePasses)), 'f', 1, 64) + "%" + " ETA: " + utils.TrimTime(utils.FormatTime(eta)))))
			}

			// if the progress bar length is greater than 0, print the progress bar
			if progbarLength > 0 {
				fmt.Print("\033[1A", utils.ProgressBar(float64(i), float64(imagePasses), progbarLength))
			} else {
				fmt.Print("\033[1A\033[0J")
			}

			// print the percentage complete
			fmt.Println(" ", strconv.FormatFloat((float64(i)*100/float64(imagePasses)), 'f', 1, 64)+"%"+" ETA: "+utils.TrimTime(utils.FormatTime(eta))+"\033[0J")

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

			eta = getETA(startTime, float64(i), float64(imagePasses))
			if unspecifiedProgbarSize {
				progbarLength = utils.ProgbarSize(len(" " + (strconv.FormatFloat((float64(i)*100/float64(imagePasses)), 'f', 1, 64) + "%" + " ETA: " + utils.TrimTime(utils.FormatTime(eta)))))
			}

			// if the progress bar length is greater than 0, print the progress bar
			if progbarLength > 0 {
				fmt.Print("\033[1A", utils.ProgressBar(float64(i), float64(imagePasses), progbarLength))
			} else {
				fmt.Print("\033[1A\033[0J")
			}

			// print the percentage complete
			fmt.Println(" ", strconv.FormatFloat((float64(i)*100/float64(imagePasses)), 'f', 1, 64)+"%"+" ETA: "+utils.TrimTime(utils.FormatTime(eta))+"\033[0J")

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

			eta = getETA(startTime, float64(i), float64(imagePasses))
			if unspecifiedProgbarSize {
				progbarLength = utils.ProgbarSize(len(" " + (strconv.FormatFloat((float64(i)*100/float64(imagePasses)), 'f', 1, 64) + "%" + " ETA: " + utils.TrimTime(utils.FormatTime(eta)))))
			}

			// if the progress bar length is greater than 0, print the progress bar
			if progbarLength > 0 {
				fmt.Print("\033[1A", utils.ProgressBar(float64(i), float64(imagePasses), progbarLength))
			} else {
				fmt.Print("\033[1A\033[0J")
			}

			// print the percentage complete
			fmt.Println(" ", strconv.FormatFloat((float64(i)*100/float64(imagePasses)), 'f', 1, 64)+"%"+" ETA: "+utils.TrimTime(utils.FormatTime(eta))+"\033[0J")
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

		eta = getETA(startTime, float64(imagePasses), float64(imagePasses))
		if unspecifiedProgbarSize {
			progbarLength = utils.ProgbarSize(len(" " + (strconv.FormatFloat((float64(imagePasses)*100/float64(imagePasses)), 'f', 1, 64) + "%" + " ETA: " + utils.TrimTime(utils.FormatTime(eta)))))
		}

		// if the progress bar length is greater than 0, print the progress bar
		if progbarLength > 0 {
			fmt.Print("\033[1A", utils.ProgressBar(float64(imagePasses), float64(imagePasses), progbarLength))
		} else {
			fmt.Print("\033[1A\033[0J")
		}

		// print the percentage complete
		fmt.Println(" ", strconv.FormatFloat((float64(imagePasses)*100/float64(imagePasses)), 'f', 1, 64)+"%"+" ETA: "+utils.TrimTime(utils.FormatTime(eta))+"\033[0J")
	}
}

func getETA(startingTime time.Time, current float64, total float64) float64 {
	return time.Since(startingTime).Seconds() * (total - current) / current
}

func newResolution(inWidth int, inHeight int) (int, int) {
	var outWidth int
	var outHeight int

	// split aspect ratio into 2 values that can be multiplied by width and height
	aspect := strings.Split(stretch, ":")
	aspectWidth, err := strconv.Atoi(aspect[0])
	if err != nil {
		log.Print(err)
	}
	aspectHeight, err := strconv.Atoi(aspect[1])
	if err != nil {
		log.Print(err)
	}

	if outScale == -1 {
		outScale = 1.0 / float64(preset)
	}

	outWidth = int(math.Round(float64(inWidth)*outScale*float64(aspectWidth))/2) * 2
	outHeight = int(math.Round(float64(inHeight)*outScale*float64(aspectHeight))/2) * 2

	return outWidth, outHeight
}

func makeTextFilter(
	outWidth int,
	inText string,
	font string,
	size float64,
	color string,
	xpos int,
	ypos int,
) string {
	fontPath, err := findfont.Find(font + ".ttf")
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

	filter := ",drawtext=fontfile='temp/font.ttf':text='" + inText + "':fontcolor=" + color + ":borderw=(" + strconv.FormatFloat(size*float64(outWidth/100), 'f', -1, 64) + "/12):fontsize=" + strconv.FormatFloat(size*float64(outWidth/100), 'f', -1, 64) + ":x=(w-(tw))*(" + strconv.Itoa(xpos) + "/100):y=(h-(th))*(" + strconv.Itoa(ypos) + "/100)"

	if debug {
		log.Println("text is ", inText)
		log.Println("fontpath: ", fontPath)
		log.Println(filter)
	}

	return filter
}

func Stream(input string, stream string) bool {
	args := []string{
		"-i", input,
		"-show_entries", "stream=duration",
		"-select_streams", stream,
		"-of", "csv=p=0",
	}

	cmd := exec.Command("ffprobe", args...)

	out, err := cmd.Output()
	if err != nil {
		return false
	}

	return len(string(out)) != 0
}
