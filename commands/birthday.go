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
	- Fix 29/02
	- For NEXT thing add a date
	- Switch to proper / commands instead of checking every message?
	- Remove non-birthday stuff (like the hi and ty stuff)?????
	- ADD TESTS
	- ADD CI
	- Add locks to database checking
	- Make run on multiple servers
	- add timezones
	- base timezone on the servers timezone (this needs to be set up when the bot is added (default to GMT))
	- change birthday with to `@everyone, it is @user's birthday today :party_face:`
	- make the message customisable
	- (REMOVE THE UNNECESSARY UTILS)
	- CLEAN UP FILES (move databse stuff to new file.)
	- MAKE WORK ASYNCHRONOUSLY
	- CREATE COMMONERRORS SORT OF THING SO ERRORS LOOK NICER
	- IMPROVE ERRORS SO THEY DON'T SEND AS MUCH INFO TO DISCORD CHANNELS AND ONLY LOG ERRORS CAUSED BY ME
	- ALSO MOVE ERRORS AROUND GET RID OF POINTLESS ONES
*/

type Birthday struct {
	ID   string
	Date string
}

type Birthdays []Birthday

func (b Birthdays) Len() int {
	return len(b)
}
func (a Birthdays) Less(i, j int) (b bool) {
	ai, err := strconv.ParseInt(a[i].Date, 10, 64)
	if err != nil {
		log.Errorf("Failed to parse unix time %s", ai)
	}
	aj, err := strconv.ParseInt(a[j].Date, 10, 64)
	if err != nil {
		log.Errorf("Failed to parse unix time %s", aj)
	}
	return time.Unix(ai, 0).YearDay() < time.Unix(aj, 0).YearDay() // YearDay :)
}

func (a Birthdays) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

type Command struct {
	Action   string
	ID       string
	DateTime string
	Channel  string
	// When running on multiple servers then we also need a serverID
	Server   string
	Database string
}

type DiscordBot struct {
	session *discordgo.Session
}

// add <user> <date>
// next
// when
// today
// help
// start <timezone> <time>
var validActions = map[string]func(*DiscordBot, *Command){
	"add":   (*DiscordBot).AddBirthday,
	"next":  (*DiscordBot).NextBirthday,
	"when":  (*DiscordBot).WhenBirthday,
	"today": (*DiscordBot).TodaysBirthdays,
	"setup": (*DiscordBot).StartDiscordBot,
	"help":  (*DiscordBot).Help,
}

type IDiscordBot interface {
	AttachBotToSession(session *discordgo.Session)
	ParseInput(input string) (command Command, err error)
	ExecuteCommand(command Command)
	StartDiscordBot(command Command)
	WishTodaysHappyBirthdays()
	TodaysBirthdays(command *Command)
	NextBirthday(command *Command)
	AddBirthday(command *Command)
	WhenBirthday(command *Command)
	Help(command *Command)
}

func (d *DiscordBot) AttachBotToSession(session *discordgo.Session) {
	d.session = session
}

func (d *DiscordBot) StartDiscordBot(command *Command) {
	var message string
	err := SetupBirthdayDatabase(command.Database, command.Channel, command.ID, command.Server, command.DateTime)
	if err != nil {
		message = "failed to set up database."
	} else {
		message = fmt.Sprintf("successfully set up database in timezone %s with reminder at %s:00", command.ID, command.DateTime)
	}
	utils.LogAndSend(d.session, command.Channel, command.Server, message, err)
}

func (d *DiscordBot) ExecuteCommand(input *discordgo.MessageCreate) {
	command, err := d.ParseInput(input)
	if err != nil {
		message := fmt.Sprintf("Error parsing command: %s", err.Error())
		utils.LogAndSend(d.session, command.Channel, command.Server, message, err)
		return
	}
	// set correct channel to execute command on
	switch command.Action {
	case "add":
		d.AddBirthday(&command)
	case "when":
		d.WhenBirthday(&command)
	case "next":
		d.NextBirthday(&command)
	case "today":
		d.TodaysBirthdays(&command)
	case "setup":
		d.StartDiscordBot(&command)
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

func (d *DiscordBot) ParseInput(m *discordgo.MessageCreate) (command Command, err error) {
	server := m.GuildID
	command.Server = server
	command.Channel = m.ChannelID
	command.Database = utils.DatabaseFromServerID(server)
	split := strings.Split(m.Content, " ")
	var cleanedSplitCommand []string
	for _, str := range split {
		if str != "" {
			cleanedSplitCommand = append(cleanedSplitCommand, str)
		}
	}

	commandLength := len(cleanedSplitCommand)

	if commandLength < 2 || commandLength > 4 {
		err = fmt.Errorf("invalid length of command: %s", m.Content)
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
		switch action {
		case "add", "when":
			user := utils.GetIDFromMention(cleanedSplitCommand[2])
			if b, id := utils.IsUser(user, d.session, server); b {
				command.ID = id
			} else {
				err = fmt.Errorf("invalid user %s", user)
				return
			}
		case "setup":
			tz := cleanedSplitCommand[2]
			_, err1 := time.LoadLocation(tz)
			if err1 != nil {
				err = fmt.Errorf("invalid time zone '%s'", tz)
				return
			}
			command.ID = tz
		}
	}

	if commandLength > 3 {
		datetime := cleanedSplitCommand[3]
		switch action {
		case "add":
			if utils.IsValidDate(datetime) {
				command.DateTime = datetime
			} else {
				err = fmt.Errorf("invalid date %s", datetime)
				return
			}
		case "setup":
			invalidTimeError := fmt.Errorf("invalid time %s, must be within 0 and 23 (inclusive)", datetime)
			n, err1 := strconv.Atoi(datetime)
			if err1 != nil {
				err = invalidTimeError
				return
			}
			if 0 <= n && n < 24 {
				command.DateTime = datetime
			} else {
				err = invalidTimeError
				return
			}
		}
	}

	return
}

func CheckForBirthdaysInDatabase(database string, t time.Time) (birthdays []string, err error) {
	path := "databases/" + database
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("could not open db, %v", err)
	}
	defer db.Close()

	birthdays = []string{}
	date := strconv.Itoa(t.YearDay())
	log.Info("Checking for today's birthdays")
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("DB")).Bucket([]byte("BIRTHDAYS"))
		b.ForEach(func(k, v []byte) (err1 error) {
			i, err1 := strconv.ParseInt(string(v), 10, 64)
			if err1 != nil {
				return
			}
			if strconv.Itoa(time.Unix(i, 0).YearDay()) == date {
				birthdays = append(birthdays, string(k))
			}
			return nil
		})
		return nil
	})
	return
}

func CheckForUsersBirthdayInDatabase(database, userID string) (birthday time.Time, err error) {
	path := "databases/" + database
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return time.Unix(0, 0), fmt.Errorf("could not open db, %v", err)
	}
	defer db.Close()

	log.Info(fmt.Sprintf("Checking for %s's birthdays in %s", userID, database))
	err = db.View(func(tx *bolt.Tx) error {
		bd := string(tx.Bucket([]byte("DB")).Bucket([]byte("BIRTHDAYS")).Get([]byte(userID)))
		i, err := strconv.ParseInt(bd, 10, 64)
		if err != nil {
			return err
		}
		birthday = time.Unix(i, 0)
		return nil
	})
	return
}

func AddBirthdayToDatabase(database, id string, date time.Time) error {
	path := "databases/" + database
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return fmt.Errorf("could not open db, %v", err)
	}
	defer db.Close()

	dateString := strconv.FormatInt(date.Unix(), 10)
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
	path := "databases/" + database
	db, err := bolt.Open(path, 0600, nil)
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

func SetupBirthdayDatabase(database, defaultChannel, timezone, server, interval string) (err error) {
	path := "databases/" + database
	db, err := bolt.Open(path, 0600, nil)
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
		err = tx.Bucket([]byte("DB")).Put([]byte("default channel"), []byte(defaultChannel))
		if err != nil {
			return fmt.Errorf("could not insert default channel: %v", err)
		}
		err = tx.Bucket([]byte("DB")).Put([]byte("ServerID"), []byte(server))
		if err != nil {
			return fmt.Errorf("could not insert server id: %v", err)
		}
		err = tx.Bucket([]byte("DB")).Put([]byte("timezone"), []byte(timezone))
		if err != nil {
			return fmt.Errorf("could not insert timezone: %v", err)
		}
		err = tx.Bucket([]byte("DB")).Put([]byte("time interval"), []byte(interval))
		if err != nil {
			return fmt.Errorf("could not insert time interval: %v", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("could not set up buckets, %v", err)
	}
	log.Info("Database Setup Done")
	return nil
}

func GetDefaultChannel(database string) (channel string, err error) {
	path := "databases/" + database
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return "", fmt.Errorf("could not open db, %v", err)
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("DB"))
		channel = string(b.Get([]byte("default channel")))
		return nil
	})
	return
}

func GetServerID(database string) (server string, err error) {
	path := "databases/" + database
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return "", fmt.Errorf("could not open db, %v", err)
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("DB"))
		server = string(b.Get([]byte("ServerID")))
		return nil
	})
	return
}

func GetTimezone(database string) (tz string, err error) {
	path := "databases/" + database
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return "", fmt.Errorf("could not open db, %v", err)
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("DB"))
		tz = string(b.Get([]byte("timezone")))
		return nil
	})
	return
}

func GetTimeInterval(database string) (interval string, err error) {
	path := "databases/" + database
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return "", fmt.Errorf("could not open db, %v", err)
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("DB"))
		interval = string(b.Get([]byte("time interval")))
		return nil
	})
	return
}

func WishTodaysHappyBirthdays(s *discordgo.Session, database string) {
	channel, err := GetDefaultChannel(database)
	if err != nil {
		log.Errorf("failed to get the default channel")
		return
	}
	server, err := GetServerID(database)
	if err != nil {
		log.Errorf("failed to get the server id")
		return
	}
	birthdays, err := CheckForBirthdaysInDatabase(database, time.Now())
	if err != nil {
		log.Errorf("failed to get todays birthdays")
		return
	}
	for _, b := range birthdays {
		message := fmt.Sprintf("Happy Birthday <@%s>!!! :partying_face:", b)
		utils.LogAndSend(s, channel, server, message, nil)
	}
}

func (d *DiscordBot) AddBirthday(command *Command) {
	var baseDate string
	// account for leap years
	if command.DateTime == "29/02" {
		baseDate = "/00 00:00:00 AM"
	} else {
		baseDate = "/01 00:00:00 AM"
	}
	datetime, err := time.Parse("02/01/06 03:04:05 PM", command.DateTime+baseDate) // get in the right format (TODO: should clean this up)
	if err != nil {
		log.Info(fmt.Sprintf("error parsing date: %s", err.Error()))
		return
	}
	err = AddBirthdayToDatabase(command.Database, command.ID, datetime)
	if err != nil {
		message := fmt.Sprintf("error adding birthday to database: %s", err.Error())
		utils.LogAndSend(d.session, command.Channel, command.Server, message, err)
		return
	}
	message := fmt.Sprintf("Successfully added birthday for <@!%s> on %s %d", command.ID, datetime.Month().String(), datetime.Day())
	utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
}

func (d *DiscordBot) TodaysBirthdays(command *Command) {
	birthdays, _ := CheckForBirthdaysInDatabase(command.Database, time.Now())
	var message string
	for _, b := range birthdays {
		message = fmt.Sprintf("<@%s> has their birthday today :smile:", b)
	}
	if len(birthdays) == 0 {
		message = "Nobody has their birthday today :cry:"
	}
	utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
}

func (d *DiscordBot) NextBirthday(command *Command) {
	today := time.Now().YearDay()
	birthdays, err := GetBirthdaysFromDatabase(command.Database)
	if err != nil {
		message := fmt.Sprintf("Error getting birthdays from database: %s", err.Error())
		utils.LogAndSend(d.session, command.Channel, command.Server, message, err)
		return
	}
	if len(birthdays) == 0 {
		message := "There are no birthdays in the database."
		utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
		return
	}
	sort.Sort(birthdays) // sort by date

	// These are for if we need to wrap around with birthdays. We keep track of the first birthday to reduce amount of parsing
	var firstBirthdayDate int
	var firstBirthdayID string

	for i, birthday := range birthdays {
		date64, err := strconv.ParseInt(birthday.Date, 10, 64)
		if err != nil {
			message := fmt.Sprintf("Error parsing birthday: %s", err.Error())
			utils.LogAndSend(d.session, command.Channel, command.Server, message, err)
			return
		}
		date := int(time.Unix(date64, 0).YearDay())
		// to reduce logic for wrap around
		if i == 0 {
			firstBirthdayDate = date
			firstBirthdayID = birthday.ID
		}
		if date > today {
			message := fmt.Sprintf("The next person to have their birthday is <@%s> in %d days.", birthday.ID, (date - today))
			utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
			return
		}
	}
	// catch any dates that have wrapped round (will only reach if no birthdays after today)
	message := fmt.Sprintf("The next person to have their birthday is <@%s> in %d days.", firstBirthdayID, (utils.DaysInThisYear() - today + firstBirthdayDate))
	utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
}

func (d *DiscordBot) WhenBirthday(command *Command) {
	var message string
	birthday, err := CheckForUsersBirthdayInDatabase(command.Database, command.ID)
	if err != nil {
		message := fmt.Sprintf("error checking for users birthday <@%s:>", err.Error())
		utils.LogAndSend(d.session, command.Channel, command.Server, message, err)
		return
	}
	if birthday == time.Unix(0, 0) {
		message = fmt.Sprintf("<@%s>'s birthday not in database", command.ID)
	} else {
		message = fmt.Sprintf("<@%s>'s birthday is the %s %d", command.ID, birthday.Month(), birthday.Day())
	}
	utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
}

func (d *DiscordBot) Help(command *Command) {
	help := "**BirthdayBot Usage:**\n`!bd add <user> <dd/mm>` - add a new birthday to the database\n`!bd next` - see who is having their birthday next\n`!bd today` - check who is having their birthday today\n`!bd when <user>` - see a specific users birthday\n`!bd setup <timezone/tz> <hour 0..23>` - run the setup\n`!bd help` - see the abysmal help"
	utils.LogAndSend(d.session, command.Channel, command.Server, help, nil)
}
