package main

import (
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"gopkg.in/hraban/opus.v2"
	"sync"
)

const (
	channels  int = 2
	frameRate int = 48000
	frameSize int = 960
	maxBytes  int = (frameSize * 2) * 2
)

type Audio struct {
	song    *Song
	playing bool
	close   chan int
	vc      *discordgo.VoiceConnection
	m       sync.Mutex
}

func (a *Audio) Play() error {
	defer func() {
		a.song.Cleanup()
	}()
	fmt.Println("waiting for media to be ready:", a.song.Url)
	select {
	case <-a.close:
		return nil
	case <-a.song.Wait():
	}
	fmt.Println("media ready playing: ", a.song.Url)

	if a.song.err != nil {
		fmt.Println("failed to download song", a.song.err)
		return errors.New("failed to access file")
	}

	enc, err := opus.NewEncoder(frameRate, channels, opus.AppAudio)
	if err != nil {
		return err
	}

	for _, buf := range a.song.Data {
		select {
		case <-a.close:
			return nil
		default:
		}

		data := make([]byte, maxBytes)
		n, err := enc.Encode(buf, data)
		if err != nil {
			return err
		}

		if a.vc.Ready == false || a.vc.OpusSend == nil {
			return err
		}
		a.vc.OpusSend <- data[:n]
	}
	return nil
}

func (a *Audio) stop() {
	close(a.close)
}
