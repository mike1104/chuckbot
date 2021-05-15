package main

import (
	"github.com/mike1104/chuckbot/pkg/twitchbot"
)

func main() {
	bot := twitchbot.Bot{
		BotName: "carlosray__norris",
	}
	bot.Start()
}
