package discord_bot

import (
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/joshjennings98/discord-bot/utils"
)

type BotConfiguration struct {
	Token string `mapstructure:"token"`
}

func DefaultBotConfig() *BotConfiguration {
	return &BotConfiguration{
		Token: "",
	}
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

func StartBot(cfg BotConfiguration) (err error) {

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return fmt.Errorf("error creating Discord session: %w", err)
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		return fmt.Errorf("error opening connection: %w", err)
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
	return nil
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	for _, user := range m.Mentions {
		if user.ID == s.State.User.ID {
			re := regexp.MustCompile(`[hello|hi] .*`)
			if re.MatchString(strings.ToLower(m.Content)) {
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Hello %s", m.Author.Mention()))
			}
		}
	}

}
