package utils

import (
	"fmt"

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

func LogAndSend(session *discordgo.Session, channelID, message string, err error) {
	if err != nil {
		log.Error(err)
	}
	log.Info(fmt.Sprintf("Sending message to channel %s: '%s'", channelID, message))
	session.ChannelMessageSend(channelID, message)
}
