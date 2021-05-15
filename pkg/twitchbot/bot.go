package twitchbot

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/textproto"
	"regexp"
	"time"
)

var reconnectWaitTime time.Duration
var authenticationErrorMessage = ":tmi.twitch.tv NOTICE * :Login authentication failed"

// Deconstruct a message
// 1: (userName) 3: (message)
var messageRegex *regexp.Regexp = regexp.MustCompile(`^:(\w+)!\w+@\w+\.tmi\.twitch\.tv PRIVMSG #?\w+ :(.*)$`)

// Bot will hit you with facts about Chuck Norris so hard your ancestors will feel it
type Bot struct {
	BotName string

	ChannelName string

	Port string

	Server string

	SecretsPath string

	oAuthToken string

	connection net.Conn
}

type secrets struct {
	// The bot account's OAuth token.
	OAuthToken string `json:"token,omitempty"`
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

}

func (bot *Bot) disconnect() {
	fmt.Printf("Disconnecting from %s\r\n", bot.Server)
	bot.connection.Close()
	fmt.Printf("Closed connection to %s\r\n", bot.Server)
}

func (bot *Bot) authenticate() {
	fmt.Printf("Authenticating %s...\r\n", bot.BotName)
	bot.connection.Write([]byte("PASS " + bot.oAuthToken + "\r\n"))
	bot.connection.Write([]byte("NICK " + bot.BotName + "\r\n"))
	fmt.Printf("Authentication sent for %s\r\n", bot.BotName)
}

func (bot *Bot) joinChannel() {
	fmt.Printf("Joining channel #%s...\r\n", bot.ChannelName)
	bot.connection.Write([]byte("JOIN #" + bot.ChannelName + "\r\n"))
	fmt.Printf("Join attempted for channel #%s...\r\n", bot.ChannelName)
}

func backoffConnectionRate() {
	if reconnectWaitTime == 0 {
		reconnectWaitTime = time.Second
	} else {
		reconnectWaitTime *= 2
	}
}

func (bot *Bot) verifyConfiguration() error {
	if bot.BotName == "" || bot.Server == "" || bot.Port == "" || bot.ChannelName == "" || bot.SecretsPath == "" {
		return errors.New("Bot is not configured")
	}

	return nil
}

func (bot *Bot) getOAuthToken() error {
	data, err := ioutil.ReadFile(bot.SecretsPath)
	if err != nil {
		return err
	}

	var str secrets
	err = json.Unmarshal(data, &str)

	if err != nil {
		return err
	}

	if str.OAuthToken == "" {
		return errors.New("Bot.getOAuthToken: 'token' is empty")
	}

	bot.oAuthToken = str.OAuthToken

	return nil
}

func (bot *Bot) listenToChat() error {
	// read from connection
	tp := textproto.NewReader(bufio.NewReader(bot.connection))

	defer bot.disconnect()

	// listen for chat messages
	for {
		line, err := tp.ReadLine()

		fmt.Println(line)

		if err != nil {
			return errors.New("Bot.listenToChat: Failed to read line from channel")
		}

		if line == authenticationErrorMessage {
			log.Fatal("Authentication failed. Check your Bot's username and token")
		}

		// handle a PRIVMSG message
		chatMatches := messageRegex.FindStringSubmatch(line)
		if chatMatches != nil {
			username := chatMatches[1]
			message := chatMatches[2]

			fmt.Printf("Message from @%s: %s\r\n", username, message)
		}

	}
}

// Start the process of connecting to Twitch...
func (bot *Bot) Start() {
	err := bot.verifyConfiguration()
	if err != nil {
		log.Fatal(err.Error())
	}

	err = bot.getOAuthToken()
	if err != nil {
		log.Println(err.Error())
		log.Fatalf("Could not find 'token' in %s", bot.SecretsPath)
	}

	for {
		reconnectWaitTime = 0
		bot.connect()
		bot.authenticate()
		bot.joinChannel()

		err = bot.listenToChat()
		if err != nil {
			fmt.Println(err.Error())
			bot.disconnect()
		} else {
			fmt.Println("Nothing more for us to do here")
		}
	}
}
