package commands

import (
	"twitchsublistener/models"

	"gorm.io/gorm"
)

func CheckUserSuspicion(db *gorm.DB, username string) (string, error) {
    var sub models.SubscriptionEvent
    var unknownFollow models.UnknownFollowage

    err := db.Where("user = ? AND channel = ?", username, "yourragegaming").
        Order("created_at desc").
        First(&sub).Error
    if err != nil {
        return "no_data", err
    }

    err = db.Where("from_user = ? AND to_channel = ?", username, "yourragegaming").
        First(&unknownFollow).Error
    isUnknown := err == nil

    // Determine suspicion
    if sub.SubType == "Prime" && (sub.FollowDuration == 0 || isUnknown) {
        return "max", nil
    }

    return sub.SusScore, nil
}

