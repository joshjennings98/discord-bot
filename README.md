# BirthdayBot3000

A discord bot written in Go that uses MongoDB and Heroku to host a bot that keeps track of users birthdays.

## Usage

- `!bd add <user> <dd/mm>` - add a new birthday to the database
- `!bd next` - see who is having their birthday next
- `!bd today` - check who is having their birthday today
- `!bd when <user>` - see a specific users birthday
- `!bd setup <timezone/tz> <hour 0..23>` - run the setup
- `!bd help` - see this help message

## Note

The channel used for the birthday alert is the channel that `setup` is called from.