# Quality Muncher Go
Quality Muncher Go (aka QM:GO) is a program written to make low quality videos, images, GIFs, and audios.

## Usage
Below are the flags. The only needed flag is the input, as all of the others have default values or are disabled by default.
```
  -i, --input string          Input file
  -o, --output string         Output file
  -d, --debug                 Print out debug information
      --no-video              Output with no video
      --no-audio              Output with no audio
  -p, --preset int            Quality preset (default 4)
      --start float           Start time of the output
      --end float             End time of the output
      --duration float        Specify the duration of the output
  -v, --volume int            Amount to increase or decrease the volume by, in dB
  -s, --scale float           Specify the output scale (default -1)
--vb, --video-bitrate int   Specify the video bitrate divisor
--ab, --audio-bitrate int   Specify the audio bitrate divisor
      --stretch string        Modify the existing aspect ratio (default "1:1")
      --fps int               Specify the output fps
      --speed int             Video and audio speed
  -z, --zoom float            Amount to zoom in or out
      --fade-in float         Fade in duration
      --fade-out float        Fade out duration
      --stutter int           Randomize the order of a frames
      --vignette float        Amount of vignette
      --corrupt int           Corrupt the output
      --interlace             Interlace the output
      --lagfun                Force darker pixels to update slower
      --resample              Blend frames together instead of dropping them
```

## Builds
I'll release them every now and then, but you can always build your own as it's very simple.

## Disclaimer
This is barely functional and still VERY much in the alpha stages of development.