package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

const (
	channels  = 2
	frameRate = 48000
	volume    = 0.25
)

type Song struct {
	Url string
	err error
}

func (s Song) Stream(ctx context.Context, frameSize int) (<-chan []int16, error) {
	ytdl := exec.Command("youtube-dl", "-f", "bestaudio", s.Url, "-o", "-")
	ffmpeg := exec.Command("ffmpeg", "-i", "pipe:0", "-filter:a", fmt.Sprintf("volume=%.2f", volume), "-ar", strconv.Itoa(frameRate), "-ac", strconv.Itoa(channels), "-f", "s16le", "pipe:1")

	// Get all the pipes
	ytStdOut, err := ytdl.StdoutPipe()
	if err != nil {
		return nil, err
	}
	ytStdErr, err := ytdl.StderrPipe()
	if err != nil {
		return nil, err
	}
	ffStdOut, err := ffmpeg.StdoutPipe()
	if err != nil {
		return nil, err
	}
	ffStdErr, err := ffmpeg.StderrPipe()
	if err != nil {
		return nil, err
	}

	// Connect ytdl to ffmpeg
	ffmpeg.Stdin = ytStdOut

	err = ffmpeg.Start()
	if err != nil {
		return nil, err
	}
	err = ytdl.Start()
	if err != nil {
		// TODO: handle errors
		ffmpeg.Process.Kill()
		return nil, err
	}

	// Stream containing sound data
	stream := make(chan []int16, 320)
	// Ok is used to close stream if no data is received
	ok := make(chan bool)

	// wait and close commands when done
	defer func() {
		go func() {
			printBuffer(ytStdErr)
			fmt.Println("--------------------------------")
			printBuffer(ffStdErr)
			// TODO: handle errors
			ffmpeg.Wait()
			ytdl.Wait()
		}()
	}()

	// Function to kill commands
	kill := func() {
		// TODO: handle errors
		if p := ffmpeg.Process; p != nil {
			p.Kill()
		}
		if p := ytdl.Process; p != nil {
			p.Kill()
		}
	}

	// Waits and kills commands if they dont produce any data in time
	go func() {
		select {
		case <-time.NewTimer(30 * time.Second).C:
			fmt.Println("command did not start in time, killing")
			kill()
		case <-ok:
			return
		}
	}()

	// Read loop from ffmpeg
	go func() {
		// Used to close ok stream
		once := sync.Once{}
		defer func() {
			close(stream)
		}()
		for {
			select {
			// If context is done before commands have completed kill them
			case <-ctx.Done():
				kill()
				return
			default:
				// Read data from command and pipe it
				buf := make([]int16, frameSize*channels)
				err = binary.Read(ffStdOut, binary.LittleEndian, &buf)
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					return
				}
				if err != nil {
					fmt.Println("unknown error during buffering")
					return
				}
				stream <- buf
				once.Do(func() {
					close(ok)
				})
			}
		}
	}()
	return stream, nil
}

func NewSong(url string) *Song {
	return &Song{
		Url: url,
	}
}

func printBuffer(closer io.ReadCloser) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(closer)
	newStr := buf.String()
	fmt.Printf("%s \n", newStr)
}
