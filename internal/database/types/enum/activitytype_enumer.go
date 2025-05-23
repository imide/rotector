// Code generated by "enumer -type=ActivityType -trimprefix=ActivityType"; DO NOT EDIT.

package enum

import (
	"fmt"
	"strings"
)

const _ActivityTypeName = "AllUserViewedUserLookupUserConfirmedUserClearedUserSkippedUserQueuedUserTrainingUpvoteUserTrainingDownvoteUserDeletedGroupViewedGroupLookupGroupConfirmedGroupConfirmedCustomGroupClearedGroupSkippedGroupTrainingUpvoteGroupTrainingDownvoteGroupDeletedUserLookupDiscordAppealSubmittedAppealClaimedAppealAcceptedAppealRejectedAppealClosedAppealReopenedUserDataDeletedUserBlacklistedDiscordUserBannedDiscordUserUnbannedBotSettingUpdatedGuildBansGroupQueued"

var _ActivityTypeIndex = [...]uint16{0, 3, 13, 23, 36, 47, 58, 68, 86, 106, 117, 128, 139, 153, 173, 185, 197, 216, 237, 249, 266, 281, 294, 308, 322, 334, 348, 363, 378, 395, 414, 431, 440, 451}

const _ActivityTypeLowerName = "alluservieweduserlookupuserconfirmedusercleareduserskippeduserqueuedusertrainingupvoteusertrainingdownvoteuserdeletedgroupviewedgrouplookupgroupconfirmedgroupconfirmedcustomgroupclearedgroupskippedgrouptrainingupvotegrouptrainingdownvotegroupdeleteduserlookupdiscordappealsubmittedappealclaimedappealacceptedappealrejectedappealclosedappealreopeneduserdatadeleteduserblacklisteddiscorduserbanneddiscorduserunbannedbotsettingupdatedguildbansgroupqueued"

func (i ActivityType) String() string {
	if i < 0 || i >= ActivityType(len(_ActivityTypeIndex)-1) {
		return fmt.Sprintf("ActivityType(%d)", i)
	}
	return _ActivityTypeName[_ActivityTypeIndex[i]:_ActivityTypeIndex[i+1]]
}

// An "invalid array index" compiler error signifies that the constant values have changed.
// Re-run the stringer command to generate them again.
func _ActivityTypeNoOp() {
	var x [1]struct{}
	_ = x[ActivityTypeAll-(0)]
	_ = x[ActivityTypeUserViewed-(1)]
	_ = x[ActivityTypeUserLookup-(2)]
	_ = x[ActivityTypeUserConfirmed-(3)]
	_ = x[ActivityTypeUserCleared-(4)]
	_ = x[ActivityTypeUserSkipped-(5)]
	_ = x[ActivityTypeUserQueued-(6)]
	_ = x[ActivityTypeUserTrainingUpvote-(7)]
	_ = x[ActivityTypeUserTrainingDownvote-(8)]
	_ = x[ActivityTypeUserDeleted-(9)]
	_ = x[ActivityTypeGroupViewed-(10)]
	_ = x[ActivityTypeGroupLookup-(11)]
	_ = x[ActivityTypeGroupConfirmed-(12)]
	_ = x[ActivityTypeGroupConfirmedCustom-(13)]
	_ = x[ActivityTypeGroupCleared-(14)]
	_ = x[ActivityTypeGroupSkipped-(15)]
	_ = x[ActivityTypeGroupTrainingUpvote-(16)]
	_ = x[ActivityTypeGroupTrainingDownvote-(17)]
	_ = x[ActivityTypeGroupDeleted-(18)]
	_ = x[ActivityTypeUserLookupDiscord-(19)]
	_ = x[ActivityTypeAppealSubmitted-(20)]
	_ = x[ActivityTypeAppealClaimed-(21)]
	_ = x[ActivityTypeAppealAccepted-(22)]
	_ = x[ActivityTypeAppealRejected-(23)]
	_ = x[ActivityTypeAppealClosed-(24)]
	_ = x[ActivityTypeAppealReopened-(25)]
	_ = x[ActivityTypeUserDataDeleted-(26)]
	_ = x[ActivityTypeUserBlacklisted-(27)]
	_ = x[ActivityTypeDiscordUserBanned-(28)]
	_ = x[ActivityTypeDiscordUserUnbanned-(29)]
	_ = x[ActivityTypeBotSettingUpdated-(30)]
	_ = x[ActivityTypeGuildBans-(31)]
	_ = x[ActivityTypeGroupQueued-(32)]
}

var _ActivityTypeValues = []ActivityType{ActivityTypeAll, ActivityTypeUserViewed, ActivityTypeUserLookup, ActivityTypeUserConfirmed, ActivityTypeUserCleared, ActivityTypeUserSkipped, ActivityTypeUserQueued, ActivityTypeUserTrainingUpvote, ActivityTypeUserTrainingDownvote, ActivityTypeUserDeleted, ActivityTypeGroupViewed, ActivityTypeGroupLookup, ActivityTypeGroupConfirmed, ActivityTypeGroupConfirmedCustom, ActivityTypeGroupCleared, ActivityTypeGroupSkipped, ActivityTypeGroupTrainingUpvote, ActivityTypeGroupTrainingDownvote, ActivityTypeGroupDeleted, ActivityTypeUserLookupDiscord, ActivityTypeAppealSubmitted, ActivityTypeAppealClaimed, ActivityTypeAppealAccepted, ActivityTypeAppealRejected, ActivityTypeAppealClosed, ActivityTypeAppealReopened, ActivityTypeUserDataDeleted, ActivityTypeUserBlacklisted, ActivityTypeDiscordUserBanned, ActivityTypeDiscordUserUnbanned, ActivityTypeBotSettingUpdated, ActivityTypeGuildBans, ActivityTypeGroupQueued}

var _ActivityTypeNameToValueMap = map[string]ActivityType{
	_ActivityTypeName[0:3]:          ActivityTypeAll,
	_ActivityTypeLowerName[0:3]:     ActivityTypeAll,
	_ActivityTypeName[3:13]:         ActivityTypeUserViewed,
	_ActivityTypeLowerName[3:13]:    ActivityTypeUserViewed,
	_ActivityTypeName[13:23]:        ActivityTypeUserLookup,
	_ActivityTypeLowerName[13:23]:   ActivityTypeUserLookup,
	_ActivityTypeName[23:36]:        ActivityTypeUserConfirmed,
	_ActivityTypeLowerName[23:36]:   ActivityTypeUserConfirmed,
	_ActivityTypeName[36:47]:        ActivityTypeUserCleared,
	_ActivityTypeLowerName[36:47]:   ActivityTypeUserCleared,
	_ActivityTypeName[47:58]:        ActivityTypeUserSkipped,
	_ActivityTypeLowerName[47:58]:   ActivityTypeUserSkipped,
	_ActivityTypeName[58:68]:        ActivityTypeUserQueued,
	_ActivityTypeLowerName[58:68]:   ActivityTypeUserQueued,
	_ActivityTypeName[68:86]:        ActivityTypeUserTrainingUpvote,
	_ActivityTypeLowerName[68:86]:   ActivityTypeUserTrainingUpvote,
	_ActivityTypeName[86:106]:       ActivityTypeUserTrainingDownvote,
	_ActivityTypeLowerName[86:106]:  ActivityTypeUserTrainingDownvote,
	_ActivityTypeName[106:117]:      ActivityTypeUserDeleted,
	_ActivityTypeLowerName[106:117]: ActivityTypeUserDeleted,
	_ActivityTypeName[117:128]:      ActivityTypeGroupViewed,
	_ActivityTypeLowerName[117:128]: ActivityTypeGroupViewed,
	_ActivityTypeName[128:139]:      ActivityTypeGroupLookup,
	_ActivityTypeLowerName[128:139]: ActivityTypeGroupLookup,
	_ActivityTypeName[139:153]:      ActivityTypeGroupConfirmed,
	_ActivityTypeLowerName[139:153]: ActivityTypeGroupConfirmed,
	_ActivityTypeName[153:173]:      ActivityTypeGroupConfirmedCustom,
	_ActivityTypeLowerName[153:173]: ActivityTypeGroupConfirmedCustom,
	_ActivityTypeName[173:185]:      ActivityTypeGroupCleared,
	_ActivityTypeLowerName[173:185]: ActivityTypeGroupCleared,
	_ActivityTypeName[185:197]:      ActivityTypeGroupSkipped,
	_ActivityTypeLowerName[185:197]: ActivityTypeGroupSkipped,
	_ActivityTypeName[197:216]:      ActivityTypeGroupTrainingUpvote,
	_ActivityTypeLowerName[197:216]: ActivityTypeGroupTrainingUpvote,
	_ActivityTypeName[216:237]:      ActivityTypeGroupTrainingDownvote,
	_ActivityTypeLowerName[216:237]: ActivityTypeGroupTrainingDownvote,
	_ActivityTypeName[237:249]:      ActivityTypeGroupDeleted,
	_ActivityTypeLowerName[237:249]: ActivityTypeGroupDeleted,
	_ActivityTypeName[249:266]:      ActivityTypeUserLookupDiscord,
	_ActivityTypeLowerName[249:266]: ActivityTypeUserLookupDiscord,
	_ActivityTypeName[266:281]:      ActivityTypeAppealSubmitted,
	_ActivityTypeLowerName[266:281]: ActivityTypeAppealSubmitted,
	_ActivityTypeName[281:294]:      ActivityTypeAppealClaimed,
	_ActivityTypeLowerName[281:294]: ActivityTypeAppealClaimed,
	_ActivityTypeName[294:308]:      ActivityTypeAppealAccepted,
	_ActivityTypeLowerName[294:308]: ActivityTypeAppealAccepted,
	_ActivityTypeName[308:322]:      ActivityTypeAppealRejected,
	_ActivityTypeLowerName[308:322]: ActivityTypeAppealRejected,
	_ActivityTypeName[322:334]:      ActivityTypeAppealClosed,
	_ActivityTypeLowerName[322:334]: ActivityTypeAppealClosed,
	_ActivityTypeName[334:348]:      ActivityTypeAppealReopened,
	_ActivityTypeLowerName[334:348]: ActivityTypeAppealReopened,
	_ActivityTypeName[348:363]:      ActivityTypeUserDataDeleted,
	_ActivityTypeLowerName[348:363]: ActivityTypeUserDataDeleted,
	_ActivityTypeName[363:378]:      ActivityTypeUserBlacklisted,
	_ActivityTypeLowerName[363:378]: ActivityTypeUserBlacklisted,
	_ActivityTypeName[378:395]:      ActivityTypeDiscordUserBanned,
	_ActivityTypeLowerName[378:395]: ActivityTypeDiscordUserBanned,
	_ActivityTypeName[395:414]:      ActivityTypeDiscordUserUnbanned,
	_ActivityTypeLowerName[395:414]: ActivityTypeDiscordUserUnbanned,
	_ActivityTypeName[414:431]:      ActivityTypeBotSettingUpdated,
	_ActivityTypeLowerName[414:431]: ActivityTypeBotSettingUpdated,
	_ActivityTypeName[431:440]:      ActivityTypeGuildBans,
	_ActivityTypeLowerName[431:440]: ActivityTypeGuildBans,
	_ActivityTypeName[440:451]:      ActivityTypeGroupQueued,
	_ActivityTypeLowerName[440:451]: ActivityTypeGroupQueued,
}

var _ActivityTypeNames = []string{
	_ActivityTypeName[0:3],
	_ActivityTypeName[3:13],
	_ActivityTypeName[13:23],
	_ActivityTypeName[23:36],
	_ActivityTypeName[36:47],
	_ActivityTypeName[47:58],
	_ActivityTypeName[58:68],
	_ActivityTypeName[68:86],
	_ActivityTypeName[86:106],
	_ActivityTypeName[106:117],
	_ActivityTypeName[117:128],
	_ActivityTypeName[128:139],
	_ActivityTypeName[139:153],
	_ActivityTypeName[153:173],
	_ActivityTypeName[173:185],
	_ActivityTypeName[185:197],
	_ActivityTypeName[197:216],
	_ActivityTypeName[216:237],
	_ActivityTypeName[237:249],
	_ActivityTypeName[249:266],
	_ActivityTypeName[266:281],
	_ActivityTypeName[281:294],
	_ActivityTypeName[294:308],
	_ActivityTypeName[308:322],
	_ActivityTypeName[322:334],
	_ActivityTypeName[334:348],
	_ActivityTypeName[348:363],
	_ActivityTypeName[363:378],
	_ActivityTypeName[378:395],
	_ActivityTypeName[395:414],
	_ActivityTypeName[414:431],
	_ActivityTypeName[431:440],
	_ActivityTypeName[440:451],
}

// ActivityTypeString retrieves an enum value from the enum constants string name.
// Throws an error if the param is not part of the enum.
func ActivityTypeString(s string) (ActivityType, error) {
	if val, ok := _ActivityTypeNameToValueMap[s]; ok {
		return val, nil
	}

	if val, ok := _ActivityTypeNameToValueMap[strings.ToLower(s)]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("%s does not belong to ActivityType values", s)
}

// ActivityTypeValues returns all values of the enum
func ActivityTypeValues() []ActivityType {
	return _ActivityTypeValues
}

// ActivityTypeStrings returns a slice of all String values of the enum
func ActivityTypeStrings() []string {
	strs := make([]string, len(_ActivityTypeNames))
	copy(strs, _ActivityTypeNames)
	return strs
}

// IsAActivityType returns "true" if the value is listed in the enum definition. "false" otherwise
func (i ActivityType) IsAActivityType() bool {
	for _, v := range _ActivityTypeValues {
		if i == v {
			return true
		}
	}
	return false
}
