package app

import "github.com/dlclark/regexp2"

var uiSubtitlePattern *regexp2.Regexp

func ApplySubtitlePattern(pattern string) {
	if pattern == "" {
		uiSubtitlePattern = nil

		return
	}

	if o := uiSubtitlePattern; o != nil && o.String() == pattern {
		return
	}

	reg, err := regexp2.Compile(pattern, regexp2.IgnoreCase|regexp2.Compiled)
	if err == nil {
		uiSubtitlePattern = reg
	} else {
		uiSubtitlePattern = nil
	}
}

func SubtitlePattern() *regexp2.Regexp {
	return uiSubtitlePattern
}
