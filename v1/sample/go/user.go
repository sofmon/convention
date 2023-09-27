package main

import (
	"time"
)

type UserID string

type User struct {
	UserID UserID
}

func (u User) ID() UserID {
	return u.UserID
}

func (u User) ShardKey() UserID {
	return u.UserID
}

func (u User) CreatedAt() time.Time {
	return time.Now().UTC()
}

func (u User) CreatedBy() string {
	return ""
}

func (u User) UpdatedAt() time.Time {
	return time.Now().UTC()
}

func (u User) UpdatedBy() string {
	return ""
}
