package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

const (
	RewriteRequestTypeGlobal       = "global"
	RewriteRequestTypeChapter      = "chapter"
	RewriteRequestTypeRange        = "range"
	RewriteRequestTypeCharacter    = "character"
	RewriteRequestTypeSetting      = "setting"
	RewriteRequestTypeRelationship = "relationship"
	RewriteRequestTypeEnding       = "ending"
	RewriteRequestTypeForbidden    = "forbidden"

	RewritePlanStatusDraft     = "draft"
	RewritePlanStatusGenerated = "generated"
	RewritePlanStatusConfirmed = "confirmed"

	ChapterMappingOneToOne = "one_to_one"
	ChapterMappingMerge    = "merge"
	ChapterMappingSplit    = "split"

	rewritePlanMaterialSplitRunes = 90000
)

type RewriteRequest struct {
	ID               string `json:"id"`
	Type             string `json:"type"`
	Scope            string `json:"scope,omitempty"`
	ChapterNum       int    `json:"chapter_num,omitempty"`
	ChapterStart     int    `json:"chapter_start,omitempty"`
	ChapterEnd       int    `json:"chapter_end,omitempty"`
	Object           string `json:"object,omitempty"`
	Instruction      string `json:"instruction"`
	Intensity        string `json:"intensity,omitempty"`
	AffectsFollowing bool   `json:"affects_following"`
	Priority         string `json:"priority,omitempty"`
	CreatedAt        string `json:"created_at,omitempty"`
	UpdatedAt        string `json:"updated_at,omitempty"`
}

type ChapterMapping struct {
	TargetChapterNum  int    `json:"target_chapter_num"`
	SourceChapterNums []int  `json:"source_chapter_nums"`
	MappingType       string `json:"mapping_type"`
}

type RewritePlan struct {
	Title               string                 `json:"title,omitempty"`
	Status              string                 `json:"status,omitempty"`
	GeneratedAt         string                 `json:"generated_at,omitempty"`
	ConfirmedAt         string                 `json:"confirmed_at,omitempty"`
	GlobalDirection     string                 `json:"global_direction,omitempty"`
	CorePremise         string                 `json:"core_premise,omitempty"`
	StyleGuide          string                 `json:"style_guide,omitempty"`
	CharacterChanges    []RewriteChangeItem    `json:"character_changes,omitempty"`
	SettingChanges      []RewriteChangeItem    `json:"setting_changes,omitempty"`
	RelationshipChanges []RewriteChangeItem    `json:"relationship_changes,omitempty"`
	RequestImpacts      []RewriteRequestImpact `json:"request_impacts,omitempty"`
	Mappings            []ChapterMapping       `json:"mappings,omitempty"`
	Chapters            []RewriteChapterPlan   `json:"chapters,omitempty"`
	Constraints         []string               `json:"constraints,omitempty"`
}

type RewriteChangeItem struct {
	Object        string `json:"object"`
	Before        string `json:"before,omitempty"`
	After         string `json:"after"`
	AffectedChaps []int  `json:"affected_chapters,omitempty"`
}

type RewriteRequestImpact struct {
	RequestID        string   `json:"request_id"`
	Summary          string   `json:"summary"`
	AffectedChapters []int    `json:"affected_chapters,omitempty"`
	AffectedObjects  []string `json:"affected_objects,omitempty"`
}

type RewriteChapterPlan struct {
	Num                  int      `json:"num"`
	Title                string   `json:"title"`
	Outline              string   `json:"outline"`
	SourceChapterNums    []int    `json:"source_chapter_nums"`
	MappingType          string   `json:"mapping_type"`
	PreservedEvents      []string `json:"preserved_events,omitempty"`
	ChangedEvents        []string `json:"changed_events,omitempty"`
	ForbiddenClosePoints []string `json:"forbidden_close_points,omitempty"`
	RequestIDs           []string `json:"request_ids,omitempty"`
	UseOriginalFullText  bool     `json:"use_original_full_text"`
	FullTextReason       string   `json:"full_text_reason,omitempty"`
}

func LoadRewriteRequests(path string) ([]RewriteRequest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []RewriteRequest{}, nil
		}
		return nil, fmt.Errorf("读取改写意见文件失败: %w", err)
	}
	var requests []RewriteRequest
	if err := json.Unmarshal(data, &requests); err != nil {
		return nil, fmt.Errorf("解析改写意见文件失败: %w", err)
	}
	return requests, nil
}

func SaveRewriteRequests(path string, requests []RewriteRequest) error {
	data, err := json.MarshalIndent(requests, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化改写意见失败: %w", err)
	}
	return writeFileAtomic(path, data)
}

func LoadRewritePlan(path string) (*RewritePlan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &RewritePlan{Status: RewritePlanStatusDraft}, nil
		}
		return nil, fmt.Errorf("读取改编方案文件失败: %w", err)
	}
	var plan RewritePlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("解析改编方案文件失败: %w", err)
	}
	if plan.Status == "" {
		plan.Status = RewritePlanStatusDraft
	}
	return &plan, nil
}

func SaveRewritePlan(path string, plan *RewritePlan) error {
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化改编方案失败: %w", err)
	}
	return writeFileAtomic(path, data)
}

func NextRewriteRequestID(requests []RewriteRequest) string {
	maxNum := 0
	for _, req := range requests {
		if strings.HasPrefix(req.ID, "rr_") {
			var n int
			if _, err := fmt.Sscanf(strings.TrimPrefix(req.ID, "rr_"), "%d", &n); err == nil && n > maxNum {
				maxNum = n
			}
		}
	}
	return fmt.Sprintf("rr_%d", maxNum+1)
}

func NormalizeRewriteRequest(req RewriteRequest) RewriteRequest {
	req.Type = normalizeRewriteRequestType(req.Type)
	req.Priority = normalizeRewritePriority(req.Priority)
	req.Intensity = normalizeRewriteIntensity(req.Intensity)
	req.Instruction = strings.TrimSpace(req.Instruction)
	req.Object = strings.TrimSpace(req.Object)
	req.Scope = strings.TrimSpace(req.Scope)
	if req.Type == RewriteRequestTypeChapter && req.ChapterNum > 0 {
		req.ChapterStart = req.ChapterNum
		req.ChapterEnd = req.ChapterNum
	}
	if req.Type == RewriteRequestTypeRange {
		if req.ChapterStart <= 0 && req.ChapterNum > 0 {
			req.ChapterStart = req.ChapterNum
		}
		if req.ChapterEnd <= 0 {
			req.ChapterEnd = req.ChapterStart
		}
		if req.ChapterEnd < req.ChapterStart {
			req.ChapterStart, req.ChapterEnd = req.ChapterEnd, req.ChapterStart
		}
	}
	return req
}

func ValidateRewriteRequest(req RewriteRequest) error {
	if req.Instruction == "" {
		return fmt.Errorf("改写意见不能为空")
	}
	switch req.Type {
	case RewriteRequestTypeChapter:
		if req.ChapterNum <= 0 {
			return fmt.Errorf("单章意见必须填写章节号")
		}
	case RewriteRequestTypeRange:
		if req.ChapterStart <= 0 || req.ChapterEnd <= 0 {
			return fmt.Errorf("章节范围意见必须填写起止章节")
		}
	case RewriteRequestTypeCharacter, RewriteRequestTypeSetting, RewriteRequestTypeRelationship:
		if req.Object == "" {
			return fmt.Errorf("该类型意见必须填写对象")
		}
	}
	return nil
}

func GenerateRewritePlanAction(ctx context.Context, apiCfg *APIConfig, cfg *Config, reference *ReferenceBook, analysis *ReferenceAnalysis, requests []RewriteRequest, logger *LogBroadcaster) (*RewritePlan, error) {
	if err := validateAPIConfig(apiCfg); err != nil {
		return nil, err
	}
	if reference == nil || len(reference.Chapters) == 0 {
		return nil, fmt.Errorf("请先导入参考小说")
	}
	if analysis == nil || len(analysis.Chapters) == 0 {
		return nil, fmt.Errorf("请先完成参考分析")
	}
	if len(requests) == 0 {
		return nil, fmt.Errorf("请先添加改写意见")
	}

	material, err := buildRewritePlanMaterial(reference, analysis, requests)
	if err != nil {
		return nil, err
	}

	planNotes := ""
	parts := splitTextByRunes(material, rewritePlanMaterialSplitRunes)
	if len(parts) > 1 {
		logger.Info(fmt.Sprintf("改编方案材料较长，先分 %d 段提取规划要点...", len(parts)))
		var notes []string
		for i, part := range parts {
			if ctx.Err() != nil {
				return nil, fmt.Errorf("任务已取消")
			}
			logger.StepInfo(i+1, len(parts)+1, fmt.Sprintf("正在提取第 %d/%d 段规划要点...", i+1, len(parts)))
			note, err := generateRewritePlanChunkNote(ctx, apiCfg, cfg, part, i+1, len(parts), logger)
			if err != nil {
				return nil, err
			}
			notes = append(notes, note)
		}
		planNotes = strings.Join(notes, "\n\n---\n\n")
		material = "参考材料较长，以下为分段规划要点：\n\n" + planNotes
	}

	logger.StepInfo(len(parts)+1, len(parts)+1, "正在生成改编总方案...")
	plan, err := generateRewritePlan(ctx, apiCfg, cfg, reference, analysis, requests, material, logger)
	if err != nil {
		return nil, err
	}
	normalizeRewritePlan(plan, reference)
	if err := ValidateRewritePlanMappings(plan, reference); err != nil {
		return nil, err
	}
	plan.Status = RewritePlanStatusGenerated
	plan.GeneratedAt = time.Now().Format(time.RFC3339)
	return plan, nil
}

func generateRewritePlanChunkNote(ctx context.Context, apiCfg *APIConfig, cfg *Config, material string, idx, total int, logger *LogBroadcaster) (string, error) {
	userPrompt := RenderPrompt(cfg.Prompts.RewritePlanChunkAnalysis, map[string]string{
		"ChunkIndex": fmt.Sprintf("%d", idx),
		"ChunkTotal": fmt.Sprintf("%d", total),
		"Material":   material,
	})
	systemPrompt := SystemPromptFor(cfg.Language, "rewrite_planner")
	resp := CallAPIWithRetryLog(ctx, apiCfg, systemPrompt, userPrompt, logger)
	if resp == "" {
		return "", fmt.Errorf("改编方案分段分析失败或被取消")
	}
	return strings.TrimSpace(resp), nil
}

func generateRewritePlan(ctx context.Context, apiCfg *APIConfig, cfg *Config, reference *ReferenceBook, analysis *ReferenceAnalysis, requests []RewriteRequest, material string, logger *LogBroadcaster) (*RewritePlan, error) {
	requestsJSON, _ := json.MarshalIndent(requests, "", "  ")
	userPrompt := RenderPrompt(cfg.Prompts.RewritePlanGeneration, map[string]string{
		"ReferenceTitle":       reference.Title,
		"SourceChapterCount":   fmt.Sprintf("%d", len(reference.Chapters)),
		"ReferenceSynopsis":    analysis.Synopsis,
		"ReferenceCoreSetting": analysis.CoreSetting,
		"RewriteRequests":      string(requestsJSON),
		"PlanningMaterial":     material,
	})
	systemPrompt := SystemPromptFor(cfg.Language, "rewrite_planner_json")
	rawResp := CallAPIWithRetryLog(ctx, apiCfg, systemPrompt, userPrompt, logger)
	if rawResp == "" {
		return nil, fmt.Errorf("改编总方案生成失败或被取消")
	}
	rawResp = cleanJSONResponse(rawResp)
	var plan RewritePlan
	if err := json.Unmarshal([]byte(rawResp), &plan); err != nil {
		return nil, fmt.Errorf("解析改编总方案 JSON 失败: %w", err)
	}
	return &plan, nil
}

func buildRewritePlanMaterial(reference *ReferenceBook, analysis *ReferenceAnalysis, requests []RewriteRequest) (string, error) {
	if reference == nil || analysis == nil {
		return "", fmt.Errorf("参考材料缺失")
	}
	compact := struct {
		Reference struct {
			Title       string                     `json:"title"`
			StoryType   string                     `json:"story_type,omitempty"`
			Synopsis    string                     `json:"synopsis,omitempty"`
			CoreSetting string                     `json:"core_setting,omitempty"`
			GlobalNotes string                     `json:"global_notes,omitempty"`
			Chapters    []ReferenceChapterAnalysis `json:"chapters"`
		} `json:"reference"`
		SourceChapters []ReferenceChapter `json:"source_chapters"`
		Requests       []RewriteRequest   `json:"requests"`
	}{}
	compact.Reference.Title = reference.Title
	compact.Reference.StoryType = analysis.StoryType
	compact.Reference.Synopsis = analysis.Synopsis
	compact.Reference.CoreSetting = analysis.CoreSetting
	compact.Reference.GlobalNotes = analysis.GlobalNotes
	compact.Reference.Chapters = analysis.Chapters
	compact.SourceChapters = reference.Chapters
	compact.Requests = requests
	data, err := json.MarshalIndent(compact, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ValidateRewritePlanMappings(plan *RewritePlan, reference *ReferenceBook) error {
	if plan == nil || len(plan.Chapters) == 0 {
		return fmt.Errorf("改编方案没有章节计划")
	}
	if reference == nil || len(reference.Chapters) == 0 {
		return fmt.Errorf("参考章节缺失")
	}
	sourceSet := make(map[int]bool)
	for _, ch := range reference.Chapters {
		sourceSet[ch.Num] = true
	}
	covered := make(map[int]bool)
	targetSeen := make(map[int]bool)
	for _, ch := range plan.Chapters {
		if ch.Num <= 0 {
			return fmt.Errorf("新稿章节编号无效")
		}
		if targetSeen[ch.Num] {
			return fmt.Errorf("新稿第 %d 章重复", ch.Num)
		}
		targetSeen[ch.Num] = true
		if strings.TrimSpace(ch.Title) == "" || strings.TrimSpace(ch.Outline) == "" {
			return fmt.Errorf("新稿第 %d 章缺少标题或大纲", ch.Num)
		}
		if len(ch.SourceChapterNums) == 0 {
			return fmt.Errorf("新稿第 %d 章缺少原文章节映射", ch.Num)
		}
		for _, sourceNum := range ch.SourceChapterNums {
			if !sourceSet[sourceNum] {
				return fmt.Errorf("新稿第 %d 章映射了不存在的原文章节 %d", ch.Num, sourceNum)
			}
			covered[sourceNum] = true
		}
	}
	for sourceNum := range sourceSet {
		if !covered[sourceNum] {
			return fmt.Errorf("原文第 %d 章未被任何新稿章节覆盖", sourceNum)
		}
	}
	return nil
}

func ConfirmRewritePlan(plan *RewritePlan, reference *ReferenceBook, cfg *Config, state *Progress, progressPath string, planPath string) error {
	if err := ValidateRewritePlanMappings(plan, reference); err != nil {
		return err
	}
	if state == nil {
		return fmt.Errorf("进度状态为空")
	}
	for _, ch := range state.Chapters {
		if ch.Content != "" || ch.Status == StatusAccepted || ch.Status == StatusReview || ch.Status == StatusWriting {
			return fmt.Errorf("已存在写作中或已完成的新稿章节，不能直接覆盖改编方案骨架")
		}
	}

	state.Title = defaultString(plan.Title, defaultRewriteTitle(reference.Title))
	state.CorePrompt = buildRewriteCorePrompt(plan)
	state.StorySynopsis = defaultString(plan.CorePremise, plan.GlobalDirection)
	state.Chapters = make([]ChapterState, 0, len(plan.Chapters))
	for _, ch := range plan.Chapters {
		state.Chapters = append(state.Chapters, ChapterState{
			Num:     ch.Num,
			Title:   ch.Title,
			Outline: rewriteChapterPlanOutline(ch),
			Status:  StatusPending,
		})
	}
	state.CurrentChapterIndex = 0
	state.Phase = "writing"
	snapshot := cfg.Story
	snapshot.Title = state.Title
	snapshot.StorySynopsis = state.StorySynopsis
	snapshot.ChapterCount = len(state.Chapters)
	state.StoryConfigSnapshot = &snapshot

	plan.Status = RewritePlanStatusConfirmed
	plan.ConfirmedAt = time.Now().Format(time.RFC3339)
	if err := SaveProgress(progressPath, state); err != nil {
		return err
	}
	if err := SaveRewritePlan(planPath, plan); err != nil {
		return err
	}
	return nil
}

func normalizeRewritePlan(plan *RewritePlan, reference *ReferenceBook) {
	if plan == nil {
		return
	}
	if plan.Status == "" {
		plan.Status = RewritePlanStatusGenerated
	}
	sort.SliceStable(plan.Chapters, func(i, j int) bool {
		return plan.Chapters[i].Num < plan.Chapters[j].Num
	})
	for i := range plan.Chapters {
		ch := &plan.Chapters[i]
		if ch.Num <= 0 {
			ch.Num = i + 1
		}
		if ch.MappingType == "" {
			ch.MappingType = inferMappingType(ch.SourceChapterNums, plan.Chapters)
		}
		if len(ch.SourceChapterNums) == 0 && reference != nil && i < len(reference.Chapters) {
			ch.SourceChapterNums = []int{reference.Chapters[i].Num}
		}
	}
	if len(plan.Mappings) == 0 {
		for _, ch := range plan.Chapters {
			plan.Mappings = append(plan.Mappings, ChapterMapping{
				TargetChapterNum:  ch.Num,
				SourceChapterNums: append([]int(nil), ch.SourceChapterNums...),
				MappingType:       defaultString(ch.MappingType, inferMappingType(ch.SourceChapterNums, plan.Chapters)),
			})
		}
	}
	sort.SliceStable(plan.Mappings, func(i, j int) bool {
		return plan.Mappings[i].TargetChapterNum < plan.Mappings[j].TargetChapterNum
	})
}

func inferMappingType(sourceNums []int, chapters []RewriteChapterPlan) string {
	if len(sourceNums) > 1 {
		return ChapterMappingMerge
	}
	if len(sourceNums) == 1 {
		count := 0
		for _, ch := range chapters {
			if containsInt(ch.SourceChapterNums, sourceNums[0]) {
				count++
			}
		}
		if count > 1 {
			return ChapterMappingSplit
		}
	}
	return ChapterMappingOneToOne
}

func rewriteChapterPlanOutline(ch RewriteChapterPlan) string {
	var sb strings.Builder
	sb.WriteString(strings.TrimSpace(ch.Outline))
	if len(ch.PreservedEvents) > 0 {
		sb.WriteString("\n\n【保留事件功能】\n")
		for _, item := range ch.PreservedEvents {
			sb.WriteString("- ")
			sb.WriteString(item)
			sb.WriteString("\n")
		}
	}
	if len(ch.ChangedEvents) > 0 {
		sb.WriteString("\n【改写变化】\n")
		for _, item := range ch.ChangedEvents {
			sb.WriteString("- ")
			sb.WriteString(item)
			sb.WriteString("\n")
		}
	}
	if len(ch.ForbiddenClosePoints) > 0 {
		sb.WriteString("\n【禁止贴近点】\n")
		for _, item := range ch.ForbiddenClosePoints {
			sb.WriteString("- ")
			sb.WriteString(item)
			sb.WriteString("\n")
		}
	}
	sb.WriteString(fmt.Sprintf("\n【原文章节映射】%s\n", joinInts(ch.SourceChapterNums, ", ")))
	return strings.TrimSpace(sb.String())
}

func buildRewriteCorePrompt(plan *RewritePlan) string {
	var parts []string
	if plan.GlobalDirection != "" {
		parts = append(parts, "改写总方向："+plan.GlobalDirection)
	}
	if plan.CorePremise != "" {
		parts = append(parts, "新稿核心设定："+plan.CorePremise)
	}
	if plan.StyleGuide != "" {
		parts = append(parts, "新稿风格："+plan.StyleGuide)
	}
	if len(plan.Constraints) > 0 {
		parts = append(parts, "全篇一致性约束：\n- "+strings.Join(plan.Constraints, "\n- "))
	}
	return strings.Join(parts, "\n\n")
}

func defaultRewriteTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return "改写新稿"
	}
	return title + "（改写稿）"
}

func containsInt(values []int, target int) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

func joinInts(values []int, sep string) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(parts, sep)
}

func normalizeRewriteRequestType(t string) string {
	switch strings.TrimSpace(t) {
	case RewriteRequestTypeChapter, RewriteRequestTypeRange, RewriteRequestTypeCharacter, RewriteRequestTypeSetting, RewriteRequestTypeRelationship, RewriteRequestTypeEnding, RewriteRequestTypeForbidden:
		return strings.TrimSpace(t)
	default:
		return RewriteRequestTypeGlobal
	}
}

func normalizeRewritePriority(p string) string {
	switch strings.ToUpper(strings.TrimSpace(p)) {
	case "P0", "P1", "P2":
		return strings.ToUpper(strings.TrimSpace(p))
	default:
		return "P1"
	}
}

func normalizeRewriteIntensity(v string) string {
	switch strings.TrimSpace(v) {
	case "light", "medium", "heavy":
		return strings.TrimSpace(v)
	default:
		return "medium"
	}
}
