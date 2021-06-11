package commands

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	"github.com/bwmarrin/discordgo"
)

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

func CheckForBirthdayInDatabase(dbPath string, t time.Time) (birthdays []string, err error) {
	db, err := bolt.Open(dbPath, 0600, nil)
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

func AddBirthdayToDatabase(dbPath string, id string, date time.Time) error {
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

func SetupBirthdayDatabase(dbPath string) (err error) {
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

func WishHappyBirthday(s string, session *discordgo.Session, cfg BotConfiguration) {
	birthdays, _ := CheckForBirthdayInDatabase(s, time.Now())
	for _, b := range birthdays {
		session.ChannelMessageSend(cfg.Channel, fmt.Sprintf("Happy Birthday <@%s>!!! :partying_face:", b))
	}
}

func CheckTodaysBirthdays(s string, session *discordgo.Session, cfg BotConfiguration) {
	birthdays, _ := CheckForBirthdayInDatabase(s, time.Now())
	for _, b := range birthdays {
		session.ChannelMessageSend(cfg.Channel, fmt.Sprintf("<@%s> has their birthday today :smile:", b))
	}
	if len(birthdays) == 0 {
		session.ChannelMessageSend(cfg.Channel, "Nobody has their birthday today :cry:")
	}
}

func NextBirthday(dbPath string, session *discordgo.Session, cfg BotConfiguration) (err error) {
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return fmt.Errorf("could not open db, %v", err)
	}
	defer db.Close()
	birthdays := Birthdays{}
	today := time.Now().YearDay()
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("DB")).Bucket([]byte("BIRTHDAYS"))
		b.ForEach(func(k, v []byte) error {
			birthdays = append(birthdays, Birthday{ID: string(k), Date: string(v)})
			return nil
		})
		return nil
	})
	if err != nil {
		return
	}
	if len(birthdays) == 0 {
		session.ChannelMessageSend(cfg.Channel, "There are no birthdays in the database.")
		return
	}
	sort.Sort(birthdays)
	for _, birthday := range birthdays {
		date, err := strconv.Atoi(birthday.Date)
		if err != nil {
			return err
		}
		if date > today {
			session.ChannelMessageSend(cfg.Channel, fmt.Sprintf("The next person to have their birthday is <@%s> in %d days.", birthday.ID, (date-today)))
			return nil
		}
	}
	date, err := strconv.Atoi(birthdays[0].Date)
	if err != nil {
		return err
	}
	// catch any dates that have wrapped round
	session.ChannelMessageSend(cfg.Channel, fmt.Sprintf("The next person to have their birthday is <@%s> in %d days.", birthdays[0].ID, (365-today+date)))
	return
}
