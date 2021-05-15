package main

import (
	"github.com/mike1104/chuckbot/pkg/twitchbot"
)

func main() {
	bot := twitchbot.Bot{
		BotName: "carlosray__norris",
		Server:  "irc.chat.twitch.tv",
		Port:    "6697",
	}
	bot.Start()
}
