Chuckbot
========
A Twitch bot that delivers hard facts about Chuck Norris.

Did you know Chuck Norris can unscramble eggs? You do now.

How To
------
Navigate to cmd/chuckbot
```
cd ./cmd/chuckbot
```

Create a "secrets.json" file in this directory with these contents where `<OAuthToken>` is in the format `oauth:xxxxxxxxxxxxxxxx`. The token can be generated from here https://twitchapps.com/tmi/.
```
{
    "token": "<OAuthToken>"
}
```




Build the application
```
go build main.go
```

Run the application
```
./chuckbot
```

