package utils

import "github.com/bwmarrin/discordgo"

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
