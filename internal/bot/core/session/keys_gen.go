//go:build ignore

package main

import (
	"bytes"
	"go/format"
	"log"
	"os"
	"text/template"
)

type KeyDef struct {
	Name    string
	Type    string
	Doc     string
	Persist bool
}

const tmpl = `// Code generated by go generate; DO NOT EDIT.
package session

import (
    "time"
    
    "github.com/robalyx/rotector/internal/common/storage/database/types"
    "github.com/robalyx/rotector/internal/common/storage/database/types/enum"
    "github.com/robalyx/rotector/internal/common/client/ai"
	"github.com/robalyx/rotector/internal/common/queue"
	"github.com/robalyx/rotector/internal/worker/core"
	apiTypes "github.com/jaxron/roapi.go/pkg/api/types"
)

var (
    {{ range .Keys }}
    // {{ .Doc }}
    {{ .Name }} = NewKey[{{ .Type }}]("{{ .Name }}", {{ .Persist }})
    {{- end }}

    {{ range .BufferKeys }}
    // {{ .Doc }}
    {{ .Name }} = NewBufferKey("{{ .Name }}", {{ .Persist }})
    {{- end }}
)
`

func main() {
	keys := []KeyDef{
		// Required keys
		{Name: "UserID", Type: "uint64", Doc: "UserID stores the user ID", Persist: false},
		{Name: "IsGuildOwner", Type: "bool", Doc: "IsGuildOwner indicates if the user is a guild owner", Persist: false},
		{Name: "ExistingSessions", Type: "[]Info", Doc: "ExistingSessions stores information about user's existing sessions", Persist: false},
		{Name: "LastUsed", Type: "time.Time", Doc: "LastUsed stores when the session was last accessed", Persist: true},

		// Navigation related keys
		{Name: "MessageID", Type: "uint64", Doc: "MessageID stores the ID of the current message", Persist: true},
		{Name: "CurrentPage", Type: "string", Doc: "CurrentPage stores the current page identifier", Persist: true},
		{Name: "PreviousPages", Type: "[]string", Doc: "PreviousPages stores the navigation history", Persist: true},

		// Pagination related keys
		{Name: "PaginationPage", Type: "int", Doc: "PaginationPage stores the current pagination page number", Persist: true},
		{Name: "PaginationOffset", Type: "int", Doc: "PaginationOffset stores the starting offset", Persist: true},
		{Name: "PaginationTotalItems", Type: "int", Doc: "PaginationTotalItems stores the total number of items", Persist: true},
		{Name: "PaginationTotalPages", Type: "int", Doc: "PaginationTotalPages stores the total number of pages", Persist: true},
		{Name: "PaginationHasNextPage", Type: "bool", Doc: "PaginationHasNextPage indicates if there is a next page", Persist: true},
		{Name: "PaginationHasPrevPage", Type: "bool", Doc: "PaginationHasPrevPage indicates if there is a previous page", Persist: true},
		{Name: "PaginationIsStreaming", Type: "bool", Doc: "PaginationIsStreaming indicates if image streaming is active", Persist: true},

		// Statistics related keys
		{Name: "StatsIsRefreshed", Type: "bool", Doc: "StatsIsRefreshed indicates if the data has been refreshed", Persist: true},
		{Name: "StatsUserCounts", Type: "*types.UserCounts", Doc: "StatsUserCounts stores user statistics", Persist: true},
		{Name: "StatsGroupCounts", Type: "*types.GroupCounts", Doc: "StatsGroupCounts stores group statistics", Persist: true},
		{Name: "StatsActiveUsers", Type: "[]uint64", Doc: "StatsActiveUsers stores the list of active reviewers", Persist: true},
		{Name: "StatsVotes", Type: "*types.VoteAccuracy", Doc: "StatsVotes stores a user's voting statistics", Persist: true},

		// Status related keys
		{Name: "StatusWorkers", Type: "[]core.Status", Doc: "StatusWorkers stores worker status information", Persist: false},

		// Settings related keys
		{Name: "SettingName", Type: "string", Doc: "SettingName stores the name of the current setting", Persist: true},
		{Name: "SettingType", Type: "string", Doc: "SettingType stores the type of the current setting", Persist: true},
		{Name: "SettingValue", Type: "*Setting", Doc: "SettingValue stores the setting value", Persist: true},
		{Name: "SettingDisplay", Type: "string", Doc: "SettingDisplay stores the display value of the setting", Persist: true},
		{Name: "SettingCustomID", Type: "string", Doc: "SettingCustomID stores the custom identifier", Persist: true},

		// User related keys
		{Name: "UserTarget", Type: "*types.ReviewUser", Doc: "UserTarget stores the currently selected user", Persist: true},
		{Name: "UserFlaggedFriends", Type: "map[uint64]*types.ReviewUser", Doc: "UserFlaggedFriends stores flagged friends", Persist: true},
		{Name: "UserFlaggedGroups", Type: "map[uint64]*types.ReviewGroup", Doc: "UserFlaggedGroups stores flagged groups", Persist: true},
		{Name: "UserFriends", Type: "[]*apiTypes.ExtendedFriend", Doc: "UserFriends stores the user's friend list", Persist: false},
		{Name: "UserPresences", Type: "map[uint64]*apiTypes.UserPresenceResponse", Doc: "UserPresences stores friend presence information", Persist: false},
		{Name: "UserGroups", Type: "[]*apiTypes.UserGroupRoles", Doc: "UserGroups stores the list of groups", Persist: false},
		{Name: "UserOutfits", Type: "[]*apiTypes.Outfit", Doc: "UserOutfits stores user outfits", Persist: false},
		{Name: "UserReviewHistory", Type: "[]uint64", Doc: "UserReviewHistory stores IDs of previously reviewed users", Persist: true},
		{Name: "UserReviewHistoryIndex", Type: "int", Doc: "UserReviewHistoryIndex stores the current position in the review history", Persist: true},

		// Group related keys
		{Name: "GroupTarget", Type: "*types.ReviewGroup", Doc: "GroupTarget stores the currently selected group", Persist: true},
		{Name: "GroupInfo", Type: "*apiTypes.GroupResponse", Doc: "GroupInfo stores additional group information", Persist: true},
		{Name: "GroupFlaggedMembersCount", Type: "int", Doc: "GroupFlaggedMembersCount stores the total number of flagged members in the current group", Persist: true},
		{Name: "GroupPageFlaggedMembers", Type: "map[uint64]*types.ReviewUser", Doc: "GroupPageFlaggedMembers stores flagged member details for the current page", Persist: false},
		{Name: "GroupPageFlaggedMemberIDs", Type: "[]uint64", Doc: "GroupPageFlaggedMemberIDs stores flagged member IDs for the current page", Persist: false},
		{Name: "GroupReviewHistory", Type: "[]uint64", Doc: "GroupReviewHistory stores IDs of previously reviewed groups", Persist: true},
		{Name: "GroupReviewHistoryIndex", Type: "int", Doc: "GroupReviewHistoryIndex stores the current position in the review history", Persist: true},

		// Review related keys
		{Name: "ReasonsChanged", Type: "bool", Doc: "ReasonsChanged indicates if reasons have been modified", Persist: true},
		{Name: "OriginalUserReasons", Type: "types.Reasons[enum.UserReasonType]", Doc: "OriginalUserReasons stores the initial user reasons", Persist: true},
		{Name: "OriginalGroupReasons", Type: "types.Reasons[enum.GroupReasonType]", Doc: "OriginalGroupReasons stores the initial group reasons", Persist: true},
		{Name: "SelectedReasonType", Type: "string", Doc: "SelectedReasonType stores the currently selected reason type for modal handling", Persist: true},
		{Name: "ReviewLogs", Type: "[]*types.ActivityLog", Doc: "ReviewLogs stores the current review logs", Persist: true},
		{Name: "ReviewLogsHasMore", Type: "bool", Doc: "ReviewLogsHasMore indicates if there are more logs available", Persist: true},
		{Name: "ReviewComments", Type: "[]*types.Comment", Doc: "ReviewComments stores comments for the current user or group", Persist: true},

		// Discord user lookup related keys
		{Name: "DiscordUserLookupID", Type: "uint64", Doc: "DiscordUserLookupID stores the Discord user ID being looked up", Persist: true},
		{Name: "DiscordUserLookupName", Type: "string", Doc: "DiscordUserLookupName stores the Discord username", Persist: true},
		{Name: "DiscordUserGuilds", Type: "[]*types.UserGuildInfo", Doc: "DiscordUserGuilds stores a Discord user's guild memberships", Persist: true},
		{Name: "DiscordUserGuildNames", Type: "map[uint64]string", Doc: "DiscordUserGuildNames stores guild names for a Discord user", Persist: true},
		{Name: "DiscordUserMessageSummary", Type: "*types.InappropriateUserSummary", Doc: "DiscordUserMessageSummary stores the user's inappropriate message summary", Persist: true},
		{Name: "DiscordUserTotalGuilds", Type: "int", Doc: "DiscordUserTotalGuilds stores the total number of flagged guilds for the user", Persist: true},
		{Name: "GuildLookupCursor", Type: "*types.GuildCursor", Doc: "GuildLookupCursor stores the current guild lookup cursor", Persist: true},
		{Name: "GuildLookupNextCursor", Type: "*types.GuildCursor", Doc: "GuildLookupNextCursor stores the next guild lookup cursor", Persist: true},
		{Name: "GuildLookupPrevCursors", Type: "[]*types.GuildCursor", Doc: "GuildLookupPrevCursors stores previous guild lookup cursors", Persist: true},

		// Discord user messages related keys
		{Name: "DiscordUserMessages", Type: "[]*types.InappropriateMessage", Doc: "DiscordUserMessages stores the current page of messages", Persist: true},
		{Name: "DiscordUserMessageCursor", Type: "*types.MessageCursor", Doc: "DiscordUserMessageCursor stores the current message cursor", Persist: true},
		{Name: "DiscordUserMessageNextCursor", Type: "*types.MessageCursor", Doc: "DiscordUserMessageNextCursor stores the next message cursor", Persist: true},
		{Name: "DiscordUserMessagePrevCursors", Type: "[]*types.MessageCursor", Doc: "DiscordUserMessagePrevCursors stores previous message cursors", Persist: true},
		{Name: "DiscordUserMessageGuildID", Type: "uint64", Doc: "DiscordUserMessageGuildID stores the currently selected guild ID for messages", Persist: true},
		{Name: "DiscordUserDataRedacted", Type: "bool", Doc: "DiscordUserDataRedacted indicates if the user has requested data deletion", Persist: true},
		{Name: "DiscordUserMessageGuilds", Type: "map[uint64]struct{}", Doc: "DiscordUserMessageGuilds stores a map of guild IDs where the user has inappropriate messages", Persist: true},

		// Chat related keys
		{Name: "ChatHistory", Type: "ai.ChatHistory", Doc: "ChatHistory stores the conversation history", Persist: true},
		{Name: "ChatContext", Type: "string", Doc: "ChatContext stores chat context information", Persist: true},

		// Log related keys
		{Name: "LogActivities", Type: "[]*types.ActivityLog", Doc: "LogActivities stores activity logs", Persist: true},
		{Name: "LogCursor", Type: "*types.LogCursor", Doc: "LogCursor stores the current log cursor", Persist: true},
		{Name: "LogNextCursor", Type: "*types.LogCursor", Doc: "LogNextCursor stores the next log cursor", Persist: true},
		{Name: "LogPrevCursors", Type: "[]*types.LogCursor", Doc: "LogPrevCursors stores previous log cursors", Persist: true},
		{Name: "LogFilterGuildID", Type: "uint64", Doc: "LogFilterGuildID stores guild ID filter", Persist: true},
		{Name: "LogFilterDiscordID", Type: "uint64", Doc: "LogFilterDiscordID stores Discord ID filter", Persist: true},
		{Name: "LogFilterUserID", Type: "uint64", Doc: "LogFilterUserID stores user ID filter", Persist: true},
		{Name: "LogFilterGroupID", Type: "uint64", Doc: "LogFilterGroupID stores group ID filter", Persist: true},
		{Name: "LogFilterReviewerID", Type: "uint64", Doc: "LogFilterReviewerID stores reviewer ID filter", Persist: true},
		{Name: "LogFilterActivityType", Type: "enum.ActivityType", Doc: "LogFilterActivityType stores activity type filter", Persist: true},
		{Name: "LogFilterActivityCategory", Type: "string", Doc: "LogFilterActivityCategory stores the currently selected activity category", Persist: true},
		{Name: "LogFilterDateRangeStart", Type: "time.Time", Doc: "LogFilterDateRangeStart stores start date filter", Persist: true},
		{Name: "LogFilterDateRangeEnd", Type: "time.Time", Doc: "LogFilterDateRangeEnd stores end date filter", Persist: true},

		// Queue related keys
		{Name: "QueueUser", Type: "uint64", Doc: "QueueUser stores the queued user", Persist: true},
		{Name: "QueueStatus", Type: "queue.Status", Doc: "QueueStatus stores the queue status", Persist: false},
		{Name: "QueuePriority", Type: "queue.Priority", Doc: "QueuePriority stores the queue priority", Persist: false},
		{Name: "QueuePosition", Type: "int", Doc: "QueuePosition stores the queue position", Persist: false},
		{Name: "QueueHighCount", Type: "int", Doc: "QueueHighCount stores high priority queue count", Persist: false},
		{Name: "QueueNormalCount", Type: "int", Doc: "QueueNormalCount stores normal priority queue count", Persist: false},
		{Name: "QueueLowCount", Type: "int", Doc: "QueueLowCount stores low priority queue count", Persist: false},

		// Appeal related keys
		{Name: "AppealList", Type: "[]*types.FullAppeal", Doc: "AppealList stores the current page of appeals", Persist: true},
		{Name: "AppealSelected", Type: "*types.FullAppeal", Doc: "AppealSelected stores the currently selected appeal", Persist: true},
		{Name: "AppealMessages", Type: "[]*types.AppealMessage", Doc: "AppealMessages stores messages for the current appeal", Persist: true},
		{Name: "AppealCursor", Type: "*types.AppealTimeline", Doc: "AppealCursor stores the current cursor position", Persist: true},
		{Name: "AppealNextCursor", Type: "*types.AppealTimeline", Doc: "AppealNextCursor stores the next cursor position", Persist: true},
		{Name: "AppealPrevCursors", Type: "[]*types.AppealTimeline", Doc: "AppealPrevCursors stores previous cursor positions", Persist: true},
		{Name: "AppealType", Type: "enum.AppealType", Doc: "AppealType stores the type of appeal being created", Persist: true},
		{Name: "AppealRejectedCount", Type: "int", Doc: "AppealRejectedCount stores the total number of rejected appeals for the user", Persist: true},

		// Verify related keys
		{Name: "VerifyUserID", Type: "uint64", Doc: "VerifyUserID stores the user ID being verified", Persist: true},
		{Name: "VerifyReason", Type: "string", Doc: "VerifyReason stores the verification reason", Persist: true},
		{Name: "VerifyCode", Type: "string", Doc: "VerifyCode stores the verification code", Persist: true},

		// CAPTCHA related keys
		{Name: "CaptchaAnswer", Type: "string", Doc: "CaptchaAnswer stores the CAPTCHA answer", Persist: true},

		// Admin related keys
		{Name: "AdminAction", Type: "string", Doc: "AdminAction stores the current admin action", Persist: true},
		{Name: "AdminActionID", Type: "string", Doc: "AdminActionID stores the admin action ID", Persist: true},
		{Name: "AdminReason", Type: "string", Doc: "AdminReason stores the admin action reason", Persist: true},
		{Name: "AdminBanReason", Type: "enum.BanReason", Doc: "AdminBanReason stores the ban reason", Persist: true},
		{Name: "AdminBanExpiry", Type: "*time.Time", Doc: "AdminBanExpiry stores the ban expiry time", Persist: true},
		{Name: "AdminBanInfo", Type: "*types.DiscordBan", Doc: "AdminBanInfo stores ban information", Persist: true},

		// Leaderboard stats related keys
		{Name: "LeaderboardStats", Type: "[]*types.VoteAccuracy", Doc: "LeaderboardStats stores leaderboard statistics", Persist: true},
		{Name: "LeaderboardUsernames", Type: "map[uint64]string", Doc: "LeaderboardUsernames stores usernames for the leaderboard", Persist: true},
		{Name: "LeaderboardCursor", Type: "*types.LeaderboardCursor", Doc: "LeaderboardCursor stores the current leaderboard cursor", Persist: true},
		{Name: "LeaderboardNextCursor", Type: "*types.LeaderboardCursor", Doc: "LeaderboardNextCursor stores the next leaderboard cursor", Persist: true},
		{Name: "LeaderboardPrevCursors", Type: "[]*types.LeaderboardCursor", Doc: "LeaderboardPrevCursors stores previous leaderboard cursors", Persist: true},
		{Name: "LeaderboardLastRefresh", Type: "time.Time", Doc: "LeaderboardLastRefresh stores the last refresh time", Persist: true},
		{Name: "LeaderboardNextRefresh", Type: "time.Time", Doc: "LeaderboardNextRefresh stores the next refresh time", Persist: true},

		// Reviewer stats related keys
		{Name: "ReviewerStats", Type: "map[uint64]*types.ReviewerStats", Doc: "ReviewerStats stores reviewer statistics", Persist: true},
		{Name: "ReviewerUsernames", Type: "map[uint64]string", Doc: "ReviewerUsernames stores usernames for reviewers", Persist: true},
		{Name: "ReviewerStatsCursor", Type: "*types.ReviewerStatsCursor", Doc: "ReviewerStatsCursor stores the current reviewer stats cursor", Persist: true},
		{Name: "ReviewerStatsNextCursor", Type: "*types.ReviewerStatsCursor", Doc: "ReviewerStatsNextCursor stores the next reviewer stats cursor", Persist: true},
		{Name: "ReviewerStatsPrevCursors", Type: "[]*types.ReviewerStatsCursor", Doc: "ReviewerStatsPrevCursors stores previous reviewer stats cursors", Persist: true},
		{Name: "ReviewerStatsLastRefresh", Type: "time.Time", Doc: "ReviewerStatsLastRefresh stores the last refresh time", Persist: true},
		{Name: "ReviewerStatsNextRefresh", Type: "time.Time", Doc: "ReviewerStatsNextRefresh stores the next refresh time", Persist: true},

		// Guild owner related keys
		{Name: "GuildScanType", Type: "string", Doc: "GuildScanType stores the type of guild scan selected (messages or servers)", Persist: true},
		{Name: "GuildStatsID", Type: "uint64", Doc: "GuildStatsID stores the current guild ID", Persist: true},
		{Name: "GuildStatsName", Type: "string", Doc: "GuildStatsName stores the current guild name", Persist: true},
		{Name: "GuildStatsUniqueGuilds", Type: "int", Doc: "GuildStatsUniqueGuilds stores the count of unique guilds in the database", Persist: false},
		{Name: "GuildStatsUniqueUsers", Type: "int", Doc: "GuildStatsUniqueUsers stores the count of unique users in the database", Persist: false},
		{Name: "GuildStatsInappropriateUsers", Type: "int", Doc: "GuildStatsInappropriateUsers stores the count of users with inappropriate messages", Persist: false},
		{Name: "GuildScanGuildNames", Type: "map[uint64]string", Doc: "GuildScanGuildNames stores guild names for flagged users", Persist: true},
		{Name: "GuildScanUserGuilds", Type: "map[uint64][]*types.UserGuildInfo", Doc: "GuildScanUserGuilds stores users and their guilds", Persist: true},
		{Name: "GuildScanMessageSummaries", Type: "map[uint64]*types.InappropriateUserSummary", Doc: "GuildScanMessageSummaries stores message summaries for users", Persist: true},
		{Name: "GuildScanFilteredUsers", Type: "map[uint64][]*types.UserGuildInfo", Doc: "GuildScanFilteredUsers stores users filtered by criteria", Persist: true},
		{Name: "GuildScanFilteredSummaries", Type: "map[uint64]*types.InappropriateUserSummary", Doc: "GuildScanFilteredSummaries stores filtered message summaries", Persist: true},
		{Name: "GuildScanMinGuilds", Type: "int", Doc: "GuildScanMinGuilds stores the minimum guilds filter value", Persist: true},
		{Name: "GuildScanMinJoinDuration", Type: "time.Duration", Doc: "GuildScanMinJoinDuration stores the minimum join duration filter value", Persist: true},

		// Guild ban logs related keys
		{Name: "GuildBanLogs", Type: "[]*types.GuildBanLog", Doc: "GuildBanLogs stores guild ban operation logs", Persist: true},
		{Name: "GuildBanLogCursor", Type: "*types.LogCursor", Doc: "GuildBanLogCursor stores the current guild ban log cursor", Persist: true},
		{Name: "GuildBanLogNextCursor", Type: "*types.LogCursor", Doc: "GuildBanLogNextCursor stores the next guild ban log cursor", Persist: true},
		{Name: "GuildBanLogPrevCursors", Type: "[]*types.LogCursor", Doc: "GuildBanLogPrevCursors stores previous guild ban log cursors", Persist: true},
	}

	bufferKeys := []KeyDef{
		{Name: "ImageBuffer", Doc: "ImageBuffer stores binary image data", Persist: false},
	}

	data := struct {
		Keys       []KeyDef
		BufferKeys []KeyDef
	}{
		Keys:       keys,
		BufferKeys: bufferKeys,
	}

	t := template.Must(template.New("keys").Parse(tmpl))
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		log.Fatal(err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile("keys_generated.go", formatted, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
