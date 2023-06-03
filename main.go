package main

import (
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var (
	conf    config
	discord *discordgo.Session
)

type config struct {
	Token        string        `yaml:"token"`
	UserCommands []userCommand `yaml:"user_commands"`
}

type userCommand struct {
	Command     string `yaml:"command"`
	Description string `yaml:"description"`
	Response    string `yaml:"response"`
}

func main() {
	configFile := os.Getenv("NTFYBOT_CONFIG")
	if configFile == "" {
		configFile = "/etc/ntfy/bot.yml"
	}
	f, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatal("cannot read config file, ", err)
	}
	if err := yaml.Unmarshal(f, &conf); err != nil {
		log.Fatal("cannot parse config file, ", err)
	}
	if conf.Token == "" || len(conf.UserCommands) == 0 {
		log.Fatal("token or responses not set in config file")
	}
	discord, err = discordgo.New("Bot " + conf.Token)
	if err != nil {
		log.Fatal("error creating Discord session, ", err)
	}
	defer discord.Close()
	discord.AddHandler(messageCreate)
	discord.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentMessageContent
	err = discord.Open()
	if err != nil {
		log.Fatal("error opening connection, ", err)
		return
	}
	log.Print("ntfybot running. Press Ctrl-C to exit.")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sigChan
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if err := messageCreateInternal(s, m); err != nil {
		log.Println(err.Error())
		s.ChannelMessageSend(m.ChannelID, "Oops, an error occurred")
	}
}

func messageCreateInternal(s *discordgo.Session, m *discordgo.MessageCreate) error {
	if m.Author.ID == s.State.User.ID {
		return nil
	}
	content := strings.TrimSpace(m.Content)
	if content == "!help" || strings.Contains(content, s.State.User.ID) {
		return handleHelp(s, m)
	} else if strings.HasPrefix(content, "!gh") {
		return handleGithub(s, m, strings.TrimSpace(strings.TrimPrefix(content, "!gh")))
	}
	return handleUserCommand(s, m, m.Content)
}

func handleHelp(s *discordgo.Session, m *discordgo.MessageCreate) error {
	var userCommands string
	for _, cmd := range conf.UserCommands {
		userCommands += fmt.Sprintf("`%s` - %s\n", cmd.Command, cmd.Description)
	}
	_, err := s.ChannelMessageSend(m.ChannelID, fmt.Sprintf(":wave: Hi, I'm a tiny bot that can do some ntfy specific things:\n\n`!gh <search-term>` - Search GitHub\n%s\n`!help` - Show this help", strings.TrimSpace(userCommands)))
	return err
}

func handleUserCommand(s *discordgo.Session, m *discordgo.MessageCreate, content string) error {
	for _, cmd := range conf.UserCommands {
		if cmd.Command == strings.TrimSpace(m.Content) {
			_, err := s.ChannelMessageSend(m.ChannelID, cmd.Response)
			return err
		}
	}
	return nil // This message was probably not for the bot
}

type item struct {
	HTMLURL string `json:"html_url"`
	Title   string `json:"title"`
	Number  int    `json:"number"`
}

type searchResult struct {
	TotalCount int    `json:"total_count"`
	Items      []item `json:"items"`
}

func handleGithub(s *discordgo.Session, m *discordgo.MessageCreate, searchTerm string) error {
	if searchTerm == "" {
		_, err := s.ChannelMessageSend(m.ChannelID, ":person_facepalming: You're doing it wrong! Try `!gh <search-term>`, e.g. `!gh dark mode`")
		return err
	}
	if !strings.Contains(searchTerm, "is:") {
		searchTerm += " is:open"
	}
	searchURL := fmt.Sprintf("https://api.github.com/search/issues?q=%s+repo:binwiederhier/ntfy", url.QueryEscape(searchTerm))
	resp, err := http.Get(searchURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var result searchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	var searchResults string
	for i, item := range result.Items {
		searchResults += fmt.Sprintf("**#%d**: %s - %s\n", item.Number, item.Title, item.HTMLURL)
		if i == 2 {
			break
		}
	}
	if result.TotalCount == 0 {
		searchResults = fmt.Sprintf(":astonished: Nothing found for `%s`", searchTerm)
	} else if result.TotalCount == 1 {
		searchResults = fmt.Sprintf(":tada: Exactly one GitHub issue found for `%s`:\n\n%s", searchTerm, searchResults)
	} else if result.TotalCount > 4 {
		searchResults = fmt.Sprintf(":face_with_spiral_eyes: I found %d results for `%s` (showing only 3):\n\n%s", result.TotalCount, searchTerm, searchResults)
	} else {
		searchResults = fmt.Sprintf(":sunglasses: I found %d results for `%s`:\n\n%s", result.TotalCount, searchTerm, searchResults)
	}
	_, err = s.ChannelMessageSend(m.ChannelID, searchResults)
	return err
}
