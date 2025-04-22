package commands

import (
	"twitchsublistener/models"

	"gorm.io/gorm"
)

func CheckUserSuspicion(db *gorm.DB, username string) (models.SubscriptionEvent, string, error) {
	var sub models.SubscriptionEvent
	var unknownFollow models.UnknownFollowage

	err := db.Where("user = ? AND channel = ?", username, "yourragegaming").
		Order("created_at desc").
		First(&sub).Error
	if err != nil {
		return sub, "no_data", err
	}

	err = db.Where("from_user = ? AND to_channel = ?", username, "yourragegaming").
		First(&unknownFollow).Error
	isUnknown := err == nil

	sus := sub.SusScore
	if sub.SubType == "Prime" && (sub.FollowDuration == 0 || isUnknown) {
		sus = "max"
	}

	return sub, sus, nil
}


