package commands

import (
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/joshjennings98/discord-bot/utils"
)

type BotConfiguration struct {
	Token string `mapstructure:"token"`
	//DB      string `mapstructure:"birthdays_db"`
	//Channel string `mapstructure:"channel"`
	//Server  string `mapstructure:"server"`
}

func (cfg *BotConfiguration) Validate() error {
	// Validate Embedded Structs
	err := utils.ValidateEmbedded(cfg)
	if err != nil {
		return err
	}

	return validation.ValidateStruct(cfg,
		validation.Field(&cfg.Token, validation.Required),
		//validation.Field(&cfg.DB, validation.Required),
		//validation.Field(&cfg.Channel, validation.Required),
		//validation.Field(&cfg.Server, validation.Required),
	)
}

func DefaultBotConfig() *BotConfiguration {
	return &BotConfiguration{
		Token: "",
		//DB:      "",
		//Channel: "",
		//Server:  "",
	}
}
