package main

import (
	"math/rand"
	"testing"
)

func riskRank(level string) int {
	switch level {
	case "high":
		return 2
	case "medium":
		return 1
	default:
		return 0
	}
}

// deterministicFiller builds an n-rune string of pseudo-random CJK ideographs.
// A fixed seed keeps tests reproducible; using math/rand (rather than an
// arithmetic step) avoids accidental long diagonal matches between two
// independently seeded fillers.
func deterministicFiller(seed int64, n int) string {
	r := rand.New(rand.NewSource(seed))
	b := make([]rune, n)
	for i := range b {
		b[i] = rune(0x4E00 + r.Intn(2000))
	}
	return string(b)
}

func TestAssessSimilarityRiskLevels(t *testing.T) {
	verbatim := "勇者历经千辛万苦终于抵达了魔王的城堡前方。决战在即，胜负难料。"
	cases := []struct {
		name      string
		source    string
		candidate string
		strict    bool
		want      string
	}{
		{"identical verbatim", verbatim, verbatim, false, "high"},
		{"empty candidate", verbatim, "", false, "low"},
		{"empty source", "", verbatim, false, "low"},
		{
			"unrelated chinese",
			"春天来了，花园里开满了五彩缤纷的鲜花，蜜蜂在花丛中忙碌地采蜜。",
			"程序员坐在电脑前，专注地敲击键盘，调试着复杂的分布式系统代码。",
			false,
			"low",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := AssessSimilarity(tc.source, tc.candidate, tc.strict)
			if got.RiskLevel != tc.want {
				t.Fatalf("RiskLevel = %q, want %q (ngram=%.3f sentence=%.3f lcs=%d)",
					got.RiskLevel, tc.want, got.CharNGramOverlapRatio, got.SentenceOverlapRatio, got.LongestCommonRunes)
			}
		})
	}
}

func TestAssessSimilarityFlagsVerbatimChineseSentence(t *testing.T) {
	shared := "勇者历经千辛万苦终于抵达了魔王的城堡前方"
	source := "开篇引子。" + shared + "。战斗就此打响。"
	candidate := "这是一段截然不同的全新开场叙述文字。" + shared + "。后续剧情也彻底改写成别的走向。"

	got := AssessSimilarity(source, candidate, false)
	if got.RiskLevel != "high" {
		t.Fatalf("RiskLevel = %q, want high", got.RiskLevel)
	}
	if got.MatchedSentenceCount == 0 {
		t.Fatalf("expected at least one matched sentence, got 0")
	}
	foundSentenceFragment := false
	for _, frag := range got.HighRiskFragments {
		if frag.Reason == "sentence_overlap" {
			foundSentenceFragment = true
		}
	}
	if !foundSentenceFragment {
		t.Fatalf("expected a sentence_overlap high-risk fragment, got %+v", got.HighRiskFragments)
	}
}

func TestAssessSimilarityChineseNeedsNoSpaces(t *testing.T) {
	// Chinese has no word spaces; the char n-gram path must still catch heavy
	// reuse without relying on whitespace tokenization.
	source := "他缓缓抬起头望向远方的群山若有所思良久没有再说出一句话来"
	got := AssessSimilarity(source, source, false)
	if got.CharNGramOverlapRatio < 0.99 {
		t.Fatalf("identical spaceless Chinese should have ~1.0 n-gram overlap, got %.3f", got.CharNGramOverlapRatio)
	}
	if got.RiskLevel != "high" {
		t.Fatalf("RiskLevel = %q, want high", got.RiskLevel)
	}
}

func TestDefaultSimilarityThresholdsStrictIsTighter(t *testing.T) {
	strict := DefaultSimilarityThresholds(true)
	loose := DefaultSimilarityThresholds(false)
	if !(strict.CharNGramOverlapWarn <= loose.CharNGramOverlapWarn &&
		strict.CharNGramOverlapFail <= loose.CharNGramOverlapFail &&
		strict.SentenceOverlapWarn <= loose.SentenceOverlapWarn &&
		strict.SentenceOverlapFail <= loose.SentenceOverlapFail &&
		strict.LongestCommonWarn <= loose.LongestCommonWarn &&
		strict.LongestCommonFail <= loose.LongestCommonFail) {
		t.Fatalf("strict thresholds must be <= loose thresholds\nstrict=%+v\nloose=%+v", strict, loose)
	}
}

func TestAssessSimilarityReturnsSelectedThresholds(t *testing.T) {
	got := AssessSimilarity("abc", "abc", true)
	if got.Thresholds != DefaultSimilarityThresholds(true) {
		t.Fatalf("result should carry the strict thresholds, got %+v", got.Thresholds)
	}
}

func TestAssessSimilarityStrictNeverWeaker(t *testing.T) {
	// For identical input, a stricter assessment can only be equal or higher risk.
	inputs := []struct{ source, candidate string }{
		{"勇者踏上旅途历经磨难最终击败魔王拯救世界苍生", "勇者踏上旅途历经磨难最终击败魔王拯救世界苍生"},
		{deterministicFiller(11, 600), deterministicFiller(11, 600)[:300] + deterministicFiller(22, 300)},
		{"完全无关的甲段文本内容描述", "彻底不同的乙段文本叙述说明"},
	}
	for i, in := range inputs {
		loose := AssessSimilarity(in.source, in.candidate, false)
		strict := AssessSimilarity(in.source, in.candidate, true)
		if riskRank(strict.RiskLevel) < riskRank(loose.RiskLevel) {
			t.Fatalf("case %d: strict (%s) weaker than loose (%s)", i, strict.RiskLevel, loose.RiskLevel)
		}
	}
}

func TestAssessSimilarityStrictEscalatesLongFragment(t *testing.T) {
	// A 70-rune reused block sits between the loose warn (60) and fail (90)
	// thresholds but at/above the strict fail (65) threshold, so the same input
	// is medium under loose rules and high under strict rules. The surrounding
	// fillers are long enough to keep the n-gram ratio well below both warns.
	shared := deterministicFiller(99, 70)
	source := deterministicFiller(1, 1000) + shared + deterministicFiller(3, 1000)
	candidate := deterministicFiller(2, 1000) + shared + deterministicFiller(4, 1000)

	loose := AssessSimilarity(source, candidate, false)
	strict := AssessSimilarity(source, candidate, true)

	if loose.LongestCommonRunes < 65 || loose.LongestCommonRunes >= 90 {
		t.Fatalf("setup expects longest common fragment in [65,90), got %d", loose.LongestCommonRunes)
	}
	if loose.RiskLevel != "medium" {
		t.Fatalf("loose RiskLevel = %q, want medium (lcs=%d ngram=%.3f)", loose.RiskLevel, loose.LongestCommonRunes, loose.CharNGramOverlapRatio)
	}
	if strict.RiskLevel != "high" {
		t.Fatalf("strict RiskLevel = %q, want high (lcs=%d)", strict.RiskLevel, strict.LongestCommonRunes)
	}
}

func TestNormalizeSimilarityText(t *testing.T) {
	got := normalizeSimilarityText("Hello, 世界！ 123")
	want := "hello世界123"
	if got != want {
		t.Fatalf("normalizeSimilarityText = %q, want %q", got, want)
	}
}

func TestSplitSentencesForSimilarity(t *testing.T) {
	got := splitSentencesForSimilarity("甲乙丙。丁戊己！  \n庚辛")
	if len(got) != 3 {
		t.Fatalf("expected 3 sentences, got %d (%+v)", len(got), got)
	}
	if got[0].norm != "甲乙丙" || got[1].norm != "丁戊己" || got[2].norm != "庚辛" {
		t.Fatalf("unexpected sentence norms: %+v", got)
	}
}

func TestLongestCommonRuneFragment(t *testing.T) {
	n, frag, capped := longestCommonRuneFragment([]rune("abcXYZWdef"), []rune("123XYZW45"))
	if n != 4 || frag != "XYZW" || capped {
		t.Fatalf("got (%d, %q, %v), want (4, \"XYZW\", false)", n, frag, capped)
	}
}

func TestLongestCommonRuneFragmentCaps(t *testing.T) {
	big := deterministicFiller(5, similarityLCSMaxRunes+2000)
	n, _, capped := longestCommonRuneFragment([]rune(big), []rune(big))
	if !capped {
		t.Fatalf("expected capped=true for input longer than %d", similarityLCSMaxRunes)
	}
	if n != similarityLCSMaxRunes {
		t.Fatalf("expected longest=%d after cap, got %d", similarityLCSMaxRunes, n)
	}
}

func TestCharNGramSetShortText(t *testing.T) {
	set := charNGramSet("abc", similarityNGramSize) // shorter than n
	if len(set) != 1 {
		t.Fatalf("short text should yield 1 gram, got %d", len(set))
	}
	if _, ok := set["abc"]; !ok {
		t.Fatalf("short-text gram should be the whole text")
	}
}

func TestAssessSimilarityCapsLongInput(t *testing.T) {
	big := deterministicFiller(7, similarityLCSMaxRunes+2000)
	got := AssessSimilarity(big, big, false)
	if !got.LongestCommonWasCapped {
		t.Fatalf("expected LongestCommonWasCapped=true for very long identical input")
	}
	if got.RiskLevel != "high" {
		t.Fatalf("identical long input should be high risk, got %q", got.RiskLevel)
	}
}
