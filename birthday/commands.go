package commands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

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

var validActions = map[string]func(*DiscordBot, *Command){
	"add":   (*DiscordBot).AddBirthday,     // add <user> <date>
	"next":  (*DiscordBot).NextBirthday,    // next
	"when":  (*DiscordBot).WhenBirthday,    // when
	"today": (*DiscordBot).TodaysBirthdays, // today
	"setup": (*DiscordBot).StartDiscordBot, // setup <timezone> <time>
	"help":  (*DiscordBot).Help,            // help
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
	tz := command.ID
	_, err := time.LoadLocation(tz)
	if err != nil {
		message := fmt.Sprintf("invalid time zone '%s'", tz)
		utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
		return
	}
	datetime := command.DateTime
	message := fmt.Sprintf("invalid time %s, must be within 0 and 23 (inclusive)", datetime)
	n, err := strconv.Atoi(datetime)
	if err != nil {
		utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
		return
	}
	if n < 0 || n > 23 {
		utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
		return
	}
	err = SetupBirthdayDatabase(command.Database, command.Channel, tz, command.Server, datetime)
	if err != nil {
		message = "failed to set up database."
	} else {
		message = fmt.Sprintf("successfully set up database in timezone %s with reminder at %s:00", tz, datetime)
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
	for action, execute := range validActions {
		if command.Action == action {
			execute(d, &command)
			return
		}
	}
	message := fmt.Sprintf("invalid action %s", command.Action)
	utils.LogAndSend(d.session, command.Channel, command.Server, message, err)
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

	command.Action = cleanedSplitCommand[1]

	if commandLength > 2 {
		command.ID = cleanedSplitCommand[2]
	}

	if commandLength > 3 {
		command.DateTime = cleanedSplitCommand[3]
	}

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
	user := utils.GetIDFromMention(command.ID)
	b, id := utils.IsUser(user, d.session, command.Server)
	if !b {
		message := fmt.Sprintf("invalid user %s", user)
		utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
		return
	}
	if !utils.IsValidDate(command.DateTime) {
		message := fmt.Sprintf("invalid date %s", user)
		utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
		return
	}
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
	err = AddBirthdayToDatabase(command.Database, id, datetime)
	if err != nil {
		message := fmt.Sprintf("error adding birthday to database: %s", err.Error())
		utils.LogAndSend(d.session, command.Channel, command.Server, message, err)
		return
	}
	message := fmt.Sprintf("Successfully added birthday for <@!%s> on %s %d", id, datetime.Month().String(), datetime.Day())
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
	user := utils.GetIDFromMention(command.ID)
	b, id := utils.IsUser(user, d.session, command.Server)
	if !b {
		message := fmt.Sprintf("invalid user %s", user)
		utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
		return
	}
	var message string
	birthday, err := CheckForUsersBirthdayInDatabase(command.Database, id)
	if err != nil {
		message := fmt.Sprintf("error checking for users birthday %s", err.Error())
		utils.LogAndSend(d.session, command.Channel, command.Server, message, err)
		return
	}
	if birthday == time.Unix(0, 0) {
		message = fmt.Sprintf("<@%s>'s birthday not in database", id)
	} else {
		message = fmt.Sprintf("<@%s>'s birthday is the %s %d", id, birthday.Month(), birthday.Day())
	}
	utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
}

func (d *DiscordBot) Help(command *Command) {
	help := "**BirthdayBot Usage:**\n`!bd add <user> <dd/mm>` - add a new birthday to the database\n`!bd next` - see who is having their birthday next\n`!bd today` - check who is having their birthday today\n`!bd when <user>` - see a specific users birthday\n`!bd setup <timezone/tz> <hour 0..23>` - run the setup\n`!bd help` - see the abysmal help"
	utils.LogAndSend(d.session, command.Channel, command.Server, help, nil)
}
