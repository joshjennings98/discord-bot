package discord_bot

import (
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joshjennings98/discord-bot/commands"
	"github.com/joshjennings98/discord-bot/utils"
)

var (
	BotConfig  commands.BotConfiguration
	DiscordBot commands.DiscordBot
)

const (
	prefixCmd = "!bd"
)

func StartBot() (err error) {
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + BotConfig.Token)
	if err != nil {
		return fmt.Errorf("error creating Discord session: %w", err)
	}
	defer dg.Close()
	dg.AddHandler(messageCreate)
	dg.AddHandler(sendMessageInTimeInterval)
	// We only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// Set up DiscordBot
	DiscordBot.SetupDiscordBot(BotConfig, dg)

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
	return
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}
	// Check for someone saying hi
	for _, user := range m.Mentions {
		if user.ID == s.State.User.ID {
			if re := regexp.MustCompile(`^(hello|hi).*$`); re.MatchString(strings.ToLower(m.Content)) {
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Hello %s", m.Author.Mention()))
			}
			if re := regexp.MustCompile(`^(thanks|ty|thank you).*$`); re.MatchString(strings.ToLower(m.Content)) {
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("You are welcome %s", m.Author.Mention()))
			}
		}
	}

	// Check for prefix
	if strings.HasPrefix(m.Content, prefixCmd) {
		command, err := DiscordBot.ParseInput(m.Content)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error parsing command: %s", err.Error()))
			DiscordBot.Help(m.ChannelID)
			return
		}
		err = DiscordBot.ExecuteCommand(m.ChannelID, command)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error executing command: %s", err.Error()))
			DiscordBot.Help(m.ChannelID)
			return
		}
	}
}

func sendMessageInTimeInterval(s *discordgo.Session, _ *discordgo.Ready) {
	_ = commands.SetupBirthdayDatabase(BotConfig.DB)
	//DiscordBot.WishTodaysHappyBirthdays()
	ticker := time.NewTicker(1 * time.Hour)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				if utils.InTimeInterval("08:00:00", "09:00:00", time.Now()) {
					DiscordBot.WishTodaysHappyBirthdays()
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}
