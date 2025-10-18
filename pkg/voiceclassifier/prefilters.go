package voiceclassifier

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/tryhavana/take-home-test/pkg/common"
)

var (
	reVoicemail = regexp.MustCompile(`(?is)\b(please leave (a )?message|at the tone|after the beep|voicemail|leave your message)\b`)
	reWrongNum  = regexp.MustCompile(`(?is)\b(wrong\s+number|you\s+have\s+the\s+wrong\s+number|n[uú]mero\s+equivocado|not\s+(my|this)\s+number)\b`)
)

// ok builds a short-circuit response for obvious cases
func ok(intent common.Intent, reason DecisionReason) *ClassifyResponse {
	return &ClassifyResponse{
		Intent:         intent,
		Verdict:        "supported",
		Evidence:       []string{"User: [prefilter matched]"},
		NeedsReview:    false,
		DecisionReason: string(reason),
	}
}

// helper functions for user content
func userLines(t string) []string {
	var lines []string
	for _, line := range strings.Split(t, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "User:") {
			lines = append(lines, trim)
		}
	}
	return lines
}

func nonWSRuneCount(s string) int {
	n := 0
	for _, r := range s {
		if !unicode.IsSpace(r) {
			n++
		}
	}
	return n
}

// flags transcripts with short user speech based on predetermined conditions
func tooShortUserContent(t string) bool {
	uls := userLines(t)
	if len(uls) == 0 {
		return true 
	}

	wordCount := 0
	var buf strings.Builder
	for _, ln := range uls {
		txt := strings.TrimSpace(strings.TrimPrefix(ln, "User:"))
		wordCount += len(strings.Fields(txt))
		buf.WriteString(txt)
	}
	runes := nonWSRuneCount(buf.String())
	return len(uls) < 2 && wordCount < 10 && runes < 60
}

func runPrefilters(t string) (*ClassifyResponse, bool) {
	switch {
	case reVoicemail.MatchString(t):
		return ok(common.IntentVoiceVoiceMail, DecisionPrefilterVoicemail), true
	case reWrongNum.MatchString(t):
		return ok(common.IntentVoiceWrongNumber, DecisionPrefilterWrongNumber), true
	case tooShortUserContent(t):
		return ok(common.IntentVoiceImmediateHangup, DecisionPrefilterHangup), true
	default:
		return nil, false
	}
}
