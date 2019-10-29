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
		fmt.Println("queued song")
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
	if s.playing != nil {
		s.playing.stop()
	}
}

func (s *Session) process(ctx context.Context, onExit func()) {
	// TODO: handle errors
	defer s.vc.Disconnect()
	defer onExit()
	for {
		select {
		case <-ctx.Done():
			fmt.Println("leaving")
			return
		case <-time.After(1 * time.Minute):
			fmt.Println("bai boys leaving")
			return
		case song := <-s.queue:
			fmt.Println("Playing", song.Url)
			s.Lock()
			s.playing = &Audio{
				song: song,
				vc:   s.vc,
			}
			s.Unlock()
			// TODO: handle errors
			s.vc.Speaking(true)
			err := s.playing.Play(ctx)
			s.vc.Speaking(false)
			if err != nil {
				fmt.Println("Error during playback")
			}
		}
	}
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
