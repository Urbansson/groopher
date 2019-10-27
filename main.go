package main

import (
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

func init() {
	flag.StringVar(&token, "t", "", "Bot Token")
	flag.Parse()
}

var token string
var sessions = make(map[string]*Session)
var lock sync.Mutex

func main() {

	if token == "" {
		fmt.Println("No token provided. Please run: airhorn -t <bot token>")
		return
	}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		return
	}

	// Register ready as a callback for the ready events.
	dg.AddHandler(ready)

	// Register messageCreate as a callback for the messageCreate events.
	dg.AddHandler(messageCreate)

	// Register guildCreate as a callback for the guildCreate events.
	dg.AddHandler(guildCreate)

	// Open the websocket and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening Discord session: ", err)
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Groopher is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// This function will be called (due to AddHandler above) when the bot receives
// the "ready" event from Discord.
func ready(s *discordgo.Session, event *discordgo.Ready) {
	fmt.Println("Connected")
	s.UpdateStatus(0, "")
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	lock.Lock()
	defer lock.Unlock()
	if s.State.Ready.User.Username == m.Author.Username {
		return
	}
	// Find the channel that the message came from.
	c, err := s.State.Channel(m.ChannelID)
	if err != nil {
		// Could not find channel.
		return
	}

	// Find the guild for that channel.
	g, err := s.State.Guild(c.GuildID)
	if err != nil {
		// Could not find guild.
		return
	}

	fmt.Printf("%20s %20s %20s %20s > %s\n", g.ID, c.ID, time.Now().Format(time.Stamp), m.Author.Username, m.Content)

	if m.Content[:1] == "!" {
		if strings.HasPrefix(m.Content, "!play") {
			activeChannel := ""
			for _, vs := range g.VoiceStates {
				if vs.UserID == m.Author.ID {
					activeChannel = vs.ChannelID
				}
			}
			if activeChannel == "" {
				fmt.Println("no channel found")
				return
			}

			if sessions[g.ID] == nil {
				sessions[g.ID], _ = CreateSession(s, g.ID, activeChannel)
				sessions[g.ID].Queue(strings.Split(m.Content, " ")[1])
				sessions[g.ID].Start(func() {
					lock.Lock()
					defer lock.Unlock()
					sessions[g.ID] = nil
				})
			} else {
				sessions[g.ID].Queue(strings.Split(m.Content, " ")[1])
			}
		} else if strings.HasPrefix(m.Content, "!stop") {
			if sessions[g.ID] != nil {
				sessions[g.ID].Stop()
			}
		} else if strings.HasPrefix(m.Content, "!skip") {
			if sessions[g.ID] != nil {
				sessions[g.ID].Skip()
			}
		} else if strings.HasPrefix(m.Content, "!help") {
			s.ChannelMessageSend(m.ChannelID, `**!play** <youtube link or query> - Search/Play Youtube link, queues up if another track is playing
**!skip** - Skip current playing track
**!stop** - Stops tracks and clears queue`)
		}
	}
}

// This function will be called (due to AddHandler above) every time a new
// guild is joined.
func guildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
	if event.Guild.Unavailable {
		return
	}
}
