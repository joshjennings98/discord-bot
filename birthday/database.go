package commands

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	log "github.com/sirupsen/logrus"
)

func CheckForBirthdaysInDatabase(database string, t time.Time) (birthdays []string, err error) {
	path := "databases/" + database
	if _, err1 := os.Stat(path); os.IsNotExist(err1) {
		err = fmt.Errorf("database doesn't exist, please run setup")
		return
	}
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
	if _, err1 := os.Stat(path); os.IsNotExist(err1) {
		err = fmt.Errorf("database doesn't exist, please run setup")
		return
	}
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return time.Unix(0, 0), fmt.Errorf("could not open db, %v", err)
	}
	defer db.Close()

	log.Info(fmt.Sprintf("Checking for %s's birthdays in %s", userID, database))
	err = db.View(func(tx *bolt.Tx) error {
		bd := string(tx.Bucket([]byte("DB")).Bucket([]byte("BIRTHDAYS")).Get([]byte(userID)))
		if bd == "" {
			err = fmt.Errorf("ID not in database: %s", userID)
			return err
		}
		i, err := strconv.ParseInt(bd, 10, 64)
		if err != nil {
			return err
		}
		birthday = time.Unix(i, 0)
		return nil
	})
	return
}

func AddBirthdayToDatabase(database, id string, date time.Time) (err error) {
	path := "databases/" + database
	if _, err1 := os.Stat(path); os.IsNotExist(err1) {
		err = fmt.Errorf("database doesn't exist, please run setup")
		return
	}
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
	if _, err1 := os.Stat(path); os.IsNotExist(err1) {
		err = fmt.Errorf("database doesn't exist, please run setup")
		return
	}

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
	if _, err1 := os.Stat(path); os.IsNotExist(err1) {
		err = fmt.Errorf("database doesn't exist, please run setup")
		return
	}
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
	if _, err1 := os.Stat(path); os.IsNotExist(err1) {
		err = fmt.Errorf("database doesn't exist, please run setup")
		return
	}
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
	if _, err1 := os.Stat(path); os.IsNotExist(err1) {
		err = fmt.Errorf("database doesn't exist, please run setup")
		return
	}
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
	if _, err1 := os.Stat(path); os.IsNotExist(err1) {
		err = fmt.Errorf("database doesn't exist, please run setup")
		return
	}
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
