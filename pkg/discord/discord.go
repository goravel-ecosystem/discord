package discord

import (
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
)

var (
	session *discordgo.Session
	once    sync.Once
)

// Init function must be called before using GetInstance.
func Init(botToken string) (err error) {
	once.Do(func() {
		session, err = discordgo.New("Bot " + botToken)
		if err != nil {
			return
		}
	})
	return err
}

// GetInstance returns the singleton instance of the Discord session.
// It panics if Init was not called or if initialization failed.
func GetInstance() *discordgo.Session {
	if session == nil {
		log.Fatal().Msg("Discord session not initialized. Call Init() first.")
	}
	return session
}
