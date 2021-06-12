package commands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/bwmarrin/discordgo"
	"github.com/joshjennings98/discord-bot/utils"
)

/*
	TODO:
	- (REMOVE THE UNNECESSARY UTILS)
	- CLEAN UP FILES
	- MAKE WORK ASYNCHRONOUSLY
	- CREATE COMMONERRORS SORT OF THING SO ERRORS LOOK NICER
	- MAKE FMT.PRINTF STUFF A PROPER LOG
*/

type Birthday struct {
	ID   string
	Date string
}

type Birthdays []Birthday

func (b Birthdays) Len() int {
	return len(b)
}
func (a Birthdays) Less(i, j int) bool {
	return a[i].Date < a[j].Date // YearDay :)
}

func (a Birthdays) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

type Command struct {
	Action string
	UserID string
	Date   time.Time
}

type DiscordBot struct {
	cfg     BotConfiguration
	session *discordgo.Session
}

type IDiscordBot interface {
	SetupDiscordBot(cfg BotConfiguration, session *discordgo.Session)
	ParseInput(input string) (command Command, err error)
	ExecuteCommand(command Command) (err error)
	WishTodaysHappyBirthdays()
	TodaysBirthdays(channelID string)
	NextBirthday(channelID string)
	AddBirthday(channelID, id string, date time.Time)
	WhenBirthday(channelID, id string)
	Help(channelID string)
}

func (d *DiscordBot) SetupDiscordBot(cfg BotConfiguration, session *discordgo.Session) {
	d.cfg = cfg
	d.session = session
}

func (d *DiscordBot) ExecuteCommand(channelID string, command Command) (err error) {
	switch command.Action {
	case "add":
		d.AddBirthday(channelID, command.UserID, command.Date)
	case "when":
		d.WhenBirthday(channelID, command.UserID)
	case "next":
		d.NextBirthday(channelID)
	case "today":
		d.TodaysBirthdays(channelID)
	case "help":
		d.Help(channelID)
	}
	return
}

func (d *DiscordBot) ParseInput(input string) (command Command, err error) {
	split := strings.Split(input, " ")
	var cleanedSplitCommand []string
	for _, str := range split {
		if str != "" {
			cleanedSplitCommand = append(cleanedSplitCommand, str)
		}
	}

	commandLength := len(cleanedSplitCommand)

	if commandLength < 2 || commandLength > 4 {
		err = fmt.Errorf("invalid length of command: %s", input)
		return
	}

	command.Action = cleanedSplitCommand[1]

	if commandLength > 2 {
		user := utils.GetIDFromMention(cleanedSplitCommand[2])
		if b, id := utils.IsUser(user, d.session, d.cfg.Server); b {
			command.UserID = id
		} else {
			err = fmt.Errorf("invalid user %s", user)
			return
		}
	}

	if commandLength > 3 {
		date := cleanedSplitCommand[3]
		if b := utils.IsValidDate(date); b {
			var baseDate string
			// account for leap years
			if date == "29/02" {
				baseDate = "/00 00:00:00 AM"
			} else {
				baseDate = "/01 00:00:00 AM"
			}
			d, err := time.Parse("02/01/06 03:04:05 PM", date+baseDate) // get in the right format (TODO: should clean this up)
			if err != nil {
				return command, err
			}
			command.Date = d
		} else {
			err = fmt.Errorf("invalid date %s", date)
			return
		}
	}

	return
}

func CheckForBirthdaysInDatabase(database string, t time.Time) (birthdays []string, err error) {
	db, err := bolt.Open(database, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("could not open db, %v", err)
	}
	defer db.Close()
	birthdays = []string{}
	date := strconv.Itoa(t.YearDay())
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

func CheckForUsersBirthdayInDatabase(database, userID string) (birthday string, err error) {
	db, err := bolt.Open(database, 0600, nil)
	if err != nil {
		return "", fmt.Errorf("could not open db, %v", err)
	}
	defer db.Close()
	println("Checking for today's birthdays")
	err = db.View(func(tx *bolt.Tx) error {
		birthday = string(tx.Bucket([]byte("DB")).Bucket([]byte("BIRTHDAYS")).Get([]byte(userID)))
		return nil
	})
	return
}

func AddBirthdayToDatabase(database, id string, date time.Time) error {
	db, err := bolt.Open(database, 0600, nil)
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
	fmt.Printf("Added Birthday for %s on %s %d\n", id, date.Month().String(), date.Day())
	return err
}

func GetBirthdaysFromDatabase(database string) (birthdays Birthdays, err error) {
	db, err := bolt.Open(database, 0600, nil)
	if err != nil {
		return birthdays, fmt.Errorf("could not open db, %v", err)
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("DB")).Bucket([]byte("BIRTHDAYS"))
		b.ForEach(func(k, v []byte) error {
			birthdays = append(birthdays, Birthday{ID: string(k), Date: string(v)})
			return nil
		})
		return nil
	})
	return
}

func SetupBirthdayDatabase(database string) (err error) {
	db, err := bolt.Open(database, 0600, nil)
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

func (d *DiscordBot) AddBirthday(channelID, id string, date time.Time) {
	err := AddBirthdayToDatabase(d.cfg.DB, id, date)
	if err != nil {
		d.session.ChannelMessageSend(channelID, fmt.Sprintf("error adding birthday to database: %s", err.Error()))
		return
	}
	d.session.ChannelMessageSend(channelID, fmt.Sprintf("Successfully added birthday for <@!%s> on %s %d", id, date.Month().String(), date.Day()))
}

func (d *DiscordBot) WishTodaysHappyBirthdays() {
	birthdays, _ := CheckForBirthdaysInDatabase(d.cfg.DB, time.Now())
	for _, b := range birthdays {
		d.session.ChannelMessageSend(d.cfg.Channel, fmt.Sprintf("Happy Birthday <@%s>!!! :partying_face:", b))
	}
}

func (d *DiscordBot) TodaysBirthdays(channelID string) {
	birthdays, _ := CheckForBirthdaysInDatabase(d.cfg.DB, time.Now())
	for _, b := range birthdays {
		d.session.ChannelMessageSend(channelID, fmt.Sprintf("<@%s> has their birthday today :smile:", b))
	}
	if len(birthdays) == 0 {
		d.session.ChannelMessageSend(channelID, "Nobody has their birthday today :cry:")
	}
}

func (d *DiscordBot) NextBirthday(channelID string) {
	today := time.Now().YearDay()
	birthdays, err := GetBirthdaysFromDatabase(d.cfg.DB)
	if err != nil {
		d.session.ChannelMessageSend(channelID, fmt.Sprintf("Error getting birthdays from database: %s", err.Error()))
		return
	}
	if len(birthdays) == 0 {
		d.session.ChannelMessageSend(channelID, "There are no birthdays in the database.")
		return
	}
	sort.Sort(birthdays) // sort by date

	// These are for if we need to wrap around with birthdays. We keep track of the first birthday to reduce amount of parsing
	var firstBirthdayDate int
	var firstBirthdayID string

	for i, birthday := range birthdays {
		date, err := strconv.Atoi(birthday.Date)
		if err != nil {
			d.session.ChannelMessageSend(channelID, fmt.Sprintf("Error parsing birthday: %s", err.Error()))
			return
		}
		// to reduce logic for wrap around
		if i == 0 {
			firstBirthdayDate = date
			firstBirthdayID = birthday.ID
		}
		if date > today {
			d.session.ChannelMessageSend(channelID, fmt.Sprintf("The next person to have their birthday is <@%s> in %d days.", birthday.ID, (date-today)))
			return
		}
	}
	// catch any dates that have wrapped round (will only reach if no birthdays after today)
	d.session.ChannelMessageSend(channelID, fmt.Sprintf("The next person to have their birthday is <@%s> in %d days.", firstBirthdayID, (utils.DaysInThisYear()-today+firstBirthdayDate)))
}

func (d *DiscordBot) WhenBirthday(channelID, id string) {
	birthday, err := CheckForUsersBirthdayInDatabase(d.cfg.DB, id)
	if err != nil {
		d.session.ChannelMessageSend(channelID, fmt.Sprintf("error checking for users birthday <@%s:>", err.Error()))
		return
	}
	if birthday == "" {
		d.session.ChannelMessageSend(channelID, fmt.Sprintf("<@%s>'s birthday not in database", id))
	} else {
		bd, err := utils.ConvertYearDayToDate(birthday)
		if err != nil {
			d.session.ChannelMessageSend(channelID, fmt.Sprintf("Error parsing birthday <@%s>:", birthday))
		}
		d.session.ChannelMessageSend(channelID, fmt.Sprintf("<@%s>'s birthday is the %s", id, bd))
	}
}

func (d *DiscordBot) Help(channelID string) {
	d.session.ChannelMessageSend(channelID, "**BirthdayBot Usage:**\n`!bd add <user> <dd/mm>` - add a new birthday to the database\n`!bd next` - see who is having their birthday next\n`!bd today` - check who is having their birthday today\n`!bd when <user>` - see a specific users birthday\n`!bd help` - see the abysmal help")
}
