package utils

import "github.com/bwmarrin/discordgo"

func IsUser(input string, s *discordgo.Session, serverID string) (b bool, id string) {
	user := RemoveChars(input, []string{"<", ">", "@", "!"})
	_, err := s.GuildMember(serverID, user)

	if err != nil {
		return false, user
	}

	return true, user
}
