package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"twitchsublistener/commands"
	"twitchsublistener/models"
	"twitchsublistener/utils"

	twitch "github.com/gempir/go-twitch-irc/v4"
)

var (
	channels   = []string{"yourragegaming"}
	botChannel = "icyjaylenn"
	oauthToken = os.Getenv("TWITCH_OAUTH")
	msgQueue   = make(chan func(), 100)
	startTime  = time.Now()
)
func handleCommand(client *twitch.Client, msg twitch.PrivateMessage) {
	fields := strings.Fields(msg.Message)
	if len(fields) == 0 {
		return
	}

	cmdName := fields[0]
	cmd, exists := commands.Registry[cmdName]
	if exists {
		cmd.Handler(client, msg)
	}
}

func main() {
	utils.InitEnvDefaults()
	utils.ParseArgs(&utils.POSTGRES_HOST, &utils.POSTGRES_PORT)

	db := utils.InitDB()
	utils.AutoMigrateDB(db)

	commands.InitCommands(db)

	client := twitch.NewClient(botChannel, "oauth:"+oauthToken)
	go msgSender(client)

	client.OnUserNoticeMessage(func(msg twitch.UserNoticeMessage) {
		handleSub(db, client, msg)
	})

	client.OnPrivateMessage(func(msg twitch.PrivateMessage) {
		handleCommand(client, msg)
	})

	client.OnConnect(func() {
		log.Println("[INFO] Bot connected successfully")
	})

	log.Println("[INFO] Joining channels...")
	client.Join(append(channels, botChannel)...)

	utils.RetryForever(func() error {
		return client.Connect()
	})
}


func handleSub(db *utils.DB, client *twitch.Client, msg twitch.UserNoticeMessage) {
	switch msg.MsgID {
	case "sub", "resub":
		go processSubscription(db, client, msg)
	}
}

func processSubscription(db *utils.DB, client *twitch.Client, msg twitch.UserNoticeMessage) {
	user := msg.User.Name
	channel := msg.Channel

	subPlan := msg.MsgParams["msg-param-sub-plan"]
	subType := map[string]string{
		"Prime": "Twitch Prime",
		"1000":  "Tier 1",
		"2000":  "Tier 2",
		"3000":  "Tier 3",
	}[subPlan]
	if subType == "" {
		subType = "Unknown Tier"
	}

	dur, err := utils.GetFollowDuration(user, channel)
	followStr := "an unknown amount of time"
	if err != nil {
		utils.LogUnknownFollowage(db, user, channel)
	} else {
		followStr = utils.FormatDuration(dur)
	}

	susScore := utils.ComputeSusScore(subType, dur, err)

	months := 0
	if msg.MsgID == "resub" {
		monthsStr := msg.MsgParams["msg-param-cumulative-months"]
		months, _ = strconv.Atoi(monthsStr)
	}

	event := models.SubscriptionEvent{
		User:             user,
		Channel:          channel,
		SubType:          subType,
		CumulativeMonths: months,
		FollowDuration:   dur,
		SusScore:         susScore,
	}
	if err := db.Create(&event).Error; err != nil {
		log.Printf("[ERROR] saving subscription event: %v", err)
	}

	var chatMsg string
	if msg.MsgID == "sub" {
		chatMsg = fmt.Sprintf("/me %s subscribed to %s with a %s subscription. They've been following for %s (sus: %s).",
			user, channel, subType, followStr, susScore)
	} else {
		chatMsg = fmt.Sprintf("/me %s resubscribed to %s (%d months) with a %s subscription. They've been following for %s (sus: %s).",
			user, channel, months, subType, followStr, susScore)
	}

	msgQueue <- func() { client.Say(botChannel, chatMsg) }
}

func msgSender(client *twitch.Client) {
	for f := range msgQueue {
		f()
		time.Sleep(1500 * time.Millisecond)
	}
}
