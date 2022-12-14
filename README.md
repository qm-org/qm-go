# Quality Muncher Go
Quality Muncher Go (aka QM:GO) is a program written to make low quality videos, images, GIFs, and audios.

## Usage
Below are the flags. The only needed flag is the input, as all of the others have default values or are disabled by default.
```
  -i, --input string           Specify the input file
  -o, --output string          Specify the output file
  -d, --debug                  Print out debug information
      --progress-bar int       Length of progress bar, defaults based on terminal width (default -1)
      --loglevel string        Specify the log level for ffmpeg (default "error")
      --update-speed float     Specify the speed at which stats will be updated (default 0.0167)
      --no-video               Produces an output with no video
      --no-audio               Produces an output with no audio
      --replace-audio string   Replace the audio with the specified file
  -p, --preset int             Specify the quality preset (default 4)
      --start float            Specify the start time of the output
      --end float              Specify the end time of the output, cannot be used when duration is specified (default -1)
      --duration float         Specify the duration of the output, cannot be used when end is specified (default -1)
  -v, --volume int             Specify the amount to increase or decrease the volume by, in dB
  -s, --scale float            Specify the output scale (default -1)
      --video-bitrate int      Specify the video bitrate divisor (default -1)
      --vb int                 Shorthand for --video-bitrate (default -1)
      --audio-bitrate int      Specify the audio bitrate divisor (default -1)
      --ab int                 Shorthand for --audio-bitrate (default -1)
      --stretch string         Modify the existing aspect ratio (default "1:1")
      --fps int                Specify the output fps (default -1)
      --speed float            Specify the video and audio speed (default 1)
  -z, --zoom float             Specify the amount to zoom in or out (default 1)
      --fade-in float          Fade in duration
      --fade-out float         Fade out duration
      --stutter int            Randomize the order of a frames
      --vignette float         Specify the amount of vignette
      --corrupt int            Corrupt the output
      --interlace              Interlace the output
      --lagfun                 Force darker pixels to update slower
      --resample               Blend frames together instead of dropping them
  -t, --text string            Text to add (if empty, no text)
      --text-font string       Text to add (if empty, no text) (default "arial")
      --text-color string      Text color (default "white")
      --text-pos-x int         horizontal position of text, 0 is far left, 100 is far right (default 50)
      --text-pos-y int         vertical position of text, 0 is top, 100 is bottom (default 90)
      --font-size float        Font size (scales with output width (default 12)
```

## Builds
I'll release them every now and then, but you can always build your own as it's very simple.

## Disclaimer
This is barely functional and still VERY much in the alpha stages of development.