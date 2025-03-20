// Code generated by go generate; DO NOT EDIT.
package session

import (
	"time"

	apiTypes "github.com/jaxron/roapi.go/pkg/api/types"
	"github.com/robalyx/rotector/internal/common/client/ai"
	"github.com/robalyx/rotector/internal/common/queue"
	"github.com/robalyx/rotector/internal/common/storage/database/types"
	"github.com/robalyx/rotector/internal/common/storage/database/types/enum"
	"github.com/robalyx/rotector/internal/worker/core"
)

var (

	// UserID stores the user ID
	UserID = NewKey[uint64]("UserID", false)
	// IsGuildOwner indicates if the user is a guild owner
	IsGuildOwner = NewKey[bool]("IsGuildOwner", false)
	// MessageID stores the ID of the current message
	MessageID = NewKey[uint64]("MessageID", true)
	// CurrentPage stores the current page identifier
	CurrentPage = NewKey[string]("CurrentPage", true)
	// PreviousPages stores the navigation history
	PreviousPages = NewKey[[]string]("PreviousPages", true)
	// PaginationPage stores the current pagination page number
	PaginationPage = NewKey[int]("PaginationPage", true)
	// PaginationOffset stores the starting offset
	PaginationOffset = NewKey[int]("PaginationOffset", true)
	// PaginationTotalItems stores the total number of items
	PaginationTotalItems = NewKey[int]("PaginationTotalItems", true)
	// PaginationTotalPages stores the total number of pages
	PaginationTotalPages = NewKey[int]("PaginationTotalPages", true)
	// PaginationHasNextPage indicates if there is a next page
	PaginationHasNextPage = NewKey[bool]("PaginationHasNextPage", true)
	// PaginationHasPrevPage indicates if there is a previous page
	PaginationHasPrevPage = NewKey[bool]("PaginationHasPrevPage", true)
	// PaginationIsStreaming indicates if image streaming is active
	PaginationIsStreaming = NewKey[bool]("PaginationIsStreaming", true)
	// StatsIsRefreshed indicates if the data has been refreshed
	StatsIsRefreshed = NewKey[bool]("StatsIsRefreshed", true)
	// StatsUserCounts stores user statistics
	StatsUserCounts = NewKey[*types.UserCounts]("StatsUserCounts", true)
	// StatsGroupCounts stores group statistics
	StatsGroupCounts = NewKey[*types.GroupCounts]("StatsGroupCounts", true)
	// StatsActiveUsers stores the list of active reviewers
	StatsActiveUsers = NewKey[[]uint64]("StatsActiveUsers", true)
	// StatsVotes stores a user's voting statistics
	StatsVotes = NewKey[*types.VoteAccuracy]("StatsVotes", true)
	// StatusWorkers stores worker status information
	StatusWorkers = NewKey[[]core.Status]("StatusWorkers", false)
	// SettingName stores the name of the current setting
	SettingName = NewKey[string]("SettingName", true)
	// SettingType stores the type of the current setting
	SettingType = NewKey[string]("SettingType", true)
	// SettingValue stores the setting value
	SettingValue = NewKey[*Setting]("SettingValue", true)
	// SettingDisplay stores the display value of the setting
	SettingDisplay = NewKey[string]("SettingDisplay", true)
	// SettingCustomID stores the custom identifier
	SettingCustomID = NewKey[string]("SettingCustomID", true)
	// UserTarget stores the currently selected user
	UserTarget = NewKey[*types.ReviewUser]("UserTarget", true)
	// UserFlaggedFriends stores flagged friends
	UserFlaggedFriends = NewKey[map[uint64]*types.ReviewUser]("UserFlaggedFriends", true)
	// UserFlaggedGroups stores flagged groups
	UserFlaggedGroups = NewKey[map[uint64]*types.ReviewGroup]("UserFlaggedGroups", true)
	// UserFriends stores the user's friend list
	UserFriends = NewKey[[]*apiTypes.ExtendedFriend]("UserFriends", false)
	// UserPresences stores friend presence information
	UserPresences = NewKey[map[uint64]*apiTypes.UserPresenceResponse]("UserPresences", false)
	// UserGroups stores the list of groups
	UserGroups = NewKey[[]*apiTypes.UserGroupRoles]("UserGroups", false)
	// UserOutfits stores user outfits
	UserOutfits = NewKey[[]*apiTypes.Outfit]("UserOutfits", false)
	// GroupTarget stores the currently selected group
	GroupTarget = NewKey[*types.ReviewGroup]("GroupTarget", true)
	// GroupInfo stores additional group information
	GroupInfo = NewKey[*apiTypes.GroupResponse]("GroupInfo", true)
	// GroupFlaggedMembersCount stores the total number of flagged members in the current group
	GroupFlaggedMembersCount = NewKey[int]("GroupFlaggedMembersCount", true)
	// GroupPageFlaggedMembers stores flagged member details for the current page
	GroupPageFlaggedMembers = NewKey[map[uint64]*types.ReviewUser]("GroupPageFlaggedMembers", false)
	// GroupPageFlaggedMemberIDs stores flagged member IDs for the current page
	GroupPageFlaggedMemberIDs = NewKey[[]uint64]("GroupPageFlaggedMemberIDs", false)
	// UserReviewHistory stores IDs of previously reviewed users
	UserReviewHistory = NewKey[[]uint64]("UserReviewHistory", true)
	// UserReviewHistoryIndex stores the current position in the review history
	UserReviewHistoryIndex = NewKey[int]("UserReviewHistoryIndex", true)
	// GroupReviewHistory stores IDs of previously reviewed groups
	GroupReviewHistory = NewKey[[]uint64]("GroupReviewHistory", true)
	// GroupReviewHistoryIndex stores the current position in the review history
	GroupReviewHistoryIndex = NewKey[int]("GroupReviewHistoryIndex", true)
	// ReasonsChanged indicates if reasons have been modified
	ReasonsChanged = NewKey[bool]("ReasonsChanged", true)
	// OriginalUserReasons stores the initial user reasons
	OriginalUserReasons = NewKey[types.Reasons[enum.UserReasonType]]("OriginalUserReasons", true)
	// OriginalGroupReasons stores the initial group reasons
	OriginalGroupReasons = NewKey[types.Reasons[enum.GroupReasonType]]("OriginalGroupReasons", true)
	// SelectedReasonType stores the currently selected reason type for modal handling
	SelectedReasonType = NewKey[string]("SelectedReasonType", true)
	// ReviewLogs stores the current review logs
	ReviewLogs = NewKey[[]*types.ActivityLog]("ReviewLogs", true)
	// ReviewLogsHasMore indicates if there are more logs available
	ReviewLogsHasMore = NewKey[bool]("ReviewLogsHasMore", true)
	// DiscordUserLookupID stores the Discord user ID being looked up
	DiscordUserLookupID = NewKey[uint64]("DiscordUserLookupID", true)
	// DiscordUserLookupName stores the Discord username
	DiscordUserLookupName = NewKey[string]("DiscordUserLookupName", true)
	// DiscordUserGuilds stores a Discord user's guild memberships
	DiscordUserGuilds = NewKey[[]*types.UserGuildInfo]("DiscordUserGuilds", true)
	// DiscordUserGuildNames stores guild names for a Discord user
	DiscordUserGuildNames = NewKey[map[uint64]string]("DiscordUserGuildNames", true)
	// DiscordUserMessageSummary stores the user's inappropriate message summary
	DiscordUserMessageSummary = NewKey[*types.InappropriateUserSummary]("DiscordUserMessageSummary", true)
	// DiscordUserTotalGuilds stores the total number of flagged guilds for the user
	DiscordUserTotalGuilds = NewKey[int]("DiscordUserTotalGuilds", true)
	// GuildLookupCursor stores the current guild lookup cursor
	GuildLookupCursor = NewKey[*types.GuildCursor]("GuildLookupCursor", true)
	// GuildLookupNextCursor stores the next guild lookup cursor
	GuildLookupNextCursor = NewKey[*types.GuildCursor]("GuildLookupNextCursor", true)
	// GuildLookupPrevCursors stores previous guild lookup cursors
	GuildLookupPrevCursors = NewKey[[]*types.GuildCursor]("GuildLookupPrevCursors", true)
	// DiscordUserMessages stores the current page of messages
	DiscordUserMessages = NewKey[[]*types.InappropriateMessage]("DiscordUserMessages", true)
	// DiscordUserMessageCursor stores the current message cursor
	DiscordUserMessageCursor = NewKey[*types.MessageCursor]("DiscordUserMessageCursor", true)
	// DiscordUserMessageNextCursor stores the next message cursor
	DiscordUserMessageNextCursor = NewKey[*types.MessageCursor]("DiscordUserMessageNextCursor", true)
	// DiscordUserMessagePrevCursors stores previous message cursors
	DiscordUserMessagePrevCursors = NewKey[[]*types.MessageCursor]("DiscordUserMessagePrevCursors", true)
	// DiscordUserMessageGuildID stores the currently selected guild ID for messages
	DiscordUserMessageGuildID = NewKey[uint64]("DiscordUserMessageGuildID", true)
	// DiscordUserDataRedacted indicates if the user has requested data deletion
	DiscordUserDataRedacted = NewKey[bool]("DiscordUserDataRedacted", true)
	// DiscordUserMessageGuilds stores a map of guild IDs where the user has inappropriate messages
	DiscordUserMessageGuilds = NewKey[map[uint64]struct{}]("DiscordUserMessageGuilds", true)
	// ChatHistory stores the conversation history
	ChatHistory = NewKey[ai.ChatHistory]("ChatHistory", true)
	// ChatContext stores chat context information
	ChatContext = NewKey[string]("ChatContext", true)
	// LogActivities stores activity logs
	LogActivities = NewKey[[]*types.ActivityLog]("LogActivities", true)
	// LogCursor stores the current log cursor
	LogCursor = NewKey[*types.LogCursor]("LogCursor", true)
	// LogNextCursor stores the next log cursor
	LogNextCursor = NewKey[*types.LogCursor]("LogNextCursor", true)
	// LogPrevCursors stores previous log cursors
	LogPrevCursors = NewKey[[]*types.LogCursor]("LogPrevCursors", true)
	// LogFilterGuildID stores guild ID filter
	LogFilterGuildID = NewKey[uint64]("LogFilterGuildID", true)
	// LogFilterDiscordID stores Discord ID filter
	LogFilterDiscordID = NewKey[uint64]("LogFilterDiscordID", true)
	// LogFilterUserID stores user ID filter
	LogFilterUserID = NewKey[uint64]("LogFilterUserID", true)
	// LogFilterGroupID stores group ID filter
	LogFilterGroupID = NewKey[uint64]("LogFilterGroupID", true)
	// LogFilterReviewerID stores reviewer ID filter
	LogFilterReviewerID = NewKey[uint64]("LogFilterReviewerID", true)
	// LogFilterActivityType stores activity type filter
	LogFilterActivityType = NewKey[enum.ActivityType]("LogFilterActivityType", true)
	// LogFilterActivityCategory stores the currently selected activity category
	LogFilterActivityCategory = NewKey[string]("LogFilterActivityCategory", true)
	// LogFilterDateRangeStart stores start date filter
	LogFilterDateRangeStart = NewKey[time.Time]("LogFilterDateRangeStart", true)
	// LogFilterDateRangeEnd stores end date filter
	LogFilterDateRangeEnd = NewKey[time.Time]("LogFilterDateRangeEnd", true)
	// QueueUser stores the queued user
	QueueUser = NewKey[uint64]("QueueUser", true)
	// QueueStatus stores the queue status
	QueueStatus = NewKey[queue.Status]("QueueStatus", false)
	// QueuePriority stores the queue priority
	QueuePriority = NewKey[queue.Priority]("QueuePriority", false)
	// QueuePosition stores the queue position
	QueuePosition = NewKey[int]("QueuePosition", false)
	// QueueHighCount stores high priority queue count
	QueueHighCount = NewKey[int]("QueueHighCount", false)
	// QueueNormalCount stores normal priority queue count
	QueueNormalCount = NewKey[int]("QueueNormalCount", false)
	// QueueLowCount stores low priority queue count
	QueueLowCount = NewKey[int]("QueueLowCount", false)
	// AppealList stores the current page of appeals
	AppealList = NewKey[[]*types.FullAppeal]("AppealList", true)
	// AppealSelected stores the currently selected appeal
	AppealSelected = NewKey[*types.FullAppeal]("AppealSelected", true)
	// AppealMessages stores messages for the current appeal
	AppealMessages = NewKey[[]*types.AppealMessage]("AppealMessages", true)
	// AppealCursor stores the current cursor position
	AppealCursor = NewKey[*types.AppealTimeline]("AppealCursor", true)
	// AppealNextCursor stores the next cursor position
	AppealNextCursor = NewKey[*types.AppealTimeline]("AppealNextCursor", true)
	// AppealPrevCursors stores previous cursor positions
	AppealPrevCursors = NewKey[[]*types.AppealTimeline]("AppealPrevCursors", true)
	// AppealType stores the type of appeal being created
	AppealType = NewKey[enum.AppealType]("AppealType", true)
	// AppealRejectedCount stores the total number of rejected appeals for the user
	AppealRejectedCount = NewKey[int]("AppealRejectedCount", true)
	// VerifyUserID stores the user ID being verified
	VerifyUserID = NewKey[uint64]("VerifyUserID", true)
	// VerifyReason stores the verification reason
	VerifyReason = NewKey[string]("VerifyReason", true)
	// VerifyCode stores the verification code
	VerifyCode = NewKey[string]("VerifyCode", true)
	// CaptchaAnswer stores the CAPTCHA answer
	CaptchaAnswer = NewKey[string]("CaptchaAnswer", true)
	// AdminAction stores the current admin action
	AdminAction = NewKey[string]("AdminAction", true)
	// AdminActionID stores the admin action ID
	AdminActionID = NewKey[string]("AdminActionID", true)
	// AdminReason stores the admin action reason
	AdminReason = NewKey[string]("AdminReason", true)
	// AdminBanReason stores the ban reason
	AdminBanReason = NewKey[enum.BanReason]("AdminBanReason", true)
	// AdminBanExpiry stores the ban expiry time
	AdminBanExpiry = NewKey[*time.Time]("AdminBanExpiry", true)
	// AdminBanInfo stores ban information
	AdminBanInfo = NewKey[*types.DiscordBan]("AdminBanInfo", true)
	// LeaderboardStats stores leaderboard statistics
	LeaderboardStats = NewKey[[]*types.VoteAccuracy]("LeaderboardStats", true)
	// LeaderboardUsernames stores usernames for the leaderboard
	LeaderboardUsernames = NewKey[map[uint64]string]("LeaderboardUsernames", true)
	// LeaderboardCursor stores the current leaderboard cursor
	LeaderboardCursor = NewKey[*types.LeaderboardCursor]("LeaderboardCursor", true)
	// LeaderboardNextCursor stores the next leaderboard cursor
	LeaderboardNextCursor = NewKey[*types.LeaderboardCursor]("LeaderboardNextCursor", true)
	// LeaderboardPrevCursors stores previous leaderboard cursors
	LeaderboardPrevCursors = NewKey[[]*types.LeaderboardCursor]("LeaderboardPrevCursors", true)
	// LeaderboardLastRefresh stores the last refresh time
	LeaderboardLastRefresh = NewKey[time.Time]("LeaderboardLastRefresh", true)
	// LeaderboardNextRefresh stores the next refresh time
	LeaderboardNextRefresh = NewKey[time.Time]("LeaderboardNextRefresh", true)
	// ReviewerStats stores reviewer statistics
	ReviewerStats = NewKey[map[uint64]*types.ReviewerStats]("ReviewerStats", true)
	// ReviewerUsernames stores usernames for reviewers
	ReviewerUsernames = NewKey[map[uint64]string]("ReviewerUsernames", true)
	// ReviewerStatsCursor stores the current reviewer stats cursor
	ReviewerStatsCursor = NewKey[*types.ReviewerStatsCursor]("ReviewerStatsCursor", true)
	// ReviewerStatsNextCursor stores the next reviewer stats cursor
	ReviewerStatsNextCursor = NewKey[*types.ReviewerStatsCursor]("ReviewerStatsNextCursor", true)
	// ReviewerStatsPrevCursors stores previous reviewer stats cursors
	ReviewerStatsPrevCursors = NewKey[[]*types.ReviewerStatsCursor]("ReviewerStatsPrevCursors", true)
	// ReviewerStatsLastRefresh stores the last refresh time
	ReviewerStatsLastRefresh = NewKey[time.Time]("ReviewerStatsLastRefresh", true)
	// ReviewerStatsNextRefresh stores the next refresh time
	ReviewerStatsNextRefresh = NewKey[time.Time]("ReviewerStatsNextRefresh", true)
	// GuildScanType stores the type of guild scan selected (messages or servers)
	GuildScanType = NewKey[string]("GuildScanType", true)
	// GuildStatsID stores the current guild ID
	GuildStatsID = NewKey[uint64]("GuildStatsID", true)
	// GuildStatsName stores the current guild name
	GuildStatsName = NewKey[string]("GuildStatsName", true)
	// GuildStatsUniqueGuilds stores the count of unique guilds in the database
	GuildStatsUniqueGuilds = NewKey[int]("GuildStatsUniqueGuilds", false)
	// GuildStatsUniqueUsers stores the count of unique users in the database
	GuildStatsUniqueUsers = NewKey[int]("GuildStatsUniqueUsers", false)
	// GuildStatsInappropriateUsers stores the count of users with inappropriate messages
	GuildStatsInappropriateUsers = NewKey[int]("GuildStatsInappropriateUsers", false)
	// GuildScanGuildNames stores guild names for flagged users
	GuildScanGuildNames = NewKey[map[uint64]string]("GuildScanGuildNames", true)
	// GuildScanUserGuilds stores users and their guilds
	GuildScanUserGuilds = NewKey[map[uint64][]*types.UserGuildInfo]("GuildScanUserGuilds", true)
	// GuildScanMessageSummaries stores message summaries for users
	GuildScanMessageSummaries = NewKey[map[uint64]*types.InappropriateUserSummary]("GuildScanMessageSummaries", true)
	// GuildScanFilteredUsers stores users filtered by criteria
	GuildScanFilteredUsers = NewKey[map[uint64][]*types.UserGuildInfo]("GuildScanFilteredUsers", true)
	// GuildScanFilteredSummaries stores filtered message summaries
	GuildScanFilteredSummaries = NewKey[map[uint64]*types.InappropriateUserSummary]("GuildScanFilteredSummaries", true)
	// GuildScanMinGuilds stores the minimum guilds filter value
	GuildScanMinGuilds = NewKey[int]("GuildScanMinGuilds", true)
	// GuildScanMinJoinDuration stores the minimum join duration filter value
	GuildScanMinJoinDuration = NewKey[time.Duration]("GuildScanMinJoinDuration", true)
	// GuildBanLogs stores guild ban operation logs
	GuildBanLogs = NewKey[[]*types.GuildBanLog]("GuildBanLogs", true)
	// GuildBanLogCursor stores the current guild ban log cursor
	GuildBanLogCursor = NewKey[*types.LogCursor]("GuildBanLogCursor", true)
	// GuildBanLogNextCursor stores the next guild ban log cursor
	GuildBanLogNextCursor = NewKey[*types.LogCursor]("GuildBanLogNextCursor", true)
	// GuildBanLogPrevCursors stores previous guild ban log cursors
	GuildBanLogPrevCursors = NewKey[[]*types.LogCursor]("GuildBanLogPrevCursors", true)

	// ImageBuffer stores binary image data
	ImageBuffer = NewBufferKey("ImageBuffer", false)
)
