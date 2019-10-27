package main

import (
	"encoding/binary"
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
}

func (s Song) Wait() <-chan bool {
	return s.Ready
}

func (s *Song) Download() error {
	filePath := filepath.Join(os.TempDir(), "groopher-"+uuid.New().String())
	defer os.Remove(filePath)
	if err := exec.Command("youtube-dl", "-f", "bestaudio", "--audio-format", "mp3", s.Url, "-o", filePath).Run(); err != nil {
		return err
	}
	run := exec.Command("ffmpeg", "-i", filePath, "-f", "s16le", "-ar", strconv.Itoa(frameRate), "-ac", strconv.Itoa(channels), "pipe:1")
	stdout, err := run.StdoutPipe()
	err = run.Start()

	var data [][]int16
	for {
		buf := make([]int16, frameSize*channels)
		err = binary.Read(stdout, binary.LittleEndian, &buf)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return err
		}
		data = append(data, buf)
	}
	s.Data = data
	close(s.Ready)
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
