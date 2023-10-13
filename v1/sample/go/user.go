package main

import (
	"time"

	convDB "github.com/sofmon/convention/v1/go/db"
)

type UserID string

type User struct {
	UserID    UserID
	Name      string
	CreatedAt time.Time
	CreatedBy string
	UpdatedAt time.Time
	UpdatedBy string
}

func (u User) Trail() convDB.Trail[UserID, UserID] {
	return convDB.Trail[UserID, UserID]{
		ID:        u.UserID,
		ShardKey:  u.UserID,
		CreatedAt: u.CreatedAt,
		CreatedBy: u.CreatedBy,
		UpdatedAt: u.UpdatedAt,
		UpdatedBy: u.UpdatedBy,
	}
}
