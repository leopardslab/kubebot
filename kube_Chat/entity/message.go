package entity

type Message struct {
	Platform string
	Sender   string
	Content  string
	BotId    string
	Mentions []string
}

func NewMessage(platform, sender, content, botId string, mentions []string) *Message {
	return &Message{
		Platform: platform,
		Sender:   sender,
		Content:  content,
		BotId:    botId,
		Mentions: mentions,
	}
}
