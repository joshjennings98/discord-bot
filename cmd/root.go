package cmd

import (
	"context"
	"os"

	commands "github.com/joshjennings98/discord-bot/birthday"
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
	DatabaseDir = "database_dir"
)

var (
	viperSession = viper.New()
)

var rootCmd = &cobra.Command{
	Use:   "discord-bot",
	Short: "Discord birthday bot.",
	Long: `This is the birthday discord bot (BirthdayBot3000).

Environment variables can be used instead of cli arguments. CLI arguments will take precedence.

Environment Variables:
	DISCORD_BOT_TOKEN 	  string 	Bot token
	DISCORD_BOT_DATABASES string 	Database directory
`,
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
		log.Errorf("Failed to start birthday bot with error: %s", err)
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
	rootCmd.Flags().StringP(DatabaseDir, "d", "", "Database directory")

	_ = utils.BindFlagToEnv(viperSession, app, "DISCORD_BOT_TOKEN", rootCmd.Flags().Lookup(Token))
	_ = utils.BindFlagToEnv(viperSession, app, "DISCORD_BOT_DATABASES", rootCmd.Flags().Lookup(DatabaseDir))
}

func RunCLI(ctx context.Context) error {
	if err := initCLI(ctx); err != nil {
		return err
	}

	return bot.StartBot()
}
