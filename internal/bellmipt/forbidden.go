package bellmipt

import "strings"

// ForbiddenLanguageAudit holds the result of scanning report text for
// forbidden promotional language.
type ForbiddenLanguageAudit struct {
	Passed bool     `json:"passed"`
	Hits   []string `json:"hits"`
}

// forbiddenPhrases are exact phrases that must not appear in the report
// (case-insensitive) unless they are part of an allowed negation sentence.
var forbiddenPhrases = []string{
	"proves the bell-mipt bridge",
	"proven bridge",
	"establishes mipt",
	"confirms holography",
	"solves",
	"breakthrough",
	"validated theory",
	"bell jumps are measurements",
	"bell jumps equal measurements",
	"explains black holes",
	"explains holography",
	"bell-mipt bridge established",
	"mipt observed",
	"holography explained",
	"bohmian mechanics validated",
	"proves mipt",
	"proves holography",
	"measurement-induced transition found",
}

// allowedSentences are negated/limitation sentences that may contain forbidden
// substrings but are permitted. They are stripped from the text before scanning.
var allowedSentences = []string{
	"this is not a physics promotion.",
	"this does not show bell jumps are measurements.",
	"no mipt claim.",
	"no holography claim.",
	"no physics promotion.",
	"no bell-jumps-equal-measurements claim.",
	// Also allow these as they appear in the markdown report
	"this does not implement mipt.",
	"this does not support any holography or black-hole claim.",
	"not a monitored quantum trajectory simulation",
}

// AuditForbiddenLanguage scans the given text for forbidden promotional phrases.
//
// Implementation: first strip all allowed negation sentences from the text
// (case-insensitive), then scan the remaining text for forbidden phrases.
func AuditForbiddenLanguage(text string) ForbiddenLanguageAudit {
	lower := strings.ToLower(text)

	// Strip allowed sentences
	for _, allowed := range allowedSentences {
		lower = strings.ReplaceAll(lower, allowed, "")
	}

	var hits []string
	for _, phrase := range forbiddenPhrases {
		if strings.Contains(lower, phrase) {
			hits = append(hits, phrase)
		}
	}

	return ForbiddenLanguageAudit{
		Passed: len(hits) == 0,
		Hits:   hits,
	}
}
