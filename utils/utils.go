package utils

import (
	"math"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

func TrimTime(time string) string {
	firstchar := strings.Split(time, ":")[0]
	if firstchar == "00" {
		time = strings.Split(time, ":")[1] + ":" + strings.Split(time, ":")[2]
		firstchar = strings.Split(time, ":")[0]
		if firstchar == "00" {
			time = strings.Split(time, ":")[1]
			firstchar = time[0:1]
			if firstchar == "0" {
				time = time[1:]
			}
		}
	}
	return time
}

func FormatTime(time float64) string {
	hour := strconv.Itoa(int(time / 3600))
	minute := strconv.Itoa(int(math.Mod(time, 3600) / 60))
	second := strconv.FormatFloat(math.Mod(time, 60), 'f', 1, 64)
	if len(minute) == 1 {
		minute = "0" + minute
	}
	if len(hour) == 1 {
		hour = "0" + hour
	}
	if len(strings.Split(second, ".")[0]) == 1 {
		second = "0" + second
	}
	return hour + ":" + minute + ":" + second + "s"
}

func ProgressBar(done float64, total float64, length int) string {
	// add comments to this code
	var bar string = "["                                // start the bar with a bracket
	var filled float64 = done / total * float64(length) // calculate the units of the bar that should be filled
	if done >= 0.995*total {
		filled = float64(length)
	}
	var percentDone int = int(filled)          // convert the filled units to an int
	var percentLeft int = length - percentDone // calculate the units of the bar that should be empty
	for i := 0; i < percentDone; i++ {         // fill the bar
		bar += "\033[92m─"
	}
	if done == total {
		bar += "─\033[0m" // if the bar is full, doesn't use a leading character
	} else {
		bar += ">\033[90m" // leading character when filling the bar
	}
	for i := 0; i < (percentLeft); i++ { // fill the rest of the bar
		bar += "─"
	}
	bar += "\033[0m]" // add the closing bracket to the bar
	return bar        // return the bar
}

func ProgbarSize(length int) int {
	terminalWidth, _, _ := term.GetSize(int(os.Stdout.Fd()))
	return terminalWidth - 7 - length
}
