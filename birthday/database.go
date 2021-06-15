package commands

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	commonerrors "github.com/joshjennings98/discord-bot/errors"
	log "github.com/sirupsen/logrus"
)

func CheckForBirthdaysInDatabase(database string, t time.Time) (birthdays []string, err error) {
	db, err := openDatabase(database, 0600, nil)
	if err != nil {
		return nil, commonerrors.ErrCannotOpenDatabase
	}
	defer db.Close()

	birthdays = []string{}
	date := t.YearDay()
	log.Info("Checking for today's birthdays")
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("DB")).Bucket([]byte("BIRTHDAYS"))
		b.ForEach(func(k, v []byte) (err1 error) {
			i, err1 := strconv.ParseInt(string(v), 10, 64)
			if err1 != nil {
				err = commonerrors.ErrCannotParse
				return
			}
			if time.Unix(i, 0).YearDay() == date {
				birthdays = append(birthdays, string(k))
			}
			return nil
		})
		return nil
	})
	return
}

func CheckForUsersBirthdayInDatabase(database, userID string) (birthday time.Time, err error) {
	db, err := openDatabase(database, 0600, nil)
	if err != nil {
		return time.Unix(0, 0), commonerrors.ErrCannotOpenDatabase
	}
	defer db.Close()

	log.Info(fmt.Sprintf("Checking for %s's birthdays in %s", userID, database))
	err = db.View(func(tx *bolt.Tx) error {
		bd := string(tx.Bucket([]byte("DB")).Bucket([]byte("BIRTHDAYS")).Get([]byte(userID)))
		if bd == "" {
			err = commonerrors.ErrIDNotInDatabase
			return err
		}
		i, err := strconv.ParseInt(bd, 10, 64)
		if err != nil {
			return commonerrors.ErrCannotParse
		}
		birthday = time.Unix(i, 0)
		return nil
	})
	return
}

func AddBirthdayToDatabase(database, id string, date time.Time) (err error) {
	db, err := openDatabase(database, 0600, nil)
	if err != nil {
		return commonerrors.ErrCannotOpenDatabase
	}
	defer db.Close()

	dateString := strconv.FormatInt(date.Unix(), 10)
	err = db.Update(func(tx *bolt.Tx) error {
		err := tx.Bucket([]byte("DB")).Bucket([]byte("BIRTHDAYS")).Put([]byte(id), []byte(dateString))
		if err != nil {
			return commonerrors.ErrCannotInsertIntoDB
		}
		return nil
	})
	log.Info(fmt.Sprintf("Added Birthday for %s on %s %d\n", id, date.Month().String(), date.Day()))
	return err
}

func GetBirthdaysFromDatabase(database string) (birthdays Birthdays, err error) {
	db, err := openDatabase(database, 0600, nil)
	if err != nil {
		return []Birthday{}, commonerrors.ErrCannotOpenDatabase
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("DB")).Bucket([]byte("BIRTHDAYS"))
		b.ForEach(func(k, v []byte) (err error) {
			numSecondsSince1970, err1 := strconv.ParseInt(string(v), 10, 64)
			if err1 != nil {
				err = commonerrors.ErrCannotParse
				return
			}
			birthdays = append(birthdays, Birthday{ID: string(k), Date: time.Unix(numSecondsSince1970, 0)})
			return nil
		})
		return nil
	})
	return
}

func SetupBirthdayDatabase(database, defaultChannel, timezone, server, interval string) (err error) {
	db, err := bolt.Open(database, 0600, nil)
	if err != nil {
		return commonerrors.ErrCannotOpenDatabase
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
		return err
	}
	log.Info("Database Setup Done")
	return nil
}

func GetDefaultChannel(database string) (channel string, err error) {
	return getFromDatabase(database, "default channel")
}

func GetServerID(database string) (server string, err error) {
	return getFromDatabase(database, "ServerID")
}

func GetTimezone(database string) (tz string, err error) {
	return getFromDatabase(database, "timezone")
}

func GetTimeInterval(database string) (interval string, err error) {
	return getFromDatabase(database, "time interval")
}

// opens database without creating it if it is missing
func openDatabase(database string, mode os.FileMode, options *bolt.Options) (db *bolt.DB, err error) {
	if _, openFileError := os.Stat(database); os.IsNotExist(openFileError) {
		err = commonerrors.ErrDatabaseNotExist
		return
	}
	return bolt.Open(database, mode, options)
}

func getFromDatabase(database, key string) (value string, err error) {
	db, err := openDatabase(database, 0600, nil)
	if err != nil {
		return "", commonerrors.ErrCannotOpenDatabase
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("DB"))
		value = string(b.Get([]byte(key)))
		return nil
	})
	return
}
