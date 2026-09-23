package notifications

import (
	"fmt"
	"html"
	"strings"

	"gopkg.in/telebot.v4/internal/storage/postgres"
)

func EventMentions(members []postgres.GroupMemberView) string {
	lines := make([]string, 0, len(members))
	for _, member := range members {
		name := strings.TrimSpace(member.RealName)
		if name == "" {
			name = strings.TrimSpace(member.FirstName + " " + member.LastName)
		}
		if name == "" && member.Username != "" {
			name = "@" + member.Username
		}
		lines = append(lines, Mention(member.UserTelegramID, name))
	}
	return strings.Join(lines, ",\n")
}

func PollInvitation(question string, members []postgres.GroupMemberView) string {
	if len(members) == 0 {
		return ""
	}
	return fmt.Sprintf("Открыта запись: %s\n\n%s\n\nОтметьте свой ответ в опросе выше.", html.EscapeString(question), EventMentions(members))
}

func Announcement(message string, members []postgres.GroupMemberView) string {
	result := html.EscapeString(message)
	if len(members) != 0 {
		result += "\n\n" + EventMentions(members)
	}
	return result
}
