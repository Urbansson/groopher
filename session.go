package main

import (
	"container/list"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"sync"
)

// Session is a active play session in a channel
type Session struct {
	vc         *discordgo.VoiceConnection
	guildId    string
	pcmChannel chan []int16
	queue      *list.List
	playing    *Audio
	sync.Mutex
}

func (s *Session) Queue(url string) {
	s.Lock()
	defer s.Unlock()
	song := NewSong(url)
	go song.Download()
	s.queue.PushBack(song)
}

func (s *Session) Skip() {
	s.Lock()
	defer s.Unlock()
	s.playing.stop()
}

func (s *Session) Stop() {
	s.Lock()
	defer s.Unlock()
	s.queue = list.New()
	s.playing.stop()
}

func (s *Session) process(onExit func()) {
	defer s.vc.Disconnect()
	for {
		s.Lock()
		next := s.queue.Front()
		if next == nil {
			s.Unlock()
			break
		}
		s.queue.Remove(next)
		s.Unlock()

		song := next.Value.(*Song)
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
	onExit()
}

func (s *Session) Start(onExit func()) {
	go s.process(onExit)
}

func CreateSession(ds *discordgo.Session, guildID, channelID string) (*Session, error) {
	vc, err := ds.ChannelVoiceJoin(guildID, channelID, false, false)
	if err != nil {
		return nil, err
	}

	ses := &Session{
		vc:         vc,
		pcmChannel: make(chan []int16, 320),
		queue:      list.New(),
		guildId:    guildID,
	}

	return ses, nil
}
