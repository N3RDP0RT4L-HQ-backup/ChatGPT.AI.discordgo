package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type Config struct {
	DiscordToken string `json:"token"`
}

type APIRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type APIResponse struct {
	Data string `json:"data"`
}

func main() {
	config := loadConfig("config.json")

	discord, err := discordgo.New("Bot " + config.DiscordToken)
	if err != nil {
		log.Fatalf("error creating Discord session: %s", err)
	}

	// add onMessage function as a handler for Discord events
	discord.AddHandler(onMessage)

	// open a connection to Discord
	err = discord.Open()
	if err != nil {
		log.Fatalf("error opening connection to Discord: %s", err)
	}

	defer discord.Close()

	fmt.Println("Bot is running. Press CTRL-C to quit.")
	select {}
}

func onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// ignore messages sent by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// check if message contains a mention of the bot
	if strings.Contains(m.Content, "<@"+s.State.User.ID+">") {
		// extract the prompt message from the user's message
		prompt := strings.ReplaceAll(m.Content, "<@"+s.State.User.ID+">", "")
		prompt = strings.TrimSpace(prompt)

		// send typing status to channel
		s.ChannelTyping(m.ChannelID)

		// send the prompt message to the AI API to generate a response
		data := APIRequest{
			Model:  "openai:gpt-3.5-turbo",
			Prompt: prompt,
		}
		resp, err := postData("http://127.0.0.1:8080/api", data)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("An error occurred while processing your request: %s", err))
			return
		}

		var output APIResponse
		err = json.Unmarshal(resp, &output)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("An error occurred while processing your request: %s", err))
			return
		}

		// send the API response to the channel in chunks of up to 2000 characters
		for _, chunk := range splitString(output.Data, 2000) {
			s.ChannelMessageSend(m.ChannelID, chunk)
		}
	}
}

func loadConfig(filename string) Config {
	// read Discord token from config file
	config := Config{}
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("failed to read config file: %s", err)
	}
	err = json.Unmarshal(file, &config)
	if err != nil {
		log.Fatalf("failed to decode config file: %s", err)
	}
	return config
}

func postData(endpoint string, data APIRequest) ([]byte, error) {
	// send a POST request to the specified endpoint with the request data
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func splitString(str string, chunkSize int) []string {
	// split a string into chunks of up to the specified size
	var chunks []string
	for i := 0; i < len(str); i += chunkSize {
		if i+chunkSize > len(str) {
			chunks = append(chunks, str[i:])
		} else {
			chunks = append(chunks, str[i:i+chunkSize])
		}
	}
	return chunks
}
