package cmd

import (
	"context"
	"os"

	"github.com/joshjennings98/discord-bot/commands"
	bot "github.com/joshjennings98/discord-bot/discord_bot"
	"github.com/joshjennings98/discord-bot/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	app = "discord_bot"
	// CLI flags
	Token       = "token"
	BirthdaysDB = "birthdays_db"
	Channel     = "channel"
	Server      = "server"
)

var (
	viperSession = viper.New()
)

var rootCmd = &cobra.Command{
	Use:   "discord-bot",
	Short: "TODO",
	Long:  `TODO`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if err := RunCLI(ctx); err != nil {
			return err
		}
		return nil
	},
	SilenceUsage: true, // otherwise 'Usage' is printed after any error
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Errorf("Failed to start bot with error: %s", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func initCLI(ctx context.Context) (err error) {
	if err := utils.LoadFromViper(viperSession, app, &bot.BotConfig, commands.DefaultBotConfig()); err != nil {
		return err
	}
	return nil
}

func init() {
	rootCmd.Flags().StringP(Token, "t", "", "Bot token")

	_ = utils.BindFlagToEnv(viperSession, app, "DISCORD_BOT_TOKEN", rootCmd.Flags().Lookup(Token))
}

func RunCLI(ctx context.Context) error {
	if err := initCLI(ctx); err != nil {
		log.Errorf("Failed to initialise CLI with error: %s", err)
		return err
	}

	return bot.StartBot()
}
