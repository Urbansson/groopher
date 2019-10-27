package main

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"sync"
	"time"
)

// Session is a active play session in a channel
type Session struct {
	vc         *discordgo.VoiceConnection
	guildId    string
	pcmChannel chan []int16
	queue      chan *Song
	playing    *Audio
	cancel     context.CancelFunc
	sync.Mutex
}

func (s *Session) Queue(url string) {
	song := NewSong(url)

	select {
	case s.queue <- song:
		go song.Download()
	default:
		fmt.Println("queue is full ignoring message")
		// message dropped
	}

}

func (s *Session) Skip() {
	s.Lock()
	defer s.Unlock()
	s.playing.stop()
}

func (s *Session) Stop() {
	s.Lock()
	defer s.Unlock()
	for len(s.queue) > 0 {
		<-s.queue
	}
	s.cancel()
	s.playing.stop()
}

func (s *Session) process(ctx context.Context, onExit func()) {
	defer s.vc.Disconnect()
	for {
		select {
		case <-ctx.Done():
			fmt.Println("leaving")
			break
		case <-time.After(1 * time.Minute):
			fmt.Println("bai boys leaving")
			break
		case song := <-s.queue:
			fmt.Println("Playing", song.Url)
			s.Lock()
			s.playing = &Audio{
				song:  song,
				close: make(chan int),
				vc:    s.vc,
			}
			s.Unlock()
			err := s.playing.Play()
			if err != nil {
				fmt.Println("Error during playback")
			}
		}
	}
	onExit()
}

func CreateSession(ds *discordgo.Session, guildID, channelID string, onExit func()) (*Session, error) {
	vc, err := ds.ChannelVoiceJoin(guildID, channelID, false, false)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	ses := &Session{
		vc:         vc,
		pcmChannel: make(chan []int16, 320),
		queue:      make(chan *Song, 50),
		guildId:    guildID,
		cancel:     cancel,
	}

	go ses.process(ctx, onExit)

	return ses, nil
}
