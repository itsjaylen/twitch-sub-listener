package models

import "time"

// SubscriptionEvent records each sub/resub event with follow duration and sus score
type SubscriptionEvent struct {
    ID               uint          `gorm:"primaryKey"`
    User             string        `gorm:"index;not null"`
    Channel          string        `gorm:"index;not null"`
    SubType          string        `gorm:"not null"`
    CumulativeMonths int           `gorm:"default:0"`
    FollowDuration   time.Duration `gorm:"not null"`
    SusScore         string        `gorm:"not null"` // "max", "medium", "none"
    CreatedAt        time.Time     `gorm:"autoCreateTime"`
}