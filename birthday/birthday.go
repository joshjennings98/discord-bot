package commands

import (
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/joshjennings98/discord-bot/utils"
	log "github.com/sirupsen/logrus"
)

type BotConfiguration struct {
	Token string `mapstructure:"token"`
}

func (cfg *BotConfiguration) Validate() error {
	// Validate Embedded Structs
	err := utils.ValidateEmbedded(cfg)
	if err != nil {
		return err
	}

	return validation.ValidateStruct(cfg,
		validation.Field(&cfg.Token, validation.Required),
	)
}

func DefaultBotConfig() *BotConfiguration {
	return &BotConfiguration{
		Token: "",
	}
}

type Birthday struct {
	ID   string
	Date string
}

type Birthdays []Birthday

func (b Birthdays) Len() int {
	return len(b)
}

func (a Birthdays) Less(i, j int) (b bool) {
	ai, err := strconv.ParseInt(a[i].Date, 10, 64)
	if err != nil {
		log.Errorf("Failed to parse unix time %s", ai)
	}
	aj, err := strconv.ParseInt(a[j].Date, 10, 64)
	if err != nil {
		log.Errorf("Failed to parse unix time %s", aj)
	}
	return time.Unix(ai, 0).YearDay() < time.Unix(aj, 0).YearDay() // YearDay :)
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
