package discord_bot

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joshjennings98/discord-bot/commands"
	"github.com/joshjennings98/discord-bot/utils"
	log "github.com/sirupsen/logrus"
)

var (
	BotConfig  commands.BotConfiguration
	DiscordBot commands.DiscordBot

	hiRegex *regexp.Regexp
	tyRegex *regexp.Regexp
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
	dg.AddHandler(onReady)
	// We only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// Attach DiscordBot to session
	DiscordBot.AttachBotToSession(dg)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		return fmt.Errorf("error opening connection: %w", err)
	}

	// setup regex stuff
	hiRegex = regexp.MustCompile(fmt.Sprintf(`^(hello|hi) <@!?%s>`, dg.State.User.ID))
	tyRegex = regexp.MustCompile(fmt.Sprintf(`^(thanks|ty|thank you) <@!?%s>`, dg.State.User.ID)) // deliberate design decision to allow for stuff after the thank you in case there is more content to the thanks

	// Wait here until CTRL-C or other term signal is received.
	log.Info("Bot is now running.  Press CTRL-C to exit.")
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

	// Check for interaction with bot
	for _, user := range m.Mentions {
		if user.ID == s.State.User.ID {
			log.Info(fmt.Sprintf("DiscordBot mentioned in message: '%s'", m.Content))
			// Check for someone saying hi
			if hiRegex.MatchString(strings.ToLower(m.Content)) {
				utils.LogAndSend(s, m.ChannelID, m.GuildID, fmt.Sprintf("Hello %s", m.Author.Mention()), nil)
			}
			// Check for someone saying thank you
			if tyRegex.MatchString(strings.ToLower(m.Content)) {
				utils.LogAndSend(s, m.ChannelID, m.GuildID, fmt.Sprintf("You are welcome %s", m.Author.Mention()), nil)
			}
		}
	}

	// Check for prefix
	if strings.HasPrefix(m.Content, prefixCmd) {
		DiscordBot.ExecuteCommand(m)
	}
}

func onReady(s *discordgo.Session, _ *discordgo.Ready) {
	ticker := time.NewTicker(1 * time.Hour)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				databases, err := ioutil.ReadDir("./databases")
				if err != nil {
					log.Errorf("Could not find databases")
				}
				for _, db := range databases {
					tz, err := commands.GetTimezone(db.Name())
					if err != nil {
						log.Error(fmt.Sprintf("Could not get timezone from database %s", db.Name()))
					}
					loc, err := time.LoadLocation(tz)
					if err != nil {
						log.Errorf("Invalid location '%s'", loc)
					}
					interval, err := commands.GetTimeInterval(db.Name())
					if err != nil {
						log.Errorf("Could not get time interval from database '%s'", db.Name())
					}
					i, err := strconv.Atoi(interval)
					if err != nil {
						log.Errorf("Invalid interval '%s'", interval)
					}
					if utils.InHourInterval(i, time.Now().In(loc)) {
						commands.WishTodaysHappyBirthdays(s, db.Name())
					}
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}
