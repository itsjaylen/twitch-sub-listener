package models

import "time"

// UnknownFollowage logs failures to retrieve follow age
// We removed the Count field; row count can be queried directly.
type UnknownFollowage struct {
    ID        uint      `gorm:"primaryKey"`
    FromUser  string    `gorm:"index;not null"`
    ToChannel string    `gorm:"index;not null"`
    CreatedAt time.Time `gorm:"autoCreateTime"`
}