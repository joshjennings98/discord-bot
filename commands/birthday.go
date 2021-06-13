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
	log "github.com/sirupsen/logrus"
)

/*
	TODO:
	- Fix how times are added
	= Fix 29/02
	- For NEXT thing add a date
	- ADD TESTS
	- ADD CI
	- Make run on multiple servers
	- add timezones
	- change birthday with to `@everyone, it is @user's birthday today :party_face:`
	- make the message customisable
	- (REMOVE THE UNNECESSARY UTILS)
	- CLEAN UP FILES (move databse stuff to new file.)
	- MAKE WORK ASYNCHRONOUSLY
	- CREATE COMMONERRORS SORT OF THING SO ERRORS LOOK NICER
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
	Action  string
	UserID  string
	Date    time.Time
	Channel string
	// When running on multiple servers then we also need a serverID
}

type DiscordBot struct {
	cfg     BotConfiguration
	session *discordgo.Session
}

var validActions = map[string]func(*DiscordBot, *Command){
	"add":   (*DiscordBot).AddBirthday,
	"next":  (*DiscordBot).NextBirthday,
	"when":  (*DiscordBot).WhenBirthday,
	"today": (*DiscordBot).TodaysBirthdays,
	"help":  (*DiscordBot).Help,
}

type IDiscordBot interface {
	SetupDiscordBot(cfg BotConfiguration, session *discordgo.Session)
	ParseInput(input string) (command Command, err error)
	ExecuteCommand(command Command)
	WishTodaysHappyBirthdays()
	TodaysBirthdays(command *Command)
	NextBirthday(command *Command)
	AddBirthday(command *Command)
	WhenBirthday(command *Command)
	Help(command *Command)
}

func (d *DiscordBot) SetupDiscordBot(cfg BotConfiguration, session *discordgo.Session) {
	d.cfg = cfg
	d.session = session
}

func (d *DiscordBot) ExecuteCommand(channelID string, input string) {
	command, err := d.ParseInput(input)
	if err != nil {
		message := fmt.Sprintf("Error parsing command: %s", err.Error())
		utils.LogAndSend(d.session, channelID, message, err)
		return
	}
	// set correct channel to execute command on
	command.Channel = channelID
	switch command.Action {
	case "add":
		d.AddBirthday(&command)
	case "when":
		d.WhenBirthday(&command)
	case "next":
		d.NextBirthday(&command)
	case "today":
		d.TodaysBirthdays(&command)
	case "help":
		d.Help(&command)
	}
}

func isValidAction(action string) bool {
	for validAction := range validActions {
		if validAction == action {
			return true
		}
	}
	return false
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

	action := cleanedSplitCommand[1]
	if isValidAction(action) {
		command.Action = action
	} else {
		err = fmt.Errorf("invalid action %s", action)
		return
	}

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
	log.Info("Checking for today's birthdays")
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

	log.Info("Checking for today's birthdays")
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
	log.Info(fmt.Sprintf("Added Birthday for %s on %s %d\n", id, date.Month().String(), date.Day()))
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
	log.Info("Database Setup Done")
	return nil
}

func (d *DiscordBot) WishTodaysHappyBirthdays() {
	birthdays, _ := CheckForBirthdaysInDatabase(d.cfg.DB, time.Now())
	for _, b := range birthdays {
		message := fmt.Sprintf("Happy Birthday <@%s>!!! :partying_face:", b)
		utils.LogAndSend(d.session, d.cfg.Channel, message, nil)
	}
}

func (d *DiscordBot) AddBirthday(command *Command) {
	err := AddBirthdayToDatabase(d.cfg.DB, command.UserID, command.Date)
	if err != nil {
		message := fmt.Sprintf("error adding birthday to database: %s", err.Error())
		utils.LogAndSend(d.session, command.Channel, message, err)
		return
	}
	message := fmt.Sprintf("Successfully added birthday for <@!%s> on %s %d", command.UserID, command.Date.Month().String(), command.Date.Day())
	utils.LogAndSend(d.session, command.Channel, message, nil)
}

func (d *DiscordBot) TodaysBirthdays(command *Command) {
	birthdays, _ := CheckForBirthdaysInDatabase(d.cfg.DB, time.Now())
	var message string
	for _, b := range birthdays {
		message = fmt.Sprintf("<@%s> has their birthday today :smile:", b)
	}
	if len(birthdays) == 0 {
		message = "Nobody has their birthday today :cry:"
	}
	utils.LogAndSend(d.session, command.Channel, message, nil)
}

func (d *DiscordBot) NextBirthday(command *Command) {
	today := time.Now().YearDay()
	birthdays, err := GetBirthdaysFromDatabase(d.cfg.DB)
	if err != nil {
		message := fmt.Sprintf("Error getting birthdays from database: %s", err.Error())
		utils.LogAndSend(d.session, command.Channel, message, err)
		return
	}
	if len(birthdays) == 0 {
		message := "There are no birthdays in the database."
		utils.LogAndSend(d.session, command.Channel, message, nil)
		return
	}
	sort.Sort(birthdays) // sort by date

	// These are for if we need to wrap around with birthdays. We keep track of the first birthday to reduce amount of parsing
	var firstBirthdayDate int
	var firstBirthdayID string

	for i, birthday := range birthdays {
		date, err := strconv.Atoi(birthday.Date)
		if err != nil {
			message := fmt.Sprintf("Error parsing birthday: %s", err.Error())
			utils.LogAndSend(d.session, command.Channel, message, err)
			return
		}
		// to reduce logic for wrap around
		if i == 0 {
			firstBirthdayDate = date
			firstBirthdayID = birthday.ID
		}
		if date > today {
			message := fmt.Sprintf("The next person to have their birthday is <@%s> in %d days.", birthday.ID, (date - today))
			utils.LogAndSend(d.session, command.Channel, message, nil)
			return
		}
	}
	// catch any dates that have wrapped round (will only reach if no birthdays after today)
	message := fmt.Sprintf("The next person to have their birthday is <@%s> in %d days.", firstBirthdayID, (utils.DaysInThisYear() - today + firstBirthdayDate))
	utils.LogAndSend(d.session, command.Channel, message, nil)
}

func (d *DiscordBot) WhenBirthday(command *Command) {
	var message string
	birthday, err := CheckForUsersBirthdayInDatabase(d.cfg.DB, command.UserID)
	if err != nil {
		message := fmt.Sprintf("error checking for users birthday <@%s:>", err.Error())
		utils.LogAndSend(d.session, command.Channel, message, err)
		return
	}
	if birthday == "" {
		message = fmt.Sprintf("<@%s>'s birthday not in database", command.UserID)
	} else {
		bd, err := utils.ConvertYearDayToDate(birthday)
		if err != nil {
			message := fmt.Sprintf("Error parsing birthday <@%s>:", birthday)
			utils.LogAndSend(d.session, command.Channel, message, err)
			return
		}
		message = fmt.Sprintf("<@%s>'s birthday is the %s", command.UserID, bd)
	}
	utils.LogAndSend(d.session, command.Channel, message, nil)
}

func (d *DiscordBot) Help(command *Command) {
	help := "**BirthdayBot Usage:**\n`!bd add <user> <dd/mm>` - add a new birthday to the database\n`!bd next` - see who is having their birthday next\n`!bd today` - check who is having their birthday today\n`!bd when <user>` - see a specific users birthday\n`!bd help` - see the abysmal help"
	utils.LogAndSend(d.session, command.Channel, help, nil)
}
