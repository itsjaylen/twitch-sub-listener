package models

import "time"

// UnknownFollowage logs failures to retrieve follow age
type UnknownFollowage struct {
    ID        uint      `gorm:"primaryKey"`
    FromUser  string    `gorm:"index;not null"`
    ToChannel string    `gorm:"index;not null"`
    CreatedAt time.Time `gorm:"autoCreateTime"`
}
