package services

type Discord interface {
	SendMessage(channelID string, message string) error
}

type DiscordImpl struct {
}

func NewDiscord() *DiscordImpl {
	return &DiscordImpl{}
}

func (d *DiscordImpl) SendMessage(channelID string, message string) error {
	return nil
}
