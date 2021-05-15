# Interview Assignment - Twitch Bot #

## Overview ##
---
Create an automated [Twitch](https://dev.twitch.tv/docs/irc) chat bot console application that can be run from a command line interface (CLI).


## Requirements
---
The bot application should be able to:
* Console output all interactions - legibly formatted, with timestamps.
* Connect to Twitch IRC over SSL.
* Join a channel.
* Read a channel.
* Read a private message.
* Write to a channel
* Reply to a private message.
* Avoid premature disconnections by handling Twitch courier ping / pong requests.
* Publicly reply to a user-issued string command within a channel (!YOUR_COMMAND_NAME).
    * MIKE SPECIFICALLY: Reply to the "!chucknorris" command by dynamically returning a random fact about Chuck Norris using the [Chuck Norris API](https://api.chucknorris.io).


## Caveats ##
---
* The application must be written in Go using the [standard library](https://golang.org/pkg/) - absolutely no third-party module dependencies.
* All interactions should be asynchronous.
* The application should account for Twitch API rate limits.
* The application should not exit prematurely.
