package discord_bot

import (
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/boltdb/bolt"
	"github.com/bwmarrin/discordgo"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/joshjennings98/discord-bot/utils"
)

type BotConfiguration struct {
	Token   string `mapstructure:"token"`
	DB      string `mapstructure:"birthdays_db"`
	Channel string `mapstructure:"channel"`
	Server  string `mapstructure:"server"`
}

var (
	BotConfig BotConfiguration
)

func DefaultBotConfig() *BotConfiguration {
	return &BotConfiguration{
		Token:   "",
		DB:      "",
		Channel: "",
		Server:  "",
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
		validation.Field(&cfg.DB, validation.Required),
		validation.Field(&cfg.Channel, validation.Required),
		validation.Field(&cfg.Server, validation.Required),
	)
}

func StartBot() (err error) {

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + BotConfig.Token)
	if err != nil {
		return fmt.Errorf("error creating Discord session: %w", err)
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)
	dg.AddHandler(atInterval)

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

	// Cleanly close down the Discord session.
	dg.Close()
	return nil
}

func isValidDate(s string) bool {
	re := regexp.MustCompile(`^(3[01]|[12][0-9]|0?[1-9])/(1[0-2]|0?[1-9])/(?:[0-9]{2})?[0-9]{2}$`)
	return re.MatchString(s)
}

func isUser(input string, s *discordgo.Session) (b bool, id string) {
	user := strings.ReplaceAll(input, "<", "")
	user = strings.ReplaceAll(user, ">", "")
	user = strings.ReplaceAll(user, "@", "")
	user = strings.ReplaceAll(user, "!", "")
	_, err := s.GuildMember(BotConfig.Server, user)

	if err != nil {
		return false, user
	}

	return true, user
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	for _, user := range m.Mentions {
		if user.ID == s.State.User.ID {
			if re := regexp.MustCompile(`[hello|hi] .*`); re.MatchString(strings.ToLower(m.Content)) {
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Hello %s", m.Author.Mention()))
			}
		}
	}

	if strings.HasPrefix(m.Content, "!bd") {
		command := strings.Split(m.Content, " ")
		if len(command) != 4 {
			s.ChannelMessageSend(m.ChannelID, "Only adding birthdays are supported right now. Please use the format '!bd add <user> <date>")
		} else if b, id := isUser(command[2], s); command[1] == "add" && b && isValidDate(command[3]) {
			date, err := time.Parse("02/01/06 03:04:05 PM", command[3]+" 00:00:00 AM")
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, "Date must be of the format dd/mm/yy")
				return
			}
			addBirthdayToDatabase(BotConfig.DB, id, date)
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Successfully added birthday for <@!%s> on %s", id, command[3]))
		} else if !isValidDate(command[3]) {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Invalid date %s", command[3]))
		}
	}
}

func checkForBirthdayInDatabase(dbPath string) (birthdays []string, err error) {
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("could not open db, %v", err)
	}
	defer db.Close()
	birthdays = []string{}
	date := strconv.Itoa(time.Now().YearDay())
	println("Checking for today's birthdays")
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("DB")).Bucket([]byte("BIRTHDAYS"))
		b.ForEach(func(k, v []byte) error {
			if string(v) == date {
				birthdays = append(birthdays, string(k))
			}
			return nil
		})
		return nil
	})
	return
}

func addBirthdayToDatabase(dbPath string, id string, date time.Time) error {
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return fmt.Errorf("could not open db, %v", err)
	}
	defer db.Close()
	dateString := strconv.Itoa(date.YearDay())
	err = db.Update(func(tx *bolt.Tx) error {
		err := tx.Bucket([]byte("DB")).Bucket([]byte("BIRTHDAYS")).Put([]byte(id), []byte(dateString))
		if err != nil {
			return fmt.Errorf("could not insert birthday: %v", err)
		}
		return nil
	})
	fmt.Printf("Added Birthday for %s on %s\n", id, date.String())
	return err
}

func setupBirthdayDatabase(dbPath string) (err error) {
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return fmt.Errorf("could not open db, %v", err)
	}
	defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error {
		root, err := tx.CreateBucketIfNotExists([]byte("DB"))
		if err != nil {
			return fmt.Errorf("could not create root bucket: %v", err)
		}
		_, err = root.CreateBucketIfNotExists([]byte("BIRTHDAYS"))
		if err != nil {
			return fmt.Errorf("could not create birthdays bucket: %v", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("could not set up buckets, %v", err)
	}
	fmt.Println("DB Setup Done")
	return nil
}

func wishHappyBirthday(s string, session *discordgo.Session) {
	birthdays, _ := checkForBirthdayInDatabase(s)
	for _, b := range birthdays {
		session.ChannelMessageSend(BotConfig.Channel, fmt.Sprintf("Happy Birthday <@%s>!!! :partying_face:", b))
	}
}

func inTimeSpan(s1, s2, s3 string) bool {
	start, err := time.Parse("15:04:05", s1)
	if err != nil {
		return false
	}
	end, err := time.Parse("15:04:05", s2)
	if err != nil {
		return false
	}
	check, err := time.Parse("15:04:05", s3)
	if err != nil {
		return false
	}

	if start.Before(end) {
		println(!check.Before(start), !check.After(end), check.String(), s1, end.String())
		return !check.Before(start) && !check.After(end)
	}
	if start.Equal(end) {
		return check.Equal(start)
	}
	return !start.After(check) || !end.Before(check)
}

func atInterval(s *discordgo.Session, _ *discordgo.Ready) {
	_ = setupBirthdayDatabase(BotConfig.DB)
	wishHappyBirthday(BotConfig.DB, s)
	ticker := time.NewTicker(1 * time.Hour)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				now := time.Now()
				if inTimeSpan("08:00:00", "09:00:00", fmt.Sprintf("%d:%d:00", now.Hour(), now.Minute())) {
					wishHappyBirthday(BotConfig.DB, s)
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}
