module github.com/joshjennings98/discord-bot

go 1.13

replace github.com/joshjennings98/discord-bot => /home/josh/discord-bot

require (
	github.com/bwmarrin/discordgo v0.23.2
	github.com/go-ozzo/ozzo-validation v3.6.0+incompatible
	github.com/joho/godotenv v1.3.0
	github.com/mitchellh/mapstructure v1.1.2
	github.com/sirupsen/logrus v1.2.0
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.0
)
