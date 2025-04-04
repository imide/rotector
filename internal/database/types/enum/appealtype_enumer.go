// Code generated by "enumer -type=AppealType -trimprefix=AppealType"; DO NOT EDIT.

package enum

import (
	"fmt"
	"strings"
)

const _AppealTypeName = "RobloxDiscord"

var _AppealTypeIndex = [...]uint8{0, 6, 13}

const _AppealTypeLowerName = "robloxdiscord"

func (i AppealType) String() string {
	if i < 0 || i >= AppealType(len(_AppealTypeIndex)-1) {
		return fmt.Sprintf("AppealType(%d)", i)
	}
	return _AppealTypeName[_AppealTypeIndex[i]:_AppealTypeIndex[i+1]]
}

// An "invalid array index" compiler error signifies that the constant values have changed.
// Re-run the stringer command to generate them again.
func _AppealTypeNoOp() {
	var x [1]struct{}
	_ = x[AppealTypeRoblox-(0)]
	_ = x[AppealTypeDiscord-(1)]
}

var _AppealTypeValues = []AppealType{AppealTypeRoblox, AppealTypeDiscord}

var _AppealTypeNameToValueMap = map[string]AppealType{
	_AppealTypeName[0:6]:       AppealTypeRoblox,
	_AppealTypeLowerName[0:6]:  AppealTypeRoblox,
	_AppealTypeName[6:13]:      AppealTypeDiscord,
	_AppealTypeLowerName[6:13]: AppealTypeDiscord,
}

var _AppealTypeNames = []string{
	_AppealTypeName[0:6],
	_AppealTypeName[6:13],
}

// AppealTypeString retrieves an enum value from the enum constants string name.
// Throws an error if the param is not part of the enum.
func AppealTypeString(s string) (AppealType, error) {
	if val, ok := _AppealTypeNameToValueMap[s]; ok {
		return val, nil
	}

	if val, ok := _AppealTypeNameToValueMap[strings.ToLower(s)]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("%s does not belong to AppealType values", s)
}

// AppealTypeValues returns all values of the enum
func AppealTypeValues() []AppealType {
	return _AppealTypeValues
}

// AppealTypeStrings returns a slice of all String values of the enum
func AppealTypeStrings() []string {
	strs := make([]string, len(_AppealTypeNames))
	copy(strs, _AppealTypeNames)
	return strs
}

// IsAAppealType returns "true" if the value is listed in the enum definition. "false" otherwise
func (i AppealType) IsAAppealType() bool {
	for _, v := range _AppealTypeValues {
		if i == v {
			return true
		}
	}
	return false
}
