package commands

import (
	"time"

	"github.com/bwmarrin/discordgo"
	validation "github.com/go-ozzo/ozzo-validation"
)

type BotConfiguration struct {
	Token      string `mapstructure:"token"`
	MongoDBURI string `mapstructure:"mongodb_uri"`
}

func (cfg *BotConfiguration) Validate() error {
	return validation.ValidateStruct(cfg,
		validation.Field(&cfg.Token, validation.Required),
		validation.Field(&cfg.MongoDBURI, validation.Required),
	)
}

func DefaultBotConfig() *BotConfiguration {
	return &BotConfiguration{
		Token:      "",
		MongoDBURI: "",
	}
}

type Birthday struct {
	ID   string
	Date time.Time
}

type Birthdays []Birthday

func (b Birthdays) Len() int {
	return len(b)
}

func (a Birthdays) Less(i, j int) (b bool) {
	return a[i].Date.YearDay() < a[j].Date.YearDay() // YearDay :)
}

func (a Birthdays) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

type Command struct {
	Action   string
	ID       string
	DateTime string
	Channel  string
	Server   string
	Database string
}

type DiscordBot struct {
	session *discordgo.Session
}
