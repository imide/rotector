// Code generated by "enumer -type=ReviewTargetMode -trimprefix=ReviewTargetMode"; DO NOT EDIT.

package enum

import (
	"fmt"
	"strings"
)

const _ReviewTargetModeName = "FlaggedConfirmedClearedBanned"

var _ReviewTargetModeIndex = [...]uint8{0, 7, 16, 23, 29}

const _ReviewTargetModeLowerName = "flaggedconfirmedclearedbanned"

func (i ReviewTargetMode) String() string {
	if i < 0 || i >= ReviewTargetMode(len(_ReviewTargetModeIndex)-1) {
		return fmt.Sprintf("ReviewTargetMode(%d)", i)
	}
	return _ReviewTargetModeName[_ReviewTargetModeIndex[i]:_ReviewTargetModeIndex[i+1]]
}

// An "invalid array index" compiler error signifies that the constant values have changed.
// Re-run the stringer command to generate them again.
func _ReviewTargetModeNoOp() {
	var x [1]struct{}
	_ = x[ReviewTargetModeFlagged-(0)]
	_ = x[ReviewTargetModeConfirmed-(1)]
	_ = x[ReviewTargetModeCleared-(2)]
	_ = x[ReviewTargetModeBanned-(3)]
}

var _ReviewTargetModeValues = []ReviewTargetMode{ReviewTargetModeFlagged, ReviewTargetModeConfirmed, ReviewTargetModeCleared, ReviewTargetModeBanned}

var _ReviewTargetModeNameToValueMap = map[string]ReviewTargetMode{
	_ReviewTargetModeName[0:7]:        ReviewTargetModeFlagged,
	_ReviewTargetModeLowerName[0:7]:   ReviewTargetModeFlagged,
	_ReviewTargetModeName[7:16]:       ReviewTargetModeConfirmed,
	_ReviewTargetModeLowerName[7:16]:  ReviewTargetModeConfirmed,
	_ReviewTargetModeName[16:23]:      ReviewTargetModeCleared,
	_ReviewTargetModeLowerName[16:23]: ReviewTargetModeCleared,
	_ReviewTargetModeName[23:29]:      ReviewTargetModeBanned,
	_ReviewTargetModeLowerName[23:29]: ReviewTargetModeBanned,
}

var _ReviewTargetModeNames = []string{
	_ReviewTargetModeName[0:7],
	_ReviewTargetModeName[7:16],
	_ReviewTargetModeName[16:23],
	_ReviewTargetModeName[23:29],
}

// ReviewTargetModeString retrieves an enum value from the enum constants string name.
// Throws an error if the param is not part of the enum.
func ReviewTargetModeString(s string) (ReviewTargetMode, error) {
	if val, ok := _ReviewTargetModeNameToValueMap[s]; ok {
		return val, nil
	}

	if val, ok := _ReviewTargetModeNameToValueMap[strings.ToLower(s)]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("%s does not belong to ReviewTargetMode values", s)
}

// ReviewTargetModeValues returns all values of the enum
func ReviewTargetModeValues() []ReviewTargetMode {
	return _ReviewTargetModeValues
}

// ReviewTargetModeStrings returns a slice of all String values of the enum
func ReviewTargetModeStrings() []string {
	strs := make([]string, len(_ReviewTargetModeNames))
	copy(strs, _ReviewTargetModeNames)
	return strs
}

// IsAReviewTargetMode returns "true" if the value is listed in the enum definition. "false" otherwise
func (i ReviewTargetMode) IsAReviewTargetMode() bool {
	for _, v := range _ReviewTargetModeValues {
		if i == v {
			return true
		}
	}
	return false
}
