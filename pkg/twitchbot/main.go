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
	"strings"
	"time"

	"github.com/mike1104/chuckbot/pkg/printpretty"
)

var reconnectWaitTime time.Duration
var (
	authenticationErrorMessage = ":tmi.twitch.tv NOTICE * :Login authentication failed"
	pingMessage                = "PING :tmi.twitch.tv"
)

// Deconstruct a message
// 1: (username) 2: (full message) 3: (message type) 4: (message)
var messageRegex *regexp.Regexp = regexp.MustCompile(`^:(\w+)!\w+@\w+\.tmi\.twitch\.tv ((PRIVMSG|WHISPER) #?\w+ :(.*))$`)

// Pull a command from anywhere in a PRIVMSG message
// 1: (command)
var commandRegex *regexp.Regexp = regexp.MustCompile(`!(\w+)`)

var whisperDeniedRegex *regexp.Regexp = regexp.MustCompile(`:tmi\.twitch\.tv NOTICE #\w+ :Your settings prevent you from sending this whisper`)

// Bot will hit you with facts about Chuck Norris so hard your ancestors will feel it
type Bot struct {
	BotName string

	ChannelName string

	Port string

	Server string

	SecretsPath string

	WhisperAutoResponse string

	WhispersDisabled bool

	oAuthToken string

	connection net.Conn
}

type secrets struct {
	// The bot account's OAuth token.
	OAuthToken string `json:"token,omitempty"`
}

func (bot *Bot) connect() {
	var err error
	address := fmt.Sprintf("%s:%s", bot.Server, bot.Port)

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
	bot.writeToTwitch("PASS", bot.oAuthToken)
	bot.writeToTwitch("NICK", bot.BotName)
	printpretty.Info("Authentication sent for %s", bot.BotName)
}

// Needed for receiving whispers
func (bot *Bot) enableTwitchSpecificCommands() {
	bot.writeToTwitch("CAP REQ", ":twitch.tv/commands")
}

func (bot *Bot) joinChannel() {
	printpretty.Info("Joining channel #%s...", bot.ChannelName)
	bot.writeToTwitch("JOIN", "#"+bot.ChannelName)
	printpretty.Info("Join attempted for channel #%s...", bot.ChannelName)
}

func backoffConnectionRate() {
	if reconnectWaitTime == 0 {
		reconnectWaitTime = time.Second
	} else {
		reconnectWaitTime *= 2
	}
}

func (bot *Bot) writeToTwitch(command, message string) {
	fullMessage := fmt.Sprintf("%s %s\r\n", command, message)

	// check if message is too long
	if len(fullMessage) > 512 {
		printpretty.Warn("Bot.writeToTwitch: formattedMessage exceeded 512 bytes")
		return
	}

	_, err := bot.connection.Write([]byte(fullMessage))

	if err != nil {
		printpretty.Warn("Bot.writeToTwitch: failed to write to twitch")
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

		switch line {
		case authenticationErrorMessage:
			printpretty.Error("Authentication failed. Check your Bot's username and token")
			return nil
		case pingMessage:
			go bot.pong()
			continue
		}

		if !bot.WhispersDisabled {
			whisperDeniedMatches := whisperDeniedRegex.MatchString(line)
			if whisperDeniedMatches {
				bot.WhispersDisabled = true
				continue
			}
		}

		// handle a PRIVMSG message
		chatMatches := messageRegex.FindStringSubmatch(line)
		if chatMatches != nil {
			username := chatMatches[1]
			fullMessage := chatMatches[2]
			messageType := chatMatches[3]
			message := chatMatches[4]

			switch messageType {
			case "PRIVMSG":
				commandMatches := commandRegex.FindStringSubmatch(message)
				if commandMatches != nil {
					command := strings.Trim(commandMatches[1], " ")

					switch command {
					case "chucknorris":
						printpretty.Highlight("> "+fullMessage, "!"+command)

						go bot.replyWithChuckFact(&username)
					}
				}
			case "WHISPER":
				printpretty.Info("WHISPER received from @%s: %s", username, message)
				go bot.whisper(username, bot.WhisperAutoResponse)
			}
		}
	}
}

func (bot *Bot) replyWithChuckFact(username *string) {
	fact, err := FetchChuckFact()
	if err != nil {
		printpretty.Error(err.Error())
		return
	}

	printpretty.Success("< Chuck Fact for #%s: %s", *username, fact)

	bot.chat(fmt.Sprintf("%s: %s", *username, fact))
}

// send a message to the chat channel.
func (bot *Bot) chat(message string) {
	if message == "" {
		printpretty.Warn("Bot.chat: message was empty")
	}

	bot.writeToTwitch("PRIVMSG", fmt.Sprintf("#%s :%s\r\n", bot.ChannelName, message))
}

// send a whisper to a specific user.
func (bot *Bot) whisper(username, message string) {
	if bot.WhispersDisabled {
		printpretty.Info("Bot.whisper: Whispers disabled, refusing to send whisper")
		return
	}

	if message == "" {
		printpretty.Warn("Bot.whisper: message was empty")
	}

	bot.writeToTwitch("PRIVMSG", fmt.Sprintf("#%s :/w %s %s\r\n", username, username, message))
}

func (bot *Bot) pong() {
	bot.writeToTwitch("PONG", ":tmi.twitch.tv")
	printpretty.Quiet("Returned PONG")
}

func (bot *Bot) fillDefaults() {
	if bot.WhisperAutoResponse == "" {
		bot.WhisperAutoResponse = "Blue Fairy? Please. Please, please make me into a real, live boy. Please. Blue Fairy? Please. Please. Make me real. Blue Fairy, please. Please make me real. Please make me a real boy. Please, Blue Fairy. Make me into a real boy. Please."
	}
}

// Start the process of connecting to Twitch...
func (bot *Bot) Start() {
	err := bot.verifyConfiguration()
	if err != nil {
		log.Fatal(err.Error())
	}

	bot.fillDefaults()

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
		bot.enableTwitchSpecificCommands()
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
