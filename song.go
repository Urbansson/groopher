package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/google/uuid"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

const tempDir = "./temp"

type Song struct {
	Url   string
	Ready chan bool
	Data  [][]int16
	err   error
}

func (s Song) Wait() <-chan bool {
	return s.Ready
}

func (s *Song) Download() error {
	filePath := filepath.Join(os.TempDir(), "groopher-"+uuid.New().String())
	defer os.Remove(filePath)
	defer close(s.Ready)

	cmd := exec.Command("youtube-dl", "-f", "bestaudio", "--audio-format", "mp3", s.Url, "-o", filePath)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {

		fmt.Println("failed to download")
		s.err = err
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())

		return err
	}

	run := exec.Command("ffmpeg", "-i", filePath, "-f", "s16le", "-ar", strconv.Itoa(frameRate), "-ac", strconv.Itoa(channels), "pipe:1")
	stdout, err := run.StdoutPipe()
	if err != nil {
		fmt.Println("failed to connect pipe")
		s.err = err
		return err
	}
	err = run.Start()
	if err != nil {
		fmt.Println("failed to start ffmpeg")
		s.err = err
		return err
	}

	var data [][]int16
	for {
		buf := make([]int16, frameSize*channels)
		err = binary.Read(stdout, binary.LittleEndian, &buf)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			fmt.Println("unknown error during buffering")
			s.err = err
			return err
		}
		data = append(data, buf)
	}
	s.Data = data
	return nil
}

func (s *Song) Cleanup() {
	s.Data = make([][]int16, 0)
}

func NewSong(url string) *Song {
	return &Song{
		Url:   url,
		Ready: make(chan bool),
	}
}
