package models

import "time"

type User struct {
	Id           string
	Email        string
	Password     string
	Name         string
	Category     string
	DOB          time.Time
	DOBFormatted string
	Bio          string
	Avatar       string
}
