package commands

import (
	"fmt"
	"strings"
	"twitchsublistener/utils"

	"github.com/gempir/go-twitch-irc/v4"
	"gorm.io/gorm"
)

func InitCommands(db *gorm.DB) {
	RegisterCommand(Command{
		Name:        "!test",
		Description: "Test the bot is alive",
		Handler: func(client *twitch.Client, msg twitch.PrivateMessage) {
			client.Say(msg.Channel, "1")
		},
	})

	RegisterCommand(Command{
		Name:        "!uptime",
		Description: "Shows how long the bot has been running.",
		Handler: func(client *twitch.Client, msg twitch.PrivateMessage) {
			response := utils.GetUptime()
			client.Say(msg.Channel, response)
		},
	})

	RegisterCommand(Command{
		Name:        "!check",
		Description: "Check suspicion for a user in this channel",
		Handler: func(client *twitch.Client, msg twitch.PrivateMessage) {
			targetUser := msg.User.Name
	
			parts := strings.Fields(msg.Message)
			if len(parts) > 1 {
				targetUser = strings.ToLower(parts[1]) 
			}
	
			sus, err := CheckUserSuspicion(db, targetUser, msg.Channel)
			if err != nil {
				client.Say(msg.Channel, fmt.Sprintf("Could not find suspicion data for %s.", targetUser))
				return
			}
	
			client.Say(msg.Channel, fmt.Sprintf("Suspicion level for %s: %s", targetUser, sus))
		},
	})
	
}
