package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/ophum/mc-client/rcon"
	"gopkg.in/yaml.v3"
)

type ConfigMinecraft struct {
	Host       string `yaml:"host"`
	Post       int    `yaml:"port"`
	Password   string `yaml:"password"`
	ServerType string `yaml:"serverType"`
}

func (c *ConfigMinecraft) GetServerType() rcon.ServerType {
	switch c.ServerType {
	case "vanilla":
		return rcon.ServerTypeVanilla
	case "spigot":
		return rcon.ServerTypeSpigot
	default:
		return rcon.ServerTypeVanilla
	}
}

type Config struct {
	URL       string `yaml:"url"`
	WSURL     string `yaml:"wsURL"`
	ChannelID string `yaml:"channelID"`
	Token     string `yaml:"token"`

	Minecraft *ConfigMinecraft `yaml:"minecraft"`
}

var config Config

func init() {
	configPath := flag.String("config", "config.yaml", "-config config.yaml")
	flag.Parse()

	if !flag.Parsed() {
		log.Fatal("not parsed flags")
	}

	f, err := os.Open(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if err := yaml.NewDecoder(f).Decode(&config); err != nil {
		log.Fatal(err)
	}
}

func main() {
	ctx := context.Background()
	bot, err := NewBot(ctx, &config)
	if err != nil {
		log.Fatal(err)
	}

	go bot.ScrapePlayers(ctx)

	bot.ListenEvent(ctx)
}

func postMsg(ctx context.Context, client *model.Client4, channelID, msg string) error {
	resPost := &model.Post{}
	resPost.ChannelId = channelID
	resPost.Message = msg

	//if post.RootId != "" {
	//	resPost.RootId = post.RootId
	//} else {
	//	resPost.RootId = post.Id
	//}
	if _, _, err := client.CreatePost(ctx, resPost); err != nil {
		return err
	}
	return nil
}
