package utils

import (
	"fmt"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

func IsUser(user string, s *discordgo.Session, serverID string) (b bool, id string) {
	_, err := s.GuildMember(serverID, user)

	if err != nil {
		return false, user
	}

	return true, user
}

func GetIDFromMention(user string) string {
	return RemoveChars(user, []string{"<", ">", "@", "!"})
}

func LogAndSend(session *discordgo.Session, channelID, serverID, message string, err error) {
	if err != nil {
		log.Error(err)
	}
	log.Info(fmt.Sprintf("Sending message to channel %s on server %s: '%s'", channelID, serverID, message))
	session.ChannelMessageSend(channelID, message)
}

func DatabaseFromServerID(server string) string {
	return fmt.Sprintf("database_%s.db", server)
}

func SnowflakeToTimestamp(snowflake string) (timestamp time.Time, err error) {
	t, err := strconv.ParseInt(snowflake[:22], 2, 64)
	if err != nil {
		return timestamp, err
	}
	t += 1420070400000               // discord epoch (unix timestamp in ms)
	return time.Unix(t/1000, 0), nil // convert ms to seconds
}
