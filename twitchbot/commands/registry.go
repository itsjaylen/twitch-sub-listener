package commands

import (
	twitch "github.com/gempir/go-twitch-irc/v4"
)

type Command struct {
	Name        string
	Description string
	Handler     func(client *twitch.Client, msg twitch.PrivateMessage)
}

var Registry = make(map[string]Command)

func RegisterCommand(cmd Command) {
	Registry[cmd.Name] = cmd
}
