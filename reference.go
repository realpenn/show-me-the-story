package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	referenceDirName           = "reference"
	referenceChapterSplitRunes = 60000

	ReferenceSettingsStatusNone            = "none"
	ReferenceSettingsStatusAutoApplied     = "auto_applied"
	ReferenceSettingsStatusPreviewRequired = "preview_required"
	ReferenceSettingsStatusApplied         = "applied"
)

type ReferenceBook struct {
	Title       string             `json:"title,omitempty"`
	ImportedAt  string             `json:"imported_at,omitempty"`
	UpdatedAt   string             `json:"updated_at,omitempty"`
	TotalRunes  int                `json:"total_runes"`
	Chapters    []ReferenceChapter `json:"chapters"`
	SourceName  string             `json:"source_name,omitempty"`
	SourceNotes string             `json:"source_notes,omitempty"`
}

type ReferenceChapter struct {
	Num         int    `json:"num"`
	Title       string `json:"title"`
	ContentPath string `json:"content_path"`
	RuneCount   int    `json:"rune_count"`
	WordCount   int    `json:"word_count"`
}

type ReferenceAnalysis struct {
	Title                string                     `json:"title,omitempty"`
	StoryType            string                     `json:"story_type,omitempty"`
	Synopsis             string                     `json:"synopsis,omitempty"`
	WritingStyle         string                     `json:"writing_style,omitempty"`
	CoreSetting          string                     `json:"core_setting,omitempty"`
	GlobalNotes          string                     `json:"global_notes,omitempty"`
	Chapters             []ReferenceChapterAnalysis `json:"chapters"`
	Settings             ReferenceSettingsCandidate `json:"settings"`
	SettingsImportStatus string                     `json:"settings_import_status,omitempty"`
	AnalyzedAt           string                     `json:"analyzed_at,omitempty"`
	Mode                 string                     `json:"mode,omitempty"`
	VolumeCount          int                        `json:"volume_count,omitempty"`
}

type ReferenceChapterAnalysis struct {
	Num               int      `json:"num"`
	Title             string   `json:"title"`
	Summary           string   `json:"summary"`
	KeyEvents         []string `json:"key_events,omitempty"`
	SceneFunction     string   `json:"scene_function,omitempty"`
	ForeshadowPayoffs []string `json:"foreshadow_payoffs,omitempty"`
	EmotionalCurve    string   `json:"emotional_curve,omitempty"`
	EndingRoute       string   `json:"ending_route,omitempty"`
	Characters        []string `json:"characters,omitempty"`
}

type ReferenceSettingsCandidate struct {
	Characters    []ReferenceCharacterSeed    `json:"characters,omitempty"`
	Worldview     []ReferenceWorldviewSeed    `json:"worldview,omitempty"`
	Organizations []ReferenceOrganizationSeed `json:"organizations,omitempty"`
	Relations     []ReferenceRelationSeed     `json:"relations,omitempty"`
}

type ReferenceCharacterSeed struct {
	Name        string `json:"name"`
	Age         string `json:"age,omitempty"`
	Appearance  string `json:"appearance,omitempty"`
	Personality string `json:"personality,omitempty"`
	Background  string `json:"background,omitempty"`
	Motivation  string `json:"motivation,omitempty"`
	Abilities   string `json:"abilities,omitempty"`
	Notes       string `json:"notes,omitempty"`
}

type ReferenceWorldviewSeed struct {
	Category    string `json:"category,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Tags        string `json:"tags,omitempty"`
}

type ReferenceOrganizationSeed struct {
	Name        string   `json:"name"`
	Type        string   `json:"type,omitempty"`
	Description string   `json:"description,omitempty"`
	MemberNames []string `json:"member_names,omitempty"`
}

type ReferenceRelationSeed struct {
	SourceName string `json:"source_name"`
	SourceType string `json:"source_type,omitempty"`
	TargetName string `json:"target_name"`
	TargetType string `json:"target_type,omitempty"`
	Label      string `json:"label"`
}

func LoadReferenceBook(path string) (*ReferenceBook, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ReferenceBook{}, nil
		}
		return nil, fmt.Errorf("读取参考书文件失败: %w", err)
	}
	var book ReferenceBook
	if err := json.Unmarshal(data, &book); err != nil {
		return nil, fmt.Errorf("解析参考书文件失败: %w", err)
	}
	if book.Chapters == nil {
		book.Chapters = []ReferenceChapter{}
	}
	return &book, nil
}

func SaveReferenceBook(path string, book *ReferenceBook) error {
	data, err := json.MarshalIndent(book, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化参考书失败: %w", err)
	}
	return writeFileAtomic(path, data)
}

func LoadReferenceAnalysis(path string) (*ReferenceAnalysis, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ReferenceAnalysis{SettingsImportStatus: ReferenceSettingsStatusNone}, nil
		}
		return nil, fmt.Errorf("读取参考分析文件失败: %w", err)
	}
	var analysis ReferenceAnalysis
	if err := json.Unmarshal(data, &analysis); err != nil {
		return nil, fmt.Errorf("解析参考分析文件失败: %w", err)
	}
	if analysis.Chapters == nil {
		analysis.Chapters = []ReferenceChapterAnalysis{}
	}
	if analysis.SettingsImportStatus == "" {
		analysis.SettingsImportStatus = ReferenceSettingsStatusNone
	}
	return &analysis, nil
}

func SaveReferenceAnalysis(path string, analysis *ReferenceAnalysis) error {
	data, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化参考分析失败: %w", err)
	}
	return writeFileAtomic(path, data)
}

func BuildReferenceBookFromContent(projectDir string, content string, sourceName string) (*ReferenceBook, error) {
	segments := splitContentByChapters(content, []ContinueChapter{{Num: 1}})
	if len(segments) == 0 {
		segments = []string{content}
	}

	refDir := filepath.Join(projectDir, referenceDirName)
	if err := os.RemoveAll(refDir); err != nil {
		return nil, fmt.Errorf("清理旧参考章节失败: %w", err)
	}
	if err := os.MkdirAll(refDir, 0755); err != nil {
		return nil, fmt.Errorf("创建参考章节目录失败: %w", err)
	}

	now := time.Now().Format(time.RFC3339)
	book := &ReferenceBook{
		ImportedAt: now,
		UpdatedAt:  now,
		SourceName: strings.TrimSpace(sourceName),
		Chapters:   make([]ReferenceChapter, 0, len(segments)),
	}

	for i, seg := range segments {
		num := i + 1
		title := extractReferenceChapterTitle(seg, num)
		relPath := filepath.ToSlash(filepath.Join(referenceDirName, fmt.Sprintf("Chapter_%03d.txt", num)))
		absPath := filepath.Join(projectDir, relPath)
		if err := writeFileAtomic(absPath, []byte(strings.TrimSpace(seg)+"\n")); err != nil {
			return nil, fmt.Errorf("保存参考章节失败: %w", err)
		}
		runeCount := len([]rune(seg))
		book.TotalRunes += runeCount
		book.Chapters = append(book.Chapters, ReferenceChapter{
			Num:         num,
			Title:       title,
			ContentPath: relPath,
			RuneCount:   runeCount,
			WordCount:   estimateWordCount(seg),
		})
	}
	if len(book.Chapters) > 0 {
		book.Title = guessReferenceBookTitle(sourceName, book.Chapters[0].Title)
	}
	return book, nil
}

func ReplaceReferenceChapters(projectDir string, book *ReferenceBook, updates []ReferenceChapterUpdate) (*ReferenceBook, error) {
	if len(updates) == 0 {
		return nil, fmt.Errorf("章节列表不能为空")
	}

	refDir := filepath.Join(projectDir, referenceDirName)
	if err := os.RemoveAll(refDir); err != nil {
		return nil, fmt.Errorf("清理旧参考章节失败: %w", err)
	}
	if err := os.MkdirAll(refDir, 0755); err != nil {
		return nil, fmt.Errorf("创建参考章节目录失败: %w", err)
	}

	next := &ReferenceBook{
		Title:       "",
		ImportedAt:  time.Now().Format(time.RFC3339),
		UpdatedAt:   time.Now().Format(time.RFC3339),
		SourceName:  "",
		SourceNotes: "",
		Chapters:    make([]ReferenceChapter, 0, len(updates)),
	}
	if book != nil {
		next.Title = book.Title
		next.ImportedAt = book.ImportedAt
		next.SourceName = book.SourceName
		next.SourceNotes = book.SourceNotes
	}

	for i, ch := range updates {
		num := i + 1
		title := strings.TrimSpace(ch.Title)
		if title == "" {
			title = defaultReferenceChapterTitle(num)
		}
		body := strings.TrimSpace(ch.Content)
		if body == "" && book != nil {
			if old := findReferenceChapter(book, ch.Num); old != nil {
				if oldContent, err := ReadReferenceChapterContent(projectDir, *old); err == nil {
					body = strings.TrimSpace(oldContent)
				}
			}
		}
		if body == "" {
			return nil, fmt.Errorf("第 %d 章内容不能为空", num)
		}
		relPath := filepath.ToSlash(filepath.Join(referenceDirName, fmt.Sprintf("Chapter_%03d.txt", num)))
		if err := writeFileAtomic(filepath.Join(projectDir, relPath), []byte(body+"\n")); err != nil {
			return nil, fmt.Errorf("保存参考章节失败: %w", err)
		}
		runeCount := len([]rune(body))
		next.TotalRunes += runeCount
		next.Chapters = append(next.Chapters, ReferenceChapter{
			Num:         num,
			Title:       title,
			ContentPath: relPath,
			RuneCount:   runeCount,
			WordCount:   estimateWordCount(body),
		})
	}
	if next.Title == "" && len(next.Chapters) > 0 {
		next.Title = next.Chapters[0].Title
	}
	return next, nil
}

type ReferenceChapterUpdate struct {
	Num     int    `json:"num"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

func ReadReferenceChapterContent(projectDir string, ch ReferenceChapter) (string, error) {
	if ch.ContentPath == "" {
		return "", fmt.Errorf("章节缺少正文路径")
	}
	path := filepath.Join(projectDir, filepath.FromSlash(ch.ContentPath))
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func AnalyzeReferenceBook(ctx context.Context, apiCfg *APIConfig, cfg *Config, settings *ProjectSettings, projectDir string, book *ReferenceBook, logger *LogBroadcaster) (*ReferenceAnalysis, bool, error) {
	if err := validateAPIConfig(apiCfg); err != nil {
		return nil, false, err
	}
	if book == nil || len(book.Chapters) == 0 {
		return nil, false, fmt.Errorf("请先导入参考小说")
	}

	chapterAnalyses := make([]ReferenceChapterAnalysis, 0, len(book.Chapters))
	totalVolumes := 0
	for i, ch := range book.Chapters {
		if ctx.Err() != nil {
			return nil, false, fmt.Errorf("任务已取消")
		}
		logger.StepInfo(i+1, len(book.Chapters), fmt.Sprintf("正在分析参考第 %d 章...", ch.Num))
		content, err := ReadReferenceChapterContent(projectDir, ch)
		if err != nil {
			return nil, false, fmt.Errorf("读取参考第 %d 章失败: %w", ch.Num, err)
		}
		parts := splitTextByRunes(content, referenceAnalysisChunkRunes(apiCfg))
		totalVolumes += len(parts)
		partAnalyses := make([]ReferenceChapterAnalysis, 0, len(parts))
		for partIdx, part := range parts {
			partNote := ""
			if len(parts) > 1 {
				partNote = fmt.Sprintf("这是本章第 %d/%d 个分块。请只分析本分块内实际出现的信息，稍后系统会合并。", partIdx+1, len(parts))
			}
			analysis, err := analyzeReferenceChapterPart(ctx, apiCfg, cfg, ch, part, partNote, logger)
			if err != nil {
				return nil, false, err
			}
			partAnalyses = append(partAnalyses, analysis)
		}
		chapterAnalyses = append(chapterAnalyses, mergeReferenceChapterAnalyses(ch, partAnalyses))
	}

	bookAnalysis, err := analyzeReferenceBookSummary(ctx, apiCfg, cfg, book, chapterAnalyses, logger)
	if err != nil {
		return nil, false, err
	}
	bookAnalysis.Chapters = chapterAnalyses
	bookAnalysis.AnalyzedAt = time.Now().Format(time.RFC3339)
	bookAnalysis.VolumeCount = totalVolumes
	if totalVolumes > len(book.Chapters) {
		bookAnalysis.Mode = "chunked"
	} else {
		bookAnalysis.Mode = "per_chapter"
	}

	appliedSettings := false
	if hasReferenceSettingsCandidate(bookAnalysis.Settings) {
		if isProjectSettingsEmpty(settings) {
			applyReferenceSettingsCandidate(settings, bookAnalysis.Settings)
			bookAnalysis.SettingsImportStatus = ReferenceSettingsStatusAutoApplied
			appliedSettings = true
		} else {
			bookAnalysis.SettingsImportStatus = ReferenceSettingsStatusPreviewRequired
		}
	} else {
		bookAnalysis.SettingsImportStatus = ReferenceSettingsStatusNone
	}

	return bookAnalysis, appliedSettings, nil
}

func analyzeReferenceChapterPart(ctx context.Context, apiCfg *APIConfig, cfg *Config, ch ReferenceChapter, content string, partNote string, logger *LogBroadcaster) (ReferenceChapterAnalysis, error) {
	userPrompt := RenderPrompt(cfg.Prompts.ReferenceChapterAnalysis, map[string]string{
		"ChapterNum":     fmt.Sprintf("%d", ch.Num),
		"ChapterTitle":   ch.Title,
		"PartNote":       partNote,
		"ChapterContent": content,
	})
	systemPrompt := SystemPromptFor(cfg.Language, "reference_analysis_json")
	rawResp := CallAPIWithRetryLog(ctx, apiCfg, systemPrompt, userPrompt, logger)
	if rawResp == "" {
		return ReferenceChapterAnalysis{}, fmt.Errorf("参考章节分析调用失败或被取消")
	}
	rawResp = cleanJSONResponse(rawResp)
	var analysis ReferenceChapterAnalysis
	if err := json.Unmarshal([]byte(rawResp), &analysis); err != nil {
		return ReferenceChapterAnalysis{}, fmt.Errorf("解析参考章节分析 JSON 失败: %w", err)
	}
	if analysis.Num <= 0 {
		analysis.Num = ch.Num
	}
	if strings.TrimSpace(analysis.Title) == "" {
		analysis.Title = ch.Title
	}
	return analysis, nil
}

func analyzeReferenceBookSummary(ctx context.Context, apiCfg *APIConfig, cfg *Config, book *ReferenceBook, chapters []ReferenceChapterAnalysis, logger *LogBroadcaster) (*ReferenceAnalysis, error) {
	chapterJSON, _ := json.MarshalIndent(chapters, "", "  ")
	userPrompt := RenderPrompt(cfg.Prompts.ReferenceBookAnalysis, map[string]string{
		"ReferenceTitle":    book.Title,
		"ChapterCount":      fmt.Sprintf("%d", len(book.Chapters)),
		"ChapterAnalyses":   string(chapterJSON),
		"ReferenceMetadata": formatReferenceBookMetadata(book),
	})
	systemPrompt := SystemPromptFor(cfg.Language, "reference_analysis_json")
	rawResp := CallAPIWithRetryLog(ctx, apiCfg, systemPrompt, userPrompt, logger)
	if rawResp == "" {
		return nil, fmt.Errorf("参考全书分析调用失败或被取消")
	}
	rawResp = cleanJSONResponse(rawResp)
	var analysis ReferenceAnalysis
	if err := json.Unmarshal([]byte(rawResp), &analysis); err != nil {
		return nil, fmt.Errorf("解析参考全书分析 JSON 失败: %w", err)
	}
	if strings.TrimSpace(analysis.Title) == "" {
		analysis.Title = book.Title
	}
	return &analysis, nil
}

func ApplyReferenceSettingsImport(settings *ProjectSettings, analysis *ReferenceAnalysis) int {
	if settings == nil || analysis == nil {
		return 0
	}
	count := applyReferenceSettingsCandidate(settings, analysis.Settings)
	if count > 0 {
		analysis.SettingsImportStatus = ReferenceSettingsStatusApplied
	}
	return count
}

func applyReferenceSettingsCandidate(settings *ProjectSettings, candidate ReferenceSettingsCandidate) int {
	if settings == nil {
		return 0
	}
	count := 0
	nameToID := buildSettingsNameIndex(settings)

	for _, seed := range candidate.Characters {
		name := strings.TrimSpace(seed.Name)
		if name == "" || nameToID[name] != "" {
			continue
		}
		c := Character{
			ID:          settings.nextCharacterID(),
			Name:        name,
			Age:         strings.TrimSpace(seed.Age),
			Appearance:  strings.TrimSpace(seed.Appearance),
			Personality: strings.TrimSpace(seed.Personality),
			Background:  strings.TrimSpace(seed.Background),
			Motivation:  strings.TrimSpace(seed.Motivation),
			Abilities:   strings.TrimSpace(seed.Abilities),
			Notes:       strings.TrimSpace(seed.Notes),
		}
		settings.Characters = append(settings.Characters, c)
		nameToID[c.Name] = c.ID
		count++
	}

	for _, seed := range candidate.Worldview {
		name := strings.TrimSpace(seed.Name)
		if name == "" || strings.TrimSpace(seed.Description) == "" || nameToID[name] != "" {
			continue
		}
		wv := WorldviewEntry{
			ID:          settings.nextWorldviewID(),
			Category:    defaultString(strings.TrimSpace(seed.Category), "设定"),
			Name:        name,
			Description: strings.TrimSpace(seed.Description),
			Tags:        strings.TrimSpace(seed.Tags),
		}
		settings.Worldview = append(settings.Worldview, wv)
		nameToID[wv.Name] = wv.ID
		count++
	}

	for _, seed := range candidate.Organizations {
		name := strings.TrimSpace(seed.Name)
		if name == "" || nameToID[name] != "" {
			continue
		}
		var memberIDs []string
		for _, memberName := range seed.MemberNames {
			if id := nameToID[strings.TrimSpace(memberName)]; id != "" {
				memberIDs = append(memberIDs, id)
			}
		}
		org := Organization{
			ID:          settings.nextOrganizationID(),
			Name:        name,
			Type:        strings.TrimSpace(seed.Type),
			Description: strings.TrimSpace(seed.Description),
			Members:     memberIDs,
		}
		settings.Organizations = append(settings.Organizations, org)
		nameToID[org.Name] = org.ID
		count++
	}

	for _, seed := range candidate.Relations {
		sourceName := strings.TrimSpace(seed.SourceName)
		targetName := strings.TrimSpace(seed.TargetName)
		label := strings.TrimSpace(seed.Label)
		sourceID := nameToID[sourceName]
		targetID := nameToID[targetName]
		if sourceID == "" || targetID == "" || label == "" || relationExists(settings, sourceID, targetID, label) {
			continue
		}
		settings.Relations = append(settings.Relations, Relation{
			ID:         settings.nextRelationID(),
			SourceID:   sourceID,
			SourceType: normalizeRelationEntityType(seed.SourceType),
			TargetID:   targetID,
			TargetType: normalizeRelationEntityType(seed.TargetType),
			Label:      label,
		})
		count++
	}
	return count
}

func referenceAnalysisChunkRunes(apiCfg *APIConfig) int {
	budget := getContextBudget(apiCfg)
	usableRunes := int(float64(budget) / 1.5 * 0.25)
	if usableRunes < 8000 {
		return 8000
	}
	if usableRunes > referenceChapterSplitRunes {
		return referenceChapterSplitRunes
	}
	return usableRunes
}

func mergeReferenceChapterAnalyses(ch ReferenceChapter, parts []ReferenceChapterAnalysis) ReferenceChapterAnalysis {
	if len(parts) == 0 {
		return ReferenceChapterAnalysis{Num: ch.Num, Title: ch.Title}
	}
	if len(parts) == 1 {
		parts[0].Num = ch.Num
		if strings.TrimSpace(parts[0].Title) == "" {
			parts[0].Title = ch.Title
		}
		return parts[0]
	}
	merged := ReferenceChapterAnalysis{Num: ch.Num, Title: ch.Title}
	var summaries []string
	for _, part := range parts {
		appendUniqueStrings(&merged.KeyEvents, part.KeyEvents)
		appendUniqueStrings(&merged.ForeshadowPayoffs, part.ForeshadowPayoffs)
		appendUniqueStrings(&merged.Characters, part.Characters)
		if strings.TrimSpace(part.Summary) != "" {
			summaries = append(summaries, strings.TrimSpace(part.Summary))
		}
		if merged.SceneFunction == "" && part.SceneFunction != "" {
			merged.SceneFunction = part.SceneFunction
		}
		if merged.EmotionalCurve == "" && part.EmotionalCurve != "" {
			merged.EmotionalCurve = part.EmotionalCurve
		} else if part.EmotionalCurve != "" && !strings.Contains(merged.EmotionalCurve, part.EmotionalCurve) {
			merged.EmotionalCurve += " / " + part.EmotionalCurve
		}
		if part.EndingRoute != "" {
			merged.EndingRoute = part.EndingRoute
		}
	}
	merged.Summary = strings.Join(summaries, "\n")
	return merged
}

func appendUniqueStrings(dst *[]string, values []string) {
	seen := make(map[string]bool)
	for _, existing := range *dst {
		seen[existing] = true
	}
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		*dst = append(*dst, v)
		seen[v] = true
	}
}

func hasReferenceSettingsCandidate(candidate ReferenceSettingsCandidate) bool {
	return len(candidate.Characters) > 0 || len(candidate.Worldview) > 0 || len(candidate.Organizations) > 0 || len(candidate.Relations) > 0
}

func isProjectSettingsEmpty(settings *ProjectSettings) bool {
	return settings == nil ||
		(len(settings.Characters) == 0 &&
			len(settings.Worldview) == 0 &&
			len(settings.Organizations) == 0 &&
			len(settings.Relations) == 0)
}

func buildSettingsNameIndex(settings *ProjectSettings) map[string]string {
	index := make(map[string]string)
	if settings == nil {
		return index
	}
	for _, c := range settings.Characters {
		index[c.Name] = c.ID
	}
	for _, wv := range settings.Worldview {
		index[wv.Name] = wv.ID
	}
	for _, org := range settings.Organizations {
		index[org.Name] = org.ID
	}
	return index
}

func relationExists(settings *ProjectSettings, sourceID, targetID, label string) bool {
	for _, rel := range settings.Relations {
		if rel.SourceID == sourceID && rel.TargetID == targetID && rel.Label == label {
			return true
		}
	}
	return false
}

func normalizeRelationEntityType(entityType string) string {
	switch strings.TrimSpace(entityType) {
	case "organization", "org", "组织":
		return "organization"
	case "worldview", "setting", "设定", "世界观":
		return "worldview"
	default:
		return "character"
	}
}

func extractReferenceChapterTitle(segment string, num int) string {
	for _, line := range strings.Split(segment, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "#"))
		if line != "" {
			return line
		}
	}
	return defaultReferenceChapterTitle(num)
}

func defaultReferenceChapterTitle(num int) string {
	return fmt.Sprintf("Chapter %d", num)
}

func guessReferenceBookTitle(sourceName string, firstChapterTitle string) string {
	sourceName = strings.TrimSpace(sourceName)
	if sourceName != "" {
		base := strings.TrimSuffix(filepath.Base(sourceName), filepath.Ext(sourceName))
		if base != "" {
			return base
		}
	}
	return strings.TrimSpace(firstChapterTitle)
}

func estimateWordCount(text string) int {
	fields := strings.Fields(text)
	if len(fields) > 1 {
		return len(fields)
	}
	return len([]rune(strings.TrimSpace(text)))
}

func findReferenceChapter(book *ReferenceBook, num int) *ReferenceChapter {
	if book == nil {
		return nil
	}
	for i := range book.Chapters {
		if book.Chapters[i].Num == num {
			return &book.Chapters[i]
		}
	}
	return nil
}

func formatReferenceBookMetadata(book *ReferenceBook) string {
	if book == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("标题: %s\n", book.Title))
	sb.WriteString(fmt.Sprintf("章节数: %d\n", len(book.Chapters)))
	sb.WriteString(fmt.Sprintf("总字数估算: %d\n", book.TotalRunes))
	if book.SourceName != "" {
		sb.WriteString(fmt.Sprintf("来源文件: %s\n", book.SourceName))
	}
	return sb.String()
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
