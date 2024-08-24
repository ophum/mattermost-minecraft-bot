package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"text/template"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	mcclient "github.com/ophum/mc-client"
)

type Bot struct {
	mmClient   *model.Client4
	mmWSClient *model.WebSocketClient
	mcClient   mcclient.Interface

	user    *model.User
	channel *model.Channel
}

func NewBot(ctx context.Context, config *Config) (*Bot, error) {
	mmClient := model.NewAPIv4Client(config.URL)
	mmClient.SetToken(config.Token)

	mcClient, err := mcclient.New(config.Minecraft.Host, config.Minecraft.Post, config.Minecraft.Password)
	if err != nil {
		return nil, err
	}

	bot := Bot{
		mmClient: mmClient,
		mcClient: mcClient,
	}
	user, _, err := bot.mmClient.GetUser(ctx, "me", "")
	if err != nil {
		return nil, err
	}

	bot.user = user

	channel, _, err := bot.mmClient.GetChannel(ctx, config.ChannelID, "")
	if err != nil {
		return nil, err
	}

	bot.channel = channel
	return &bot, nil
}

func (a *Bot) ListenEvent(ctx context.Context) {
	var err error
	failCount := 0
	for {
		a.mmWSClient, err = model.NewWebSocketClient4(
			config.WSURL,
			a.mmClient.AuthToken,
		)
		if err != nil {
			failCount++

			log.Printf("failed to connect mattermost websocket, count=%d", failCount)
			time.Sleep(time.Second * 10)
			continue
		}
		log.Println("Mattermost websocket connected")

		a.mmWSClient.Listen()

		for event := range a.mmWSClient.EventChannel {
			go a.HandleWebsocketEvent(ctx, event)
		}
	}
}

func (a *Bot) HandleWebsocketEvent(ctx context.Context, event *model.WebSocketEvent) {
	if event.EventType() != model.WebsocketEventPosted {
		return
	}

	post := model.Post{}
	if err := json.Unmarshal([]byte(event.GetData()["post"].(string)), &post); err != nil {
		log.Println("failed to unmarshal post:", err)
		return
	}

	// skip self post
	if post.UserId == a.user.Id {
		return
	}

	if post.ChannelId != a.channel.Id {
		return
	}

	// mention to bot
	if !strings.HasPrefix(post.Message, "@"+a.user.Username) {
		return
	}

	commands := strings.Split(post.Message, " ")

	for i, command := range commands {
		commands[i] = strings.TrimSpace(command)
	}

	// only mention
	if len(commands) == 1 {
		//TODO response help
		return
	}

	// strip mention
	commands = commands[1:]

	command := commands[0]
	args := commands[1:]

	log.Println(command, args)

	var output string
	var err error
	switch command {
	case "whitelist":
		output, err = a.commandWhitelist(ctx, args)
		if err != nil {
			log.Println(err)
			output = err.Error()
		}
	}
	if err := postMsg(ctx, a.mmClient, a.channel.Id, output); err != nil {
		log.Println("postMsg:", err)
	}
}

const whitelistHelpTemplate = "```" + `
@{{ .BotUsername }} whitelist
helpを表示します。
@{{ .BotUsername }} whitelist list
whitelistの一覧を表示します。
@{{ .BotUsername }} whitelist add [プレイヤー名]
[プレイヤー名]をwhitelistに追加します。
@{{ .BotUsername }} whitelist remove [プレイヤー名]
[プレイヤー名]をwhitelistから削除します。

example:
@{{ .BotUsername }} whitelist add hum_op
` + "\n```"

func (a *Bot) commandWhitelist(ctx context.Context, args []string) (string, error) {
	subCommand := ""
	if len(args) > 0 {
		subCommand = args[0]
		args = args[1:]
	}

	switch subCommand {
	case "":
		// TODO 毎回実行しないようにする
		t, err := template.New("").Parse(whitelistHelpTemplate)
		if err != nil {
			return "", err
		}

		b := bytes.Buffer{}
		if err := t.Execute(&b, map[string]any{
			"BotUsername": a.user.Username,
		}); err != nil {
			return "", err
		}

		return b.String(), nil
	case "list":
		list, err := a.mcClient.Whitelist().List(ctx)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("whitelistに登録されているプレイヤー一覧\n```\n%s\n```\n",
			strings.Join(list, "\n"),
		), nil
	case "add":
		if len(args) != 1 {
			return "プレイヤー名を指定してください。`@minecraft-ops whitelist add ユーザー名`", nil
		}
		player := args[0]
		if err := a.mcClient.Whitelist().Add(ctx, player); err != nil {
			return "", err
		}

		return fmt.Sprintf("プレイヤー `%s` をwhitelistに追加しました。", player), nil
	case "remove":
		if len(args) != 1 {
			return "プレイヤー名を指定してください。`@minecraft-ops whitelist remove ユーザー名`", nil
		}

		player := args[0]
		if err := a.mcClient.Whitelist().Remove(ctx, player); err != nil {
			return "", err
		}

		return fmt.Sprintf("プレイヤー `%s` をwhitelistから削除しました。", player), nil

	}
	return "不明なコマンド", nil
}
