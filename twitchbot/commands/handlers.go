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
			fmt.Printf("[DEBUG] Raw message: %q\n", msg.Message)
	
			targetUser := msg.User.Name
	
			rawArgs := strings.TrimSpace(msg.Message[len("!check"):])
			fmt.Printf("[DEBUG] rawArgs after trimming: %q\n", rawArgs)
	
			if rawArgs != "" {
				targetUser = strings.ToLower(strings.Fields(rawArgs)[0])
			}
	
			fmt.Printf("[DEBUG] targetUser: %q\n", targetUser)
	
			sub, sus, err := CheckUserSuspicion(db, targetUser)
			if err != nil {
				client.Say(msg.Channel, fmt.Sprintf("Could not find suspicion data for %s.", targetUser))
				return
			}
	
			formattedFollow := utils.FormatDuration(sub.FollowDuration)
	
			client.Say(msg.Channel, fmt.Sprintf(
				"Suspicion for %s: %s | SubType: %s | Months: %d | Followed: %s ago",
				targetUser,
				sus,
				sub.SubType,
				sub.CumulativeMonths,
				formattedFollow,
			))
		},
	})
	
	
	
}
