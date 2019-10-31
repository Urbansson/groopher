package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strconv"
	"syscall"
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
	ctx, cancel := context.WithCancel(ctx)
	ytdl := exec.CommandContext(ctx, "youtube-dl", s.Url, "-o", "-")
	ffmpeg := exec.CommandContext(ctx, "ffmpeg", "-i", "pipe:0", "-filter:a", fmt.Sprintf("volume=%.2f", volume), "-ar", strconv.Itoa(frameRate), "-ac", strconv.Itoa(channels), "-f", "s16le", "pipe:1")

	// Get all the pipes
	ytStdOut, err := ytdl.StdoutPipe()
	if err != nil {
		return nil, err
	}
	ffStdOut, err := ffmpeg.StdoutPipe()
	if err != nil {
		return nil, err
	}

	// Connect ytdl to ffmpeg
	ffmpeg.Stdin = ytStdOut

	defer func() {
		go func() {
			if err := ffmpeg.Wait(); err != nil {
				if exiterr, ok := err.(*exec.ExitError); ok {
					if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
						log.Printf("Exit Status: %d\n%s", status.ExitStatus(), string(exiterr.Stderr))
					}
				}
			}
			if err := ytdl.Wait(); err != nil {
				if exiterr, ok := err.(*exec.ExitError); ok {
					if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
						log.Printf("Exit Status: %d\n%s", status.ExitStatus(), string(exiterr.Stderr))
					}
				}
			}
		}()
	}()

	err = ffmpeg.Start()
	if err != nil {
		return nil, err
	}
	err = ytdl.Start()
	if err != nil {
		return nil, err
	}
	// Stream containing sound data
	stream := make(chan []int16, 320)

	// Read loop from ffmpeg
	go func() {
		// Used to close ok stream
		defer func() {
			close(stream)
		}()
		for {
			// Read data from command and pipe it
			buf := make([]int16, frameSize*channels)
			err = binary.Read(ffStdOut, binary.LittleEndian, &buf)
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return
			}
			if err != nil {
				log.Println("unknown error during buffering")
				return
			}
			select {
			case stream <- buf:
				continue
			case <-time.NewTimer(30 * time.Second).C:
				log.Println("command did receive data in time, killing")
				cancel()
			case <-ctx.Done():
				log.Println("context canceled stopping ")
				return
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
