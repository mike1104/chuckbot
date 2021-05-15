package main

import (
	"github.com/mike1104/chuckbot/pkg/twitchbot"
)

func main() {
	bot := twitchbot.Bot{
		BotName:     "carlosray__norris",
		ChannelName: "mikkeever",
		Server:      "irc.chat.twitch.tv",
		Port:        "6697",
		SecretsPath: "./secrets.json",
	}
	bot.Start()
}
