package twitchbot

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"time"
)

var reconnectWaitTime = time.Duration(0)

// Bot will hit you with facts about Chuck Norris so hard your ancestors will feel it
type Bot struct {
	BotName string

	Port string

	Server string

	connection net.Conn
}

func (bot *Bot) connect() {
	var err error
	address := bot.Server + ":" + bot.Port

	fmt.Printf("Establishing connection to %s...\r\n", address)

	bot.connection, err = tls.Dial("tcp", address, nil)
	if err != nil {
		log.Printf("Connection to %s failed, trying again in %s\r\n", address, reconnectWaitTime)
		time.Sleep(reconnectWaitTime)
		backoffConnectionRate()
		bot.connect()
		return
	}

	fmt.Printf("Connected to %s\r\n", address)

	bot.disconnect()
}

func (bot *Bot) disconnect() {
	bot.connection.Close()
	fmt.Printf("Closed connection to %s\r\n", bot.Server)
}

func backoffConnectionRate() {
	if reconnectWaitTime == 0 {
		reconnectWaitTime = time.Second
	} else {
		reconnectWaitTime *= 2
	}
}

func (bot *Bot) isConfigured() bool {
	return bot.BotName != "" &&
		bot.Server != "" &&
		bot.Port != ""
}

// Start the process of connecting to Twitch...
func (bot *Bot) Start() {
	if !bot.isConfigured() {
		log.Fatal("Bot not configured")
	}

	bot.connect()
}
