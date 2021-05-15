package twitchbot

import "fmt"

// Bot will hit you with facts so hard about Chuck Norris your ancestors will feel it
type Bot struct {
	BotName string
}

// Start the process of connecting to Twitch...
func (bot *Bot) Start() {
	fmt.Printf("I'm a twitch bot and my name is %s\r\n", bot.BotName)
}
