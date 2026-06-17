package main

import (
	"strings"
	"unicode"
)

const (
	similarityNGramSize      = 8
	similaritySentenceMinLen = 14
	similarityLCSMaxRunes    = 10000
)

type SimilarityThresholds struct {
	CharNGramOverlapWarn float64 `json:"char_ngram_overlap_warn"`
	CharNGramOverlapFail float64 `json:"char_ngram_overlap_fail"`
	SentenceOverlapWarn  float64 `json:"sentence_overlap_warn"`
	SentenceOverlapFail  float64 `json:"sentence_overlap_fail"`
	LongestCommonWarn    int     `json:"longest_common_warn"`
	LongestCommonFail    int     `json:"longest_common_fail"`
}

type SimilarityFragment struct {
	Source string `json:"source"`
	Runes  int    `json:"runes"`
	Reason string `json:"reason"`
}

type SimilarityResult struct {
	SourceRunes            int                  `json:"source_runes"`
	CandidateRunes         int                  `json:"candidate_runes"`
	CharNGramSize          int                  `json:"char_ngram_size"`
	CharNGramOverlapRatio  float64              `json:"char_ngram_overlap_ratio"`
	SentenceOverlapRatio   float64              `json:"sentence_overlap_ratio"`
	LongestCommonRunes     int                  `json:"longest_common_runes"`
	RiskLevel              string               `json:"risk_level"`
	HighRiskFragments      []SimilarityFragment `json:"high_risk_fragments,omitempty"`
	Thresholds             SimilarityThresholds `json:"thresholds"`
	LongestCommonWasCapped bool                 `json:"longest_common_was_capped,omitempty"`
	MatchedSentenceCount   int                  `json:"matched_sentence_count"`
	ComparedSentenceCount  int                  `json:"compared_sentence_count"`
	CharNGramOverlapCount  int                  `json:"char_ngram_overlap_count"`
	ComparedCharNGramCount int                  `json:"compared_char_ngram_count"`
}

func DefaultSimilarityThresholds(strict bool) SimilarityThresholds {
	if strict {
		return SimilarityThresholds{
			CharNGramOverlapWarn: 0.08,
			CharNGramOverlapFail: 0.14,
			SentenceOverlapWarn:  0.02,
			SentenceOverlapFail:  0.05,
			LongestCommonWarn:    40,
			LongestCommonFail:    65,
		}
	}
	return SimilarityThresholds{
		CharNGramOverlapWarn: 0.12,
		CharNGramOverlapFail: 0.22,
		SentenceOverlapWarn:  0.03,
		SentenceOverlapFail:  0.08,
		LongestCommonWarn:    60,
		LongestCommonFail:    90,
	}
}

func AssessSimilarity(source, candidate string, strict bool) SimilarityResult {
	thresholds := DefaultSimilarityThresholds(strict)
	sourceNorm := normalizeSimilarityText(source)
	candidateNorm := normalizeSimilarityText(candidate)

	result := SimilarityResult{
		SourceRunes:    len([]rune(source)),
		CandidateRunes: len([]rune(candidate)),
		CharNGramSize:  similarityNGramSize,
		RiskLevel:      "low",
		Thresholds:     thresholds,
	}

	sourceSet := charNGramSet(sourceNorm, similarityNGramSize)
	candidateSet := charNGramSet(candidateNorm, similarityNGramSize)
	result.ComparedCharNGramCount = minInt(len(sourceSet), len(candidateSet))
	if result.ComparedCharNGramCount > 0 {
		overlap := 0
		for gram := range candidateSet {
			if _, ok := sourceSet[gram]; ok {
				overlap++
			}
		}
		result.CharNGramOverlapCount = overlap
		result.CharNGramOverlapRatio = float64(overlap) / float64(result.ComparedCharNGramCount)
	}

	matchedSentences, comparedSentences := matchedSourceSentences(source, candidate)
	result.MatchedSentenceCount = len(matchedSentences)
	result.ComparedSentenceCount = comparedSentences
	if comparedSentences > 0 {
		result.SentenceOverlapRatio = float64(len(matchedSentences)) / float64(comparedSentences)
	}
	for _, sentence := range matchedSentences {
		if len(result.HighRiskFragments) >= 5 {
			break
		}
		result.HighRiskFragments = append(result.HighRiskFragments, SimilarityFragment{
			Source: truncate(sentence, 160),
			Runes:  len([]rune(sentence)),
			Reason: "sentence_overlap",
		})
	}

	longest, fragment, capped := longestCommonRuneFragment([]rune(sourceNorm), []rune(candidateNorm))
	result.LongestCommonRunes = longest
	result.LongestCommonWasCapped = capped
	if longest >= thresholds.LongestCommonWarn && fragment != "" {
		result.HighRiskFragments = append(result.HighRiskFragments, SimilarityFragment{
			Source: truncate(fragment, 180),
			Runes:  longest,
			Reason: "long_common_fragment",
		})
	}

	if result.CharNGramOverlapRatio >= thresholds.CharNGramOverlapFail ||
		result.SentenceOverlapRatio >= thresholds.SentenceOverlapFail ||
		result.LongestCommonRunes >= thresholds.LongestCommonFail {
		result.RiskLevel = "high"
	} else if result.CharNGramOverlapRatio >= thresholds.CharNGramOverlapWarn ||
		result.SentenceOverlapRatio >= thresholds.SentenceOverlapWarn ||
		result.LongestCommonRunes >= thresholds.LongestCommonWarn {
		result.RiskLevel = "medium"
	}
	return result
}

func normalizeSimilarityText(text string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(text) {
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func charNGramSet(text string, n int) map[string]struct{} {
	runes := []rune(text)
	set := make(map[string]struct{})
	if len(runes) < n || n <= 0 {
		if len(runes) > 0 {
			set[string(runes)] = struct{}{}
		}
		return set
	}
	for i := 0; i+n <= len(runes); i++ {
		set[string(runes[i:i+n])] = struct{}{}
	}
	return set
}

type sentenceForSimilarity struct {
	raw  string
	norm string
}

func matchedSourceSentences(source, candidate string) ([]string, int) {
	sourceSentences := splitSentencesForSimilarity(source)
	sourceByNorm := make(map[string]string)
	for _, sentence := range sourceSentences {
		if len([]rune(sentence.norm)) >= similaritySentenceMinLen {
			sourceByNorm[sentence.norm] = sentence.raw
		}
	}

	var matches []string
	seen := make(map[string]bool)
	compared := 0
	for _, sentence := range splitSentencesForSimilarity(candidate) {
		if len([]rune(sentence.norm)) < similaritySentenceMinLen {
			continue
		}
		compared++
		if raw := sourceByNorm[sentence.norm]; raw != "" && !seen[sentence.norm] {
			matches = append(matches, raw)
			seen[sentence.norm] = true
		}
	}
	return matches, compared
}

func splitSentencesForSimilarity(text string) []sentenceForSimilarity {
	var out []sentenceForSimilarity
	var b strings.Builder
	flush := func() {
		raw := strings.TrimSpace(b.String())
		b.Reset()
		if raw == "" {
			return
		}
		norm := normalizeSimilarityText(raw)
		if norm == "" {
			return
		}
		out = append(out, sentenceForSimilarity{raw: raw, norm: norm})
	}
	for _, r := range text {
		b.WriteRune(r)
		if isSentenceDelimiter(r) {
			flush()
		}
	}
	flush()
	return out
}

func isSentenceDelimiter(r rune) bool {
	switch r {
	case '。', '！', '？', '!', '?', '；', ';', '\n', '\r':
		return true
	default:
		return false
	}
}

func longestCommonRuneFragment(a, b []rune) (int, string, bool) {
	capped := false
	if len(a) > similarityLCSMaxRunes {
		a = a[:similarityLCSMaxRunes]
		capped = true
	}
	if len(b) > similarityLCSMaxRunes {
		b = b[:similarityLCSMaxRunes]
		capped = true
	}
	if len(a) == 0 || len(b) == 0 {
		return 0, "", capped
	}
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	best := 0
	bestEnd := 0
	for i := 0; i < len(a); i++ {
		for j := 0; j < len(b); j++ {
			if a[i] == b[j] {
				curr[j+1] = prev[j] + 1
				if curr[j+1] > best {
					best = curr[j+1]
					bestEnd = i + 1
				}
			} else {
				curr[j+1] = 0
			}
		}
		prev, curr = curr, prev
		for j := range curr {
			curr[j] = 0
		}
	}
	if best == 0 {
		return 0, "", capped
	}
	return best, string(a[bestEnd-best : bestEnd]), capped
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
