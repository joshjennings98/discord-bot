package cmd

import (
	"context"
	"os"

	bot "github.com/joshjennings98/discord-bot/discord_bot"
	"github.com/joshjennings98/discord-bot/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	app = "bot"
	// CLI flags
	Token = "token"
)

var (
	viperSession = viper.New()
	BotConfig    bot.BotConfiguration
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
		os.Exit(1)
	}
}

func initCLI(ctx context.Context) (err error) {
	if err := utils.LoadFromViper(viperSession, app, &BotConfig, bot.DefaultBotConfig()); err != nil {
		return err
	}
	return nil
}

func init() {
	rootCmd.Flags().StringP(Token, "t", "", "Bot token")

	_ = utils.BindFlagToEnv(viperSession, app, "BOT_TOKEN", rootCmd.Flags().Lookup(Token))
}

func RunCLI(ctx context.Context) error {
	if err := initCLI(ctx); err != nil {
		log.Errorf("Failed to initialise CLI with error: %s", err)
		return err
	}

	return bot.StartBot(BotConfig)
}
