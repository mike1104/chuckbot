package twitchbot

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"net/textproto"
	"regexp"
	"strings"
	"time"

	"github.com/mike1104/chuckbot/pkg/printpretty"
)

var reconnectWaitTime time.Duration
var authenticationErrorMessage = ":tmi.twitch.tv NOTICE * :Login authentication failed"

// Deconstruct a message
// 1: (username) 2: (full message) 3: (message)
var messageRegex *regexp.Regexp = regexp.MustCompile(`^:(\w+)!\w+@\w+\.tmi\.twitch\.tv (PRIVMSG #?\w+ :(.*))$`)

// Pull a command from anywhere in a PRIVMSG message
// 1: (command)
var commandRegex *regexp.Regexp = regexp.MustCompile(`!(\w+)`)

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

	printpretty.Info("Establishing connection to %s...", address)

	bot.connection, err = tls.Dial("tcp", address, nil)
	if err != nil {
		printpretty.Info("Connection to %s failed, trying again in %s", address, reconnectWaitTime)
		time.Sleep(reconnectWaitTime)
		backoffConnectionRate()
		bot.connect()
		return
	}

	printpretty.Info("Connected to %s", address)

}

func (bot *Bot) disconnect() {
	printpretty.Info("Disconnecting from %s", bot.Server)
	bot.connection.Close()
	printpretty.Info("Closed connection to %s", bot.Server)
}

func (bot *Bot) authenticate() {
	printpretty.Info("Authenticating %s...", bot.BotName)
	bot.connection.Write([]byte("PASS " + bot.oAuthToken + "\r\n"))
	bot.connection.Write([]byte("NICK " + bot.BotName + "\r\n"))
	printpretty.Info("Authentication sent for %s", bot.BotName)
}

func (bot *Bot) joinChannel() {
	printpretty.Info("Joining channel #%s...", bot.ChannelName)
	bot.connection.Write([]byte("JOIN #" + bot.ChannelName + "\r\n"))
	printpretty.Info("Join attempted for channel #%s...", bot.ChannelName)
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

		// Quietly log everything from Twitch
		printpretty.Quiet(line)

		if err != nil {
			return errors.New("Bot.listenToChat: Failed to read line from channel")
		}

		if line == authenticationErrorMessage {
			printpretty.Error("Authentication failed. Check your Bot's username and token")
			return nil
		}

		// handle a PRIVMSG message
		chatMatches := messageRegex.FindStringSubmatch(line)
		if chatMatches != nil {
			username := chatMatches[1]
			fullMessage := chatMatches[2]
			message := chatMatches[3]

			commandMatches := commandRegex.FindStringSubmatch(message)
			if commandMatches != nil {
				command := strings.Trim(commandMatches[1], " ")

				switch command {
				case "chucknorris":
					printpretty.Highlight("> "+fullMessage, "!"+command)

					continue
				}
			} else {

			}

			printpretty.Info("Message from @%s: %s", username, message)

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
		printpretty.Error(err.Error())
		printpretty.Error("Could not find 'token' in %s", bot.SecretsPath)
		return
	}

	for {
		reconnectWaitTime = 0
		bot.connect()
		bot.authenticate()
		bot.joinChannel()

		err = bot.listenToChat()
		if err != nil {
			printpretty.Warn(err.Error())
		} else {
			// Nothing more can be done here but break the loop and exit.
			return
		}
	}
}
