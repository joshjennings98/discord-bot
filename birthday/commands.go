package commands

import (
	"fmt"
	"path/filepath"
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
	- Switch to proper '/' commands instead of checking every message?
	- Add tests and CI
	- Make everything work asynchronously (not necessary as it isn't on more than a couple of servers)
*/

var validActions = map[string]func(*DiscordBot, *Command){
	"add":   (*DiscordBot).AddBirthday,     // add <user> <date>
	"next":  (*DiscordBot).NextBirthday,    // next
	"when":  (*DiscordBot).WhenBirthday,    // when <user>
	"today": (*DiscordBot).TodaysBirthdays, // today
	"setup": (*DiscordBot).StartDiscordBot, // setup <timezone> <time>
	"help":  (*DiscordBot).Help,            // help
}

// Need to use backticks so can't use normal multiline strings
const helpMessage = "**BirthdayBot Usage:**\n" +
	"`!bd add <user> <dd/mm>` - set a users birthday in the database\n" +
	"`!bd next` - see who is having their birthday next\n" +
	"`!bd today` - check who is having their birthday today\n" +
	"`!bd when <user>` - see a specific users birthday\n" +
	"`!bd setup <timezone/tz> <hour 0..23>` - run the setup\n" +
	"`!bd help` - see this help message"

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
	if command.ID == "" || command.DateTime == "" {
		message := "Error parsing command: command must be in the form '!bd <action> <arg1> <arg2>'"
		utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
		return
	}
	tz := command.ID
	_, err := time.LoadLocation(tz)
	if err != nil {
		message := fmt.Sprintf("Invalid time zone '%s'.", tz)
		utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
		return
	}
	datetime := command.DateTime
	message := fmt.Sprintf("Invalid hour interval '%s'. The hour interval must be within 0 and 23 (inclusive).", datetime)
	datetimeInt, err := strconv.Atoi(datetime)
	if err != nil {
		utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
		return
	}
	if datetimeInt < 0 || datetimeInt > 23 {
		utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
		return
	}
	err = SetupBirthdayDatabase(command.Database, command.Channel, tz, command.Server, datetime)
	if err != nil {
		message = "Failed to set up database."
	} else {
		message = fmt.Sprintf("Successfully set up database in timezone '%s' with reminder between %s:00 and %s:00.", tz, utils.AppendZero(datetimeInt), utils.AppendZero((datetimeInt+1)%24))
	}
	utils.LogAndSend(d.session, command.Channel, command.Server, message, err)
}

func (d *DiscordBot) ExecuteCommand(input *discordgo.MessageCreate) {
	command, err := d.ParseInput(input)
	if err != nil {
		message := fmt.Sprintf("Error parsing command: %s.", err.Error())
		utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
		return
	}
	// set correct channel to execute command on
	for action, execute := range validActions {
		if command.Action == action {
			execute(d, &command)
			return
		}
	}
	message := fmt.Sprintf("Invalid action '%s'.", command.Action)
	utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
}

func (d *DiscordBot) ParseInput(m *discordgo.MessageCreate) (command Command, err error) {
	server := m.GuildID
	command.Server = server
	command.Channel = m.ChannelID
	command.Database = filepath.Join(server /*d.databases, utils.DatabaseFromServerID(server) */)
	split := strings.Split(m.Content, " ")
	var cleanedSplitCommand []string
	for _, str := range split {
		if str != "" {
			cleanedSplitCommand = append(cleanedSplitCommand, str)
		}
	}

	commandLength := len(cleanedSplitCommand)

	if commandLength < 2 || commandLength > 4 {
		err = fmt.Errorf("command must be in the form '!bd <action> <arg1> <arg2>'")
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
		log.Errorf("Failed to get the default channel from the database.")
		return
	}
	server, err := GetServerID(database)
	if err != nil {
		log.Errorf("Failed to get the server id from the database.")
		return
	}
	birthdays, err := CheckForBirthdaysInDatabase(database, time.Now())
	if err != nil {
		log.Errorf("Failed to get todays birthdays from the database.")
		return
	}
	for _, b := range birthdays {
		message := fmt.Sprintf("Happy Birthday <@%s>!!! :partying_face:", b)
		utils.LogAndSend(s, channel, server, message, nil)
	}
}

func (d *DiscordBot) AddBirthday(command *Command) {
	if command.ID == "" || command.DateTime == "" {
		message := "Error parsing command: command must be in the form '!bd <action> <arg1> <arg2>'"
		utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
		return
	}
	user := utils.GetIDFromMention(command.ID)
	b, id := utils.IsUser(user, d.session, command.Server)
	if !b {
		message := fmt.Sprintf("Invalid user '%s'.", user)
		utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
		return
	}
	if !utils.IsValidDate(command.DateTime) {
		message := fmt.Sprintf("Invalid date '%s'.", command.DateTime)
		utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
		return
	}
	var fullDate string
	// account for leap years
	if command.DateTime == "29/02" {
		fullDate = fmt.Sprintf("%s/00 00:00:00 AM", command.DateTime) // Only care about the information relevant to the YearDay()
	} else {
		fullDate = fmt.Sprintf("%s/01 00:00:00 AM", command.DateTime) // Adjust year based on whether it is a leap year
	}
	datetime, _ := time.Parse(utils.FullDateFormat, fullDate) // We know at this point that the date is valid
	err := AddBirthdayToDatabase(command.Database, id, datetime)
	if err != nil {
		message := fmt.Sprintf("Error adding birthday to database: %s.", err.Error())
		utils.LogAndSend(d.session, command.Channel, command.Server, message, err)
		return
	}
	message := fmt.Sprintf("Successfully set birthday for <@!%s> to %s %s.", id, datetime.Month(), utils.AddNumSuffix(datetime.Day()))
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
		message := fmt.Sprintf("Error retrieving birthdays from database: %s.", err.Error())
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
	var firstBirthdayDate time.Time
	var firstBirthdayID string

	for i, birthday := range birthdays {
		t := birthday.Date
		date := int(t.YearDay())
		// to reduce logic for wrap around
		if i == 0 {
			firstBirthdayDate = t
			firstBirthdayID = birthday.ID
		}
		if date > today {
			message := fmt.Sprintf("The next person to have their birthday is <@%s> in %d days on %s %s.", birthday.ID, (date - today), t.Month(), utils.AddNumSuffix(t.Day()))
			utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
			return
		}
	}
	// catch any dates that have wrapped round (will only reach if no birthdays after today)
	message := fmt.Sprintf("The next person to have their birthday is <@%s> in %d days on %s %s.", firstBirthdayID, (utils.DaysInThisYear() - today + int(firstBirthdayDate.YearDay())), firstBirthdayDate.Month(), utils.AddNumSuffix(firstBirthdayDate.Day()))
	utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
}

func (d *DiscordBot) WhenBirthday(command *Command) {
	if command.ID == "" {
		message := "Error parsing command: command must be in the form '!bd <action> <arg1> <arg2>'"
		utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
		return
	}
	user := utils.GetIDFromMention(command.ID)
	b, id := utils.IsUser(user, d.session, command.Server)
	if !b {
		message := fmt.Sprintf("Invalid user '%s'.", user)
		utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
		return
	}
	var message string
	birthday, err := CheckForUsersBirthdayInDatabase(command.Database, id)
	if err != nil {
		message := fmt.Sprintf("Error checking for users birthday: %s.", err.Error())
		utils.LogAndSend(d.session, command.Channel, command.Server, message, err)
		return
	}
	if birthday == time.Unix(0, 0) {
		message = fmt.Sprintf("<@%s>'s birthday not in database.", id)
	} else {
		message = fmt.Sprintf("<@%s>'s birthday is on %s %s.", id, birthday.Month(), utils.AddNumSuffix(birthday.Day()))
	}
	utils.LogAndSend(d.session, command.Channel, command.Server, message, nil)
}

func (d *DiscordBot) Help(command *Command) {
	utils.LogAndSend(d.session, command.Channel, command.Server, helpMessage, nil)
}
