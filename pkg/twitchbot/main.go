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

// 1: (message)
var noticeRegex *regexp.Regexp = regexp.MustCompile(`^:tmi\.twitch\.tv NOTICE #\w+ :(.+)`)

// Notice messages
const (
	messageRateNotice   = "Your message was not sent because you are sending messages too quickly."
	whisperDeniedNotice = "Your settings prevent you from sending this whisper."
)

var chatRateLimit = 30 * time.Second / 20

const maxMessageQueueLength = 10

// Bot will hit you with facts about Chuck Norris so hard your ancestors will feel it
type Bot struct {
	BotName string

	ChannelName string

	Port string

	Server string

	SecretsPath string

	WhisperAutoResponse string

	WhispersDisabled bool

	messageChannel chan string

	oAuthToken string

	connection net.Conn

	reconnectWaitTime time.Duration
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
		printpretty.Info("Connection to %s failed, trying again in %s", address, bot.reconnectWaitTime)
		time.Sleep(bot.reconnectWaitTime)
		bot.backoffConnectionRate()
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
	printpretty.Info("Enabling twitch commands")
	bot.writeToTwitch("CAP REQ", ":twitch.tv/commands")
	printpretty.Info("Enabled twitch commands")
}

func (bot *Bot) joinChannel() {
	printpretty.Info("Joining channel #%s...", bot.ChannelName)
	bot.writeToTwitch("JOIN", "#"+bot.ChannelName)
	printpretty.Info("Join attempted for channel #%s...", bot.ChannelName)
}

// Reduce the rate of reconnection attempts exponentially (first attempt is immediate)
func (bot *Bot) backoffConnectionRate() {
	if bot.reconnectWaitTime == 0 {
		bot.reconnectWaitTime = time.Second
	} else {
		bot.reconnectWaitTime *= 2
	}
}

// Validates the total message length and writes to the twitch connection
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

// Ensures all of the necessary configuration is present for the Bot
func (bot *Bot) verifyConfiguration() error {
	if bot.BotName == "" || bot.Server == "" || bot.Port == "" || bot.ChannelName == "" || bot.SecretsPath == "" {
		return errors.New("Bot is not configured")
	}

	return nil
}

// Get the OAuth token from a JSON file
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

// Add a message to the rate limited queue
func (bot *Bot) queueMessage(msg string) {
	bot.messageChannel <- msg
}

// A rate limiter for message sends
func (bot *Bot) createMessageChannel() {
	bot.messageChannel = make(chan string, maxMessageQueueLength)

	go func() {
		for message := range bot.messageChannel {
			bot.writeToTwitch("PRIVMSG", message)
			time.Sleep(chatRateLimit)
		}
	}()
}

func (bot *Bot) listenToChat() error {
	// read from connection
	tp := textproto.NewReader(bufio.NewReader(bot.connection))

	defer bot.disconnect()

	bot.createMessageChannel()

	bot.chat("Hello everyone! Type `!chucknorris` to get some Chuck Norris facts!")

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

		noticeMatches := noticeRegex.FindStringSubmatch(line)
		if noticeMatches != nil {
			noticeMessage := noticeMatches[1]

			switch noticeMessage {
			case messageRateNotice:
				printpretty.Notice(noticeMessage)
				continue
			case whisperDeniedNotice:
				printpretty.Notice(noticeMessage)
				bot.WhispersDisabled = true
			}
			continue
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

					if command == "chucknorris" {
						printpretty.Highlight("> "+fullMessage, "!"+command)
						// Don't make more requests to the API if the message queue has maxed out
						if len(bot.messageChannel) < maxMessageQueueLength {
							go bot.replyWithChuckFact(&username)
						} else {
							printpretty.Info("Too many messages queued up. Not sending request for more facts")
						}
					}
				}
			case "WHISPER":
				printpretty.Info("WHISPER received from @%s: %s", username, message)
				go bot.whisper(username, bot.WhisperAutoResponse)
			}
		}
	}
}

// Call out to the Chuck Norris API and send the returned fact to the Twitch channel
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
		return
	}

	bot.queueMessage(fmt.Sprintf("#%s :%s\r\n", bot.ChannelName, message))
}

// send a whisper to a specific user.
func (bot *Bot) whisper(username, message string) {
	if bot.WhispersDisabled {
		printpretty.Info("Bot.whisper: Whispers disabled, refusing to send whisper")
		return
	}

	if message == "" {
		printpretty.Warn("Bot.whisper: message was empty")
		return
	}

	bot.queueMessage(fmt.Sprintf("#%s :/w %s %s\r\n", username, username, message))
}

// Lets Twicth know the Bot is still active
func (bot *Bot) pong() {
	bot.writeToTwitch("PONG", ":tmi.twitch.tv")
	printpretty.Quiet("Returned PONG")
}

// Fills in any of the Bot's optional config with default values
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
		bot.reconnectWaitTime = 0
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
