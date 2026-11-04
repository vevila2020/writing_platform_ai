package skills

import (
	"context"
	"strings"
	"unicode/utf8"
)

const (
	// Keyword matching is intentionally narrow: genre cards declare a handful
	// of distinctive compound terms, and one card is preloaded at most.
	maxKeywordsPerSkill  = 32
	maxKeywordRunes      = 64
	minKeywordRunes      = 2
	minASCIIKeywordRunes = 3
)

// KeywordMatch is the single genre card selected from the current message.
type KeywordMatch struct {
	Skill   Skill
	Keyword string
}

// ResolveKeywordInvocation returns at most one active genre card whose declared
// keyword appears in the message. It never runs in explicit-only catalogs and
// never applies to Agents other than the writing and general Agents, so
// interactive and image turns keep their existing context.
func (b *Backend) ResolveKeywordInvocation(ctx context.Context, message string) (KeywordMatch, bool) {
	if b == nil || b.explicitOnly || !keywordAutoLoadAgent(b.agentKind) {
		return KeywordMatch{}, false
	}
	if ctx.Err() != nil || strings.TrimSpace(message) == "" {
		return KeywordMatch{}, false
	}
	return matchKeywordSkill(b.activeRecords(ctx), message)
}

// keywordAutoLoadAgent limits automatic genre preloading to the Agents whose
// turn preparation accepts writing guidance.
func keywordAutoLoadAgent(agentKind string) bool {
	switch strings.TrimSpace(agentKind) {
	case "", "ide", "general":
		return true
	}
	return false
}

type keywordCandidate struct {
	skill       Skill
	bestKeyword string
	bestLength  int
	matches     int
}

// matchKeywordSkill selects the most specific match: the longest matched
// keyword by rune length, then the most matched keywords, then Skill name.
// Deterministic ordering matters because the choice changes model context.
func matchKeywordSkill(records []record, message string) (KeywordMatch, bool) {
	lowered := strings.ToLower(message)
	var best *keywordCandidate
	for _, rec := range records {
		if !rec.skill.HasCapability(CapabilityGenreCard) || len(rec.skill.Keywords) == 0 {
			continue
		}
		candidate := keywordCandidate{skill: rec.skill}
		for _, keyword := range rec.skill.Keywords {
			normalized := normalizeKeyword(keyword)
			if normalized == "" || !keywordMatches(lowered, normalized) {
				continue
			}
			candidate.matches++
			if length := utf8.RuneCountInString(normalized); length > candidate.bestLength {
				candidate.bestLength = length
				candidate.bestKeyword = strings.TrimSpace(keyword)
			}
		}
		if candidate.matches == 0 {
			continue
		}
		if best == nil || betterKeywordCandidate(candidate, *best) {
			selected := candidate
			best = &selected
		}
	}
	if best == nil {
		return KeywordMatch{}, false
	}
	return KeywordMatch{Skill: best.skill, Keyword: best.bestKeyword}, true
}

func betterKeywordCandidate(candidate, current keywordCandidate) bool {
	if candidate.bestLength != current.bestLength {
		return candidate.bestLength > current.bestLength
	}
	if candidate.matches != current.matches {
		return candidate.matches > current.matches
	}
	return candidate.skill.Name < current.skill.Name
}

// normalizeKeyword returns the lowercase matching form, or "" when the keyword
// is too short or too long to be a reliable signal.
func normalizeKeyword(keyword string) string {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" || utf8.RuneCountInString(keyword) > maxKeywordRunes {
		return ""
	}
	if isASCIIString(keyword) {
		if len(keyword) < minASCIIKeywordRunes {
			return ""
		}
	} else if utf8.RuneCountInString(keyword) < minKeywordRunes {
		return ""
	}
	return strings.ToLower(keyword)
}

// keywordMatches uses substring matching for CJK terms and word boundaries for
// ASCII terms so "infinite" does not match inside "infinitely".
func keywordMatches(loweredMessage, normalizedKeyword string) bool {
	if normalizedKeyword == "" {
		return false
	}
	if isASCIIString(normalizedKeyword) {
		return containsASCIIWord(loweredMessage, normalizedKeyword)
	}
	return strings.Contains(loweredMessage, normalizedKeyword)
}

func containsASCIIWord(message, word string) bool {
	for offset := 0; offset <= len(message); {
		index := strings.Index(message[offset:], word)
		if index < 0 {
			return false
		}
		start := offset + index
		end := start + len(word)
		if (start == 0 || !isKeywordWordByte(message[start-1])) &&
			(end == len(message) || !isKeywordWordByte(message[end])) {
			return true
		}
		offset = start + 1
	}
	return false
}

func isKeywordWordByte(value byte) bool {
	return value >= 'a' && value <= 'z' ||
		value >= 'A' && value <= 'Z' ||
		value >= '0' && value <= '9'
}

func isASCIIString(value string) bool {
	for index := 0; index < len(value); index++ {
		if value[index] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}
