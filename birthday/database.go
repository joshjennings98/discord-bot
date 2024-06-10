package commands

import (
	"context"
	"fmt"
	"time"

	commonerrors "github.com/joshjennings98/discord-bot/errors"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var BirthdaysDatabase *mongo.Database

const (
	BirthdayDatabaseName = "databases"
	Timeout              = 5 * time.Second
)

func CheckForBirthdaysInDatabase(database string, t time.Time) (birthdays []string, err error) {
	server_db := BirthdaysDatabase.Collection(BirthdayDatabaseName)
	ctx, _ := context.WithTimeout(context.Background(), Timeout)

	var item ServerContent
	if err1 := server_db.FindOne(ctx, bson.M{"server": database}).Decode(&item); err1 != nil {
		err = commonerrors.ErrCannotOpenDatabase
		return
	}

	todayMonth := t.Month()
	todayDay := t.Day()
	for _, birthdayItem := range item.Birthdays {
		birthday := birthdayItem.Date
		if birthday.Month() == todayMonth && birthday.Day() == todayDay {
			birthdays = append(birthdays, birthdayItem.ID)
		}
	}
	return
}

func CheckForUsersBirthdayInDatabase(database, userID string) (birthday time.Time, err error) {
	server_db := BirthdaysDatabase.Collection(BirthdayDatabaseName)
	ctx, _ := context.WithTimeout(context.Background(), Timeout)

	var item ServerContent
	if err1 := server_db.FindOne(ctx, bson.M{"server": database}).Decode(&item); err1 != nil {
		err = commonerrors.ErrCannotOpenDatabase
		return
	}

	for _, birthdayItem := range item.Birthdays {
		if birthdayItem.ID == userID {
			birthday = birthdayItem.Date
			return
		}
	}
	birthday = time.Unix(0, 0)
	return
}

type ServerContent struct {
	Id        primitive.ObjectID `bson:"_id,omitempty"`
	Server    string             `bson:"server,omitempty"`
	Channel   string             `bson:"channel,omitempty"`
	Timezone  string             `bson:"timezone,omitempty"`
	Time      string             `bson:"time,omitempty"`
	Birthdays []Birthday         `bson:"birthdays,omitempty"`
}

type ServerKeys struct {
	Id        primitive.ObjectID `bson:"_id,omitempty"`
	IsKeyList bool               `bson:"isKeyList,omitempty"`
	Keys      []string           `bson:"keys,omitempty"`
}

func AddBirthdayToDatabase(database, id string, date time.Time) (err error) {
	server_db := BirthdaysDatabase.Collection(BirthdayDatabaseName)
	ctx, _ := context.WithTimeout(context.Background(), Timeout)

	var item ServerContent

	if err = server_db.FindOne(ctx, bson.M{"server": database}).Decode(&item); err != nil {
		return commonerrors.ErrCannotOpenDatabase
	}

	existsInDB := false
	birthdays := item.Birthdays
	for i := range birthdays {
		if birthdays[i].ID == id {
			birthdays[i].Date = date
			existsInDB = true
		}
	}
	if !existsInDB {
		birthdays = append(birthdays, Birthday{
			ID:   id,
			Date: date,
		})
	}

	if _, err = server_db.UpdateOne(ctx,
		bson.M{"server": database},
		bson.D{{"$set", bson.D{{"birthdays", birthdays}}}}); err != nil {
		return commonerrors.ErrCannotInsertIntoDB
	}

	log.Info(fmt.Sprintf("Added Birthday for %s on %s %d\n", id, date.Month().String(), date.Day()))
	return err
}

func GetBirthdaysFromDatabase(database string) (birthdays Birthdays, err error) {
	serverContent, err1 := getServerContent(database)
	if err1 != nil {
		err = err1
		return
	}
	return serverContent.Birthdays, nil
}

func SetupBirthdayDatabase(database, defaultChannel, timezone, server, interval string) (err error) {
	server_db := BirthdaysDatabase.Collection(BirthdayDatabaseName)
	ctx, _ := context.WithTimeout(context.Background(), Timeout)

	var item ServerKeys
	if err = server_db.FindOne(ctx, bson.M{"isKeyList": true}).Decode(&item); err != nil {
		return commonerrors.ErrCannotOpenDatabase
	}
	keys := item.Keys
	for i := range keys {
		if keys[i] == server {
			if _, err = server_db.UpdateOne(ctx,
				bson.M{"server": server},
				bson.D{{"$set", bson.D{
					{Key: "channel", Value: defaultChannel},
					{Key: "timezone", Value: timezone},
					{Key: "time", Value: interval}}}}); err != nil {
				return commonerrors.ErrCannotUpdateDB
			}
			return
		}
	}

	keys = append(keys, server)

	if _, err = server_db.UpdateOne(ctx,
		bson.M{"isKeyList": true},
		bson.D{{"$set", bson.D{{"keys", keys}}}}); err != nil {
		return commonerrors.ErrCannotInsertIntoDB
	}

	_, err = server_db.InsertOne(ctx, bson.D{
		{Key: "server", Value: server},
		{Key: "channel", Value: defaultChannel},
		{Key: "timezone", Value: timezone},
		{Key: "time", Value: interval},
		{Key: "birthdays", Value: []Birthday{}},
	})

	if err != nil {
		log.Error(err)
		return commonerrors.ErrCannotOpenDatabase
	}

	log.Info("Database Setup Done")
	return nil
}

func GetDefaultChannel(database string) (channel string, err error) {
	serverContent, err1 := getServerContent(database)
	if err1 != nil {
		err = err1
		return
	}
	return serverContent.Channel, nil
}

func GetServerID(database string) (server string, err error) {
	serverContent, err1 := getServerContent(database)
	if err1 != nil {
		err = err1
		return
	}
	return serverContent.Server, nil
}

func GetTimezone(database string) (tz string, err error) {
	serverContent, err1 := getServerContent(database)
	if err1 != nil {
		err = err1
		return
	}
	return serverContent.Timezone, nil
}

func GetTimeInterval(database string) (interval string, err error) {
	serverContent, err1 := getServerContent(database)
	if err1 != nil {
		err = err1
		return
	}
	return serverContent.Time, nil
}

func getServerContent(database string) (value ServerContent, err error) {
	server_db := BirthdaysDatabase.Collection(BirthdayDatabaseName)
	ctx, _ := context.WithTimeout(context.Background(), Timeout)

	if err1 := server_db.FindOne(ctx, bson.M{"server": database}).Decode(&value); err1 != nil {
		err = commonerrors.ErrCannotOpenDatabase
		return
	}
	return
}

func GetServerKeys() (keys []string, err error) {
	server_db := BirthdaysDatabase.Collection(BirthdayDatabaseName)
	ctx, _ := context.WithTimeout(context.Background(), Timeout)

	var item ServerKeys
	if err1 := server_db.FindOne(ctx, bson.M{"isKeyList": true}).Decode(&item); err1 != nil {
		err = commonerrors.ErrCannotOpenDatabase
		return
	}
	keys = item.Keys
	return
}
