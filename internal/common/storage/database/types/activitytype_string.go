// Code generated by "stringer -type=ActivityType -linecomment"; DO NOT EDIT.

package types

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[ActivityTypeAll-0]
	_ = x[ActivityTypeUserViewed-1]
	_ = x[ActivityTypeUserLookup-2]
	_ = x[ActivityTypeUserConfirmed-3]
	_ = x[ActivityTypeUserConfirmedCustom-4]
	_ = x[ActivityTypeUserCleared-5]
	_ = x[ActivityTypeUserSkipped-6]
	_ = x[ActivityTypeUserRechecked-7]
	_ = x[ActivityTypeUserTrainingUpvote-8]
	_ = x[ActivityTypeUserTrainingDownvote-9]
	_ = x[ActivityTypeUserDeleted-10]
	_ = x[ActivityTypeGroupViewed-11]
	_ = x[ActivityTypeGroupLookup-12]
	_ = x[ActivityTypeGroupConfirmed-13]
	_ = x[ActivityTypeGroupConfirmedCustom-14]
	_ = x[ActivityTypeGroupCleared-15]
	_ = x[ActivityTypeGroupSkipped-16]
	_ = x[ActivityTypeGroupTrainingUpvote-17]
	_ = x[ActivityTypeGroupTrainingDownvote-18]
	_ = x[ActivityTypeGroupDeleted-19]
	_ = x[ActivityTypeAppealSubmitted-20]
	_ = x[ActivityTypeAppealSkipped-21]
	_ = x[ActivityTypeAppealAccepted-22]
	_ = x[ActivityTypeAppealRejected-23]
	_ = x[ActivityTypeAppealClosed-24]
}

const _ActivityType_name = "ALLUSER_VIEWEDUSER_LOOKUPUSER_CONFIRMEDUSER_CONFIRMED_CUSTOMUSER_CLEAREDUSER_SKIPPEDUSER_RECHECKEDUSER_TRAINING_UPVOTEUSER_TRAINING_DOWNVOTEUSER_DELETEDGROUP_VIEWEDGROUP_LOOKUPGROUP_CONFIRMEDGROUP_CONFIRMED_CUSTOMGROUP_CLEAREDGROUP_SKIPPEDGROUP_TRAINING_UPVOTEGROUP_TRAINING_DOWNVOTEGROUP_DELETEDAPPEAL_SUBMITTEDAPPEAL_SKIPPEDAPPEAL_ACCEPTEDAPPEAL_REJECTEDAPPEAL_CLOSED"

var _ActivityType_index = [...]uint16{0, 3, 14, 25, 39, 60, 72, 84, 98, 118, 140, 152, 164, 176, 191, 213, 226, 239, 260, 283, 296, 312, 326, 341, 356, 369}

func (i ActivityType) String() string {
	if i < 0 || i >= ActivityType(len(_ActivityType_index)-1) {
		return "ActivityType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _ActivityType_name[_ActivityType_index[i]:_ActivityType_index[i+1]]
}
