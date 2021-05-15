package twitchbot

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

type chuckFact struct {
	Value string `json:"value,omitempty"`
}

// FetchChuckFact requests a "joke" from api.chucknorris.io
func FetchChuckFact() (string, error) {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get("https://api.chucknorris.io/jokes/random")
	if err != nil {
		return "", errors.New("FetchChuckFact: " + err.Error())
	}

	defer resp.Body.Close()
	fact := chuckFact{}
	err = json.NewDecoder(resp.Body).Decode(&fact)

	if err != nil {
		return "", errors.New("FetchChuckFact: " + err.Error())
	}

	return fact.Value, nil
}
