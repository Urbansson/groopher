package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"gopkg.in/hraban/opus.v2"
	"sync"
)

const (
	frameSize int = 960
	maxBytes      = (frameSize * 2) * 2
)

type Audio struct {
	song    *Song
	playing bool
	cancel  context.CancelFunc
	vc      *discordgo.VoiceConnection
	m       sync.Mutex
}

func (a *Audio) Play(ctx context.Context) error {
	cctx, cancel := context.WithCancel(ctx)
	a.cancel = cancel
	select {
	case <-ctx.Done():
		return nil
	default:
	}
	if a.song.err != nil {
		fmt.Println("failed to download song", a.song.err)
		return errors.New("failed to access file")
	}

	enc, err := opus.NewEncoder(frameRate, channels, opus.AppAudio)
	if err != nil {
		return err
	}

	buf, err := a.song.Stream(cctx, frameSize)
	fmt.Println("starting opus stream transmission")
	for sbuf := range buf {
		select {
		case <-cctx.Done():
			return nil
		default:
		}
		data := make([]byte, maxBytes)
		n, err := enc.Encode(sbuf, data)
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
	a.cancel()
}
