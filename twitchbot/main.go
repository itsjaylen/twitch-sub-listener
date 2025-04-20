package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
	"twitchsublistener/models"
	"twitchsublistener/utils"

	twitch "github.com/gempir/go-twitch-irc/v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	channels       = []string{"yourragegaming"}
	botChannel     = "icyjaylenn"
	oauthToken     = os.Getenv("TWITCH_OAUTH")
	POSTGRES_USER  = os.Getenv("POSTGRES_USER")
	POSTGRES_PASSWORD = os.Getenv("POSTGRES_PASSWORD")
	POSTGRES_DB    = os.Getenv("POSTGRES_DB")
	POSTGRES_PORT  string
    POSTGRES_HOST  string
	db             *gorm.DB
	msgQueue       = make(chan func(), 100)
)


func initDB() {
    var err error

	port, err := strconv.Atoi(POSTGRES_PORT)
	if err != nil {
		log.Fatalf("invalid POSTGRES_PORT: %v", err)
	}
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
		POSTGRES_HOST,POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB, port)
        for i := range 5 {
            db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
            if err == nil {
                break
            }
            log.Printf("Retrying DB connection in 3s... attempt #%d", i+1)
            time.Sleep(3 * time.Second)
        }
        if err != nil {
            log.Fatalf("failed to connect database: %v", err)
        }
        

    if err := db.AutoMigrate(&models.UnknownFollowage{}, &models.SubscriptionEvent{}); err != nil {
        log.Fatalf("auto migrate failed: %v", err)
    }
}


func main() {
    defaultPort := "5432"
    POSTGRES_PORT = os.Getenv("POSTGRES_PORT")
    if POSTGRES_PORT == "" {
        POSTGRES_PORT = defaultPort
    }
    POSTGRES_HOST = os.Getenv("POSTGRES_HOST")
    if POSTGRES_HOST == "" {
        POSTGRES_HOST = "postgres"
    }

    for _, arg := range os.Args[1:] {
        if strings.HasPrefix(arg, "--port=") {
            POSTGRES_PORT = strings.TrimPrefix(arg, "--port=")
        } else if strings.HasPrefix(arg, "--host=") {
            POSTGRES_HOST = strings.TrimPrefix(arg, "--host=")
        } else if arg == "-h" || arg == "--help" {
            fmt.Println("Usage: go run main.go [--host=HOST] [--port=PORT]")
            return
        }
    }

    initDB()

    client := twitch.NewClient("icyjaylenn", "oauth:"+oauthToken)
    go msgSender(client)

    client.OnUserNoticeMessage(func(msg twitch.UserNoticeMessage) {
        handleSub(client, msg)
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




func handleSub(client *twitch.Client, msg twitch.UserNoticeMessage) {
	switch msg.MsgID {
	case "sub", "resub":
		go processSubscription(client, msg)
	}
}

func processSubscription(client *twitch.Client, msg twitch.UserNoticeMessage) {
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
	followStr := ""
	if err != nil {
		followStr = "an unknown amount of time"
		logUnknownFollowage(user, channel)
	} else {
		followStr = formatDuration(dur)
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



func logUnknownFollowage(fromUser, toChannel string) {
	record := models.UnknownFollowage{FromUser: fromUser, ToChannel: toChannel}
	if err := db.Create(&record).Error; err != nil {
		log.Printf("[ERROR] logging unknown followage to DB: %v", err)
	}
}

func msgSender(client *twitch.Client) {
	for f := range msgQueue {
		f()
		time.Sleep(1500 * time.Millisecond)
	}
}

func formatDuration(d time.Duration) string {
	years := int(d.Hours()) / (24 * 365)
	months := (int(d.Hours()) % (24 * 365)) / (24 * 30)
	days := (int(d.Hours()) % (24 * 30)) / 24

	var parts []string
	if years > 0 {
		parts = append(parts, fmt.Sprintf("%d year(s)", years))
	}
	if months > 0 {
		parts = append(parts, fmt.Sprintf("%d month(s)", months))
	}
	if days > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%d day(s)", days))
	}
	return strings.Join(parts, ", ")
}


