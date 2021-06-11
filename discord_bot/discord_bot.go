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
	BotConfig commands.BotConfiguration
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
			if re := regexp.MustCompile(`[hello|hi] .*`); re.MatchString(strings.ToLower(m.Content)) {
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Hello %s", m.Author.Mention()))
			}
		}
	}

	// Check for prefix
	if strings.HasPrefix(m.Content, prefixCmd) {
		command := utils.SplitCommand(m.Content)
		if len(command) == 4 {
			if command[1] == "add" {
				if isValidUser, id := utils.IsUser(command[2], s, BotConfig.Server); isValidUser && utils.IsValidDate(command[3]) {
					date, err := time.Parse("02/01/06 03:04:05 PM", command[3]+"/00 00:00:00 AM")
					if err != nil {
						s.ChannelMessageSend(m.ChannelID, "Date must be of the format dd/mm")
						return
					}
					commands.AddBirthdayToDatabase(BotConfig.DB, id, date)
					s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Successfully added birthday for <@!%s> on %s", id, command[3]))
				} else if !isValidUser {
					s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("@%s is not a user on this server.", id))
				} else if !utils.IsValidDate(command[3]) {
					s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Invalid date %s", command[3]))
				}
			} else {
				s.ChannelMessageSend(m.ChannelID, "**BirthdayBot Usage:**\n`!bd add <user> <dd/mm>` - add a new birthday to the database\n`!bd next` - see who is having their birthday next\n`!bd today` - check who is having their birthday today\n`!bd help` - see the abysmal help")
			}
		} else if len(command) == 2 {
			if command[1] == "today" {
				commands.CheckTodaysBirthdays(BotConfig.DB, s, BotConfig)
			} else if command[1] == "next" {
				err := commands.NextBirthday(BotConfig.DB, s, BotConfig)
				if err != nil {
					s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("error getting next birthday %s", err.Error()))
					return
				}
			} else if command[1] == "help" {
				s.ChannelMessageSend(m.ChannelID, "**BirthdayBot Usage:**\n`!bd add <user> <dd/mm>` - add a new birthday to the database\n`!bd next` - see who is having their birthday next\n`!bd today` - check who is having their birthday today\n`!bd help` - see the abysmal help")
			} else {
				s.ChannelMessageSend(m.ChannelID, "**BirthdayBot Usage:**\n`!bd add <user> <dd/mm>` - add a new birthday to the database\n`!bd next` - see who is having their birthday next\n`!bd today` - check who is having their birthday today\n`!bd help` - see the abysmal help")
			}
		} else {
			s.ChannelMessageSend(m.ChannelID, "**BirthdayBot Usage:**\n`!bd add <user> <dd/mm>` - add a new birthday to the database\n`!bd next` - see who is having their birthday next\n`!bd today` - check who is having their birthday today\n`!bd help` - see the abysmal help")
		}
	}
}

func sendMessageInTimeInterval(s *discordgo.Session, _ *discordgo.Ready) {
	_ = commands.SetupBirthdayDatabase(BotConfig.DB)
	//wishHappyBirthday(BotConfig.DB, s)
	ticker := time.NewTicker(1 * time.Hour)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				if utils.InTimeInterval("08:00:00", "09:00:00", time.Now()) {
					commands.WishHappyBirthday(BotConfig.DB, s, BotConfig)
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}
