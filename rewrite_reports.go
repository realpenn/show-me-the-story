package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

func BuildRewriteReportsAction(ctx context.Context, cfg *Config, settings *ProjectSettings, state *Progress, reference *ReferenceBook, analysis *ReferenceAnalysis, plan *RewritePlan, requests []RewriteRequest) (*RewriteReportState, error) {
	if ctx != nil && ctx.Err() != nil {
		return nil, fmt.Errorf("任务已取消")
	}
	if cfg == nil || NormalizeProjectType(cfg.ProjectType) != ProjectTypeRewrite {
		return nil, fmt.Errorf("当前项目不是改写项目")
	}
	if state == nil || plan == nil || plan.Status != RewritePlanStatusConfirmed {
		return nil, fmt.Errorf("请先确认改编总方案")
	}
	if reference == nil || len(reference.Chapters) == 0 {
		return nil, fmt.Errorf("参考小说缺失")
	}

	now := time.Now().Format(time.RFC3339)
	reports := &RewriteReportState{
		GeneratedAt:      now,
		RequestReport:    buildRewriteRequestReport(cfg.Language, state, plan, requests, now),
		StructureReport:  buildRewriteStructureReport(cfg.Language, state, reference, analysis, plan, now),
		SimilarityReport: buildRewriteSimilarityReport(cfg.Language, plan, now),
		SettingsReport:   buildRewriteSettingsReport(cfg.Language, settings, analysis, plan, now),
	}
	return reports, nil
}

func buildRewriteRequestReport(lang string, state *Progress, plan *RewritePlan, requests []RewriteRequest, generatedAt string) string {
	var sb strings.Builder
	writeReportTitle(&sb, lang, "改写意见落实报告", "Rewrite Request Implementation Report", generatedAt)

	accepted, total := acceptedChapterCount(state)
	checks, passedChecks := rewriteCheckCount(plan)
	sb.WriteString(reportHeading(lang, "总览", "Overview", 2))
	writeReportBullet(&sb, lang, "改写意见", "Rewrite requests", fmt.Sprintf("%d", len(requests)))
	writeReportBullet(&sb, lang, "新稿章节", "New manuscript chapters", fmt.Sprintf("%d", total))
	writeReportBullet(&sb, lang, "已确认章节", "Accepted chapters", fmt.Sprintf("%d/%d", accepted, total))
	writeReportBullet(&sb, lang, "有检查记录章节", "Chapters with checks", fmt.Sprintf("%d", checks))
	writeReportBullet(&sb, lang, "检查通过章节", "Chapters passing checks", fmt.Sprintf("%d", passedChecks))

	if len(requests) == 0 {
		sb.WriteString(reportEmptyLine(lang, "暂无改写意见。", "No rewrite requests recorded."))
		return sb.String()
	}

	impacts := rewriteImpactMap(plan)
	sb.WriteString(reportHeading(lang, "意见清单", "Request List", 2))
	for _, req := range requests {
		if strings.TrimSpace(req.ID) == "" {
			continue
		}
		title := fmt.Sprintf("%s [%s]", req.ID, rewriteRequestTypeLabel(req.Type, lang))
		if req.Priority != "" {
			title += " " + req.Priority
		}
		sb.WriteString(reportHeading(lang, title, title, 3))
		writeReportBullet(&sb, lang, "指令", "Instruction", req.Instruction)
		writeReportBullet(&sb, lang, "作用范围", "Scope", rewriteRequestScopeText(req, lang))
		writeReportBullet(&sb, lang, "影响章节", "Affected chapters", reportChapterList(rewriteRequestAffectedChapters(plan, state, req, impacts)))
		if impact := impacts[req.ID]; impact != nil {
			writeReportBullet(&sb, lang, "方案说明", "Plan note", impact.Summary)
			if len(impact.AffectedObjects) > 0 {
				writeReportBullet(&sb, lang, "影响对象", "Affected objects", strings.Join(impact.AffectedObjects, ", "))
			}
		}
		writeReportBullet(&sb, lang, "落实状态", "Implementation status", rewriteRequestImplementationStatus(plan, state, req, impacts, lang))
		sb.WriteString("\n")
	}

	sb.WriteString(reportHeading(lang, "导出交付", "Export Deliverables", 2))
	writeReportBullet(&sb, lang, "TXT", "TXT", reportText(lang, "写作页可导出当前完整新稿 TXT。", "The Writing page exports the current complete new manuscript as TXT."))
	writeReportBullet(&sb, lang, "Markdown", "Markdown", reportText(lang, "每章确认/修订后会同步保存为项目目录内的 Markdown 章节文件。", "Each accepted/revised chapter is saved as a Markdown chapter file in the project directory."))
	return sb.String()
}

func buildRewriteStructureReport(lang string, state *Progress, reference *ReferenceBook, analysis *ReferenceAnalysis, plan *RewritePlan, generatedAt string) string {
	var sb strings.Builder
	writeReportTitle(&sb, lang, "原文结构保真报告", "Source Structure Fidelity Report", generatedAt)

	mappingCounts := map[string]int{}
	for _, ch := range plan.Chapters {
		mappingCounts[defaultString(ch.MappingType, ChapterMappingOneToOne)]++
	}

	sb.WriteString(reportHeading(lang, "总览", "Overview", 2))
	writeReportBullet(&sb, lang, "原文章节", "Source chapters", fmt.Sprintf("%d", len(reference.Chapters)))
	writeReportBullet(&sb, lang, "新稿章节", "New manuscript chapters", fmt.Sprintf("%d", len(plan.Chapters)))
	writeReportBullet(&sb, lang, "一对一映射", "One-to-one mappings", fmt.Sprintf("%d", mappingCounts[ChapterMappingOneToOne]))
	writeReportBullet(&sb, lang, "合并映射", "Merge mappings", fmt.Sprintf("%d", mappingCounts[ChapterMappingMerge]))
	writeReportBullet(&sb, lang, "拆分映射", "Split mappings", fmt.Sprintf("%d", mappingCounts[ChapterMappingSplit]))

	sb.WriteString(reportHeading(lang, "逐章结构", "Chapter Structure", 2))
	for _, planCh := range plan.Chapters {
		title := fmt.Sprintf("%s %d %s", reportText(lang, "新稿第", "New Ch."), planCh.Num, strings.TrimSpace(planCh.Title))
		sb.WriteString(reportHeading(lang, title, title, 3))
		writeReportBullet(&sb, lang, "原文章节映射", "Source mapping", reportChapterList(planCh.SourceChapterNums))
		writeReportBullet(&sb, lang, "映射类型", "Mapping type", rewriteMappingTypeLabel(planCh.MappingType, lang))
		writeReportBullet(&sb, lang, "新稿大纲", "New outline", planCh.Outline)
		writeReportList(&sb, lang, "保留事件功能", "Preserved event functions", planCh.PreservedEvents)
		writeReportList(&sb, lang, "改写变化", "Changed events", planCh.ChangedEvents)
		writeReportList(&sb, lang, "禁止贴近点", "Forbidden close points", planCh.ForbiddenClosePoints)
		if sourceNotes := referenceAnalysisSummaryForReport(lang, analysis, planCh.SourceChapterNums); sourceNotes != "" {
			writeReportBullet(&sb, lang, "原文结构摘要", "Source structure summary", sourceNotes)
		}
		if check := latestRewriteCheckForChapter(plan, planCh.Num); check != nil {
			writeReportBullet(&sb, lang, "结构检查", "Structure check", rewriteAICheckSummary(check.Structure, lang))
		} else {
			writeReportBullet(&sb, lang, "结构检查", "Structure check", reportText(lang, "暂无检查记录", "No check recorded"))
		}
		if stateCh := chapterStateByNum(state, planCh.Num); stateCh != nil {
			writeReportBullet(&sb, lang, "章节状态", "Chapter status", stateCh.Status)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func buildRewriteSimilarityReport(lang string, plan *RewritePlan, generatedAt string) string {
	var sb strings.Builder
	writeReportTitle(&sb, lang, "相似度风险报告", "Similarity Risk Report", generatedAt)

	var fullSource []RewriteChapterPlan
	var highRisk []RewriteChapterPlan
	var mediumRisk []RewriteChapterPlan
	checked := 0
	passed := 0
	for _, ch := range plan.Chapters {
		if ch.UseOriginalFullText {
			fullSource = append(fullSource, ch)
		}
		check := latestRewriteCheckForChapter(plan, ch.Num)
		if check == nil {
			continue
		}
		checked++
		if check.Passed {
			passed++
		}
		risk := check.Closeness.Deterministic.RiskLevel
		if risk == "high" || strings.EqualFold(check.Closeness.Result, "FAIL") {
			highRisk = append(highRisk, ch)
		} else if risk == "medium" {
			mediumRisk = append(mediumRisk, ch)
		}
	}

	sb.WriteString(reportHeading(lang, "总览", "Overview", 2))
	writeReportBullet(&sb, lang, "检查章节", "Checked chapters", fmt.Sprintf("%d/%d", checked, len(plan.Chapters)))
	writeReportBullet(&sb, lang, "通过章节", "Passed chapters", fmt.Sprintf("%d", passed))
	writeReportBullet(&sb, lang, "高风险章节", "High-risk chapters", fmt.Sprintf("%d", len(highRisk)))
	writeReportBullet(&sb, lang, "中风险章节", "Medium-risk chapters", fmt.Sprintf("%d", len(mediumRisk)))
	writeReportBullet(&sb, lang, "使用原文全文参考章节", "Full-source reference chapters", fmt.Sprintf("%d", len(fullSource)))

	sb.WriteString(reportHeading(lang, "使用原文全文参考章节", "Full-Source Reference Chapters", 2))
	if len(fullSource) == 0 {
		sb.WriteString(reportEmptyLine(lang, "无章节启用原文全文参考。", "No chapters used full-source reference."))
	} else {
		for _, ch := range fullSource {
			line := fmt.Sprintf("%s %d %s", reportText(lang, "第", "Ch."), ch.Num, strings.TrimSpace(ch.Title))
			writeReportBullet(&sb, lang, line, line, defaultString(ch.FullTextReason, reportText(lang, "未填写理由", "No reason recorded")))
		}
	}

	sb.WriteString(reportHeading(lang, "高风险章节", "High-Risk Chapters", 2))
	writeSimilarityChapterList(&sb, lang, plan, highRisk)

	sb.WriteString(reportHeading(lang, "中风险章节", "Medium-Risk Chapters", 2))
	writeSimilarityChapterList(&sb, lang, plan, mediumRisk)

	sb.WriteString(reportHeading(lang, "全部检查指标", "All Check Metrics", 2))
	for _, ch := range plan.Chapters {
		check := latestRewriteCheckForChapter(plan, ch.Num)
		if check == nil {
			continue
		}
		result := check.Closeness.Deterministic
		title := fmt.Sprintf("%s %d %s", reportText(lang, "第", "Ch."), ch.Num, strings.TrimSpace(ch.Title))
		sb.WriteString(reportHeading(lang, title, title, 3))
		writeReportBullet(&sb, lang, "风险等级", "Risk level", reportRiskLabel(result.RiskLevel, lang))
		writeReportBullet(&sb, lang, "AI 相似度检查", "AI closeness check", rewriteAICheckSummary(RewriteAICheckResult{
			Result: check.Closeness.Result,
			Issues: check.Closeness.Issues,
			Notes:  check.Closeness.Notes,
		}, lang))
		writeReportBullet(&sb, lang, "8 字符片段重合", "8-char n-gram overlap", fmt.Sprintf("%s (%d/%d)", reportPercent(result.CharNGramOverlapRatio), result.CharNGramOverlapCount, result.ComparedCharNGramCount))
		writeReportBullet(&sb, lang, "句子重合", "Sentence overlap", fmt.Sprintf("%s (%d/%d)", reportPercent(result.SentenceOverlapRatio), result.MatchedSentenceCount, result.ComparedSentenceCount))
		writeReportBullet(&sb, lang, "最长连续公共片段", "Longest common fragment", fmt.Sprintf("%d", result.LongestCommonRunes))
		writeReportFragments(&sb, lang, result.HighRiskFragments)
		sb.WriteString("\n")
	}

	return sb.String()
}

func buildRewriteSettingsReport(lang string, settings *ProjectSettings, analysis *ReferenceAnalysis, plan *RewritePlan, generatedAt string) string {
	var sb strings.Builder
	writeReportTitle(&sb, lang, "角色/设定变化一致性报告", "Character and Settings Change Consistency Report", generatedAt)

	sb.WriteString(reportHeading(lang, "当前新稿设定库", "Current New-Manuscript Settings", 2))
	if settings == nil {
		sb.WriteString(reportEmptyLine(lang, "暂无结构化设定。", "No structured settings recorded."))
	} else {
		writeReportBullet(&sb, lang, "角色", "Characters", fmt.Sprintf("%d", len(settings.Characters)))
		writeReportBullet(&sb, lang, "世界观", "Worldview entries", fmt.Sprintf("%d", len(settings.Worldview)))
		writeReportBullet(&sb, lang, "组织", "Organizations", fmt.Sprintf("%d", len(settings.Organizations)))
		writeReportBullet(&sb, lang, "关系", "Relations", fmt.Sprintf("%d", len(settings.Relations)))
	}

	if analysis != nil {
		sb.WriteString(reportHeading(lang, "原文设定导入状态", "Source Settings Import Status", 2))
		writeReportBullet(&sb, lang, "导入状态", "Import status", defaultString(analysis.SettingsImportStatus, ReferenceSettingsStatusNone))
		writeReportBullet(&sb, lang, "原文候选角色", "Source candidate characters", fmt.Sprintf("%d", len(analysis.Settings.Characters)))
		writeReportBullet(&sb, lang, "原文候选世界观", "Source candidate worldview", fmt.Sprintf("%d", len(analysis.Settings.Worldview)))
		writeReportBullet(&sb, lang, "原文候选组织", "Source candidate organizations", fmt.Sprintf("%d", len(analysis.Settings.Organizations)))
		writeReportBullet(&sb, lang, "原文候选关系", "Source candidate relations", fmt.Sprintf("%d", len(analysis.Settings.Relations)))
	}

	sb.WriteString(reportHeading(lang, "角色变化", "Character Changes", 2))
	writeRewriteChangeItems(&sb, lang, plan.CharacterChanges)
	sb.WriteString(reportHeading(lang, "设定变化", "Setting Changes", 2))
	writeRewriteChangeItems(&sb, lang, plan.SettingChanges)
	sb.WriteString(reportHeading(lang, "关系变化", "Relationship Changes", 2))
	writeRewriteChangeItems(&sb, lang, plan.RelationshipChanges)

	sb.WriteString(reportHeading(lang, "全篇约束", "Global Constraints", 2))
	writeReportList(&sb, lang, "约束", "Constraints", plan.Constraints)

	return sb.String()
}

func writeReportTitle(sb *strings.Builder, lang, zh, en, generatedAt string) {
	sb.WriteString("# ")
	sb.WriteString(reportText(lang, zh, en))
	sb.WriteString("\n\n")
	writeReportBullet(sb, lang, "生成时间", "Generated at", generatedAt)
	sb.WriteString("\n")
}

func reportHeading(lang, zh, en string, level int) string {
	if level < 1 {
		level = 1
	}
	return strings.Repeat("#", level) + " " + reportText(lang, zh, en) + "\n\n"
}

func writeReportBullet(sb *strings.Builder, lang, zhLabel, enLabel, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = reportText(lang, "未记录", "Not recorded")
	}
	sb.WriteString("- **")
	sb.WriteString(reportText(lang, zhLabel, enLabel))
	sb.WriteString("**: ")
	sb.WriteString(value)
	sb.WriteString("\n")
}

func writeReportList(sb *strings.Builder, lang, zhLabel, enLabel string, values []string) {
	sb.WriteString("- **")
	sb.WriteString(reportText(lang, zhLabel, enLabel))
	sb.WriteString("**:\n")
	if len(values) == 0 {
		sb.WriteString("  - ")
		sb.WriteString(reportText(lang, "未记录", "Not recorded"))
		sb.WriteString("\n")
		return
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		sb.WriteString("  - ")
		sb.WriteString(value)
		sb.WriteString("\n")
	}
}

func reportText(lang, zh, en string) string {
	if NormalizeLanguage(lang) == LangEN {
		return en
	}
	return zh
}

func reportEmptyLine(lang, zh, en string) string {
	return reportText(lang, zh, en) + "\n\n"
}

func acceptedChapterCount(state *Progress) (int, int) {
	if state == nil {
		return 0, 0
	}
	accepted := 0
	for _, ch := range state.Chapters {
		if ch.Status == StatusAccepted && strings.TrimSpace(ch.Content) != "" {
			accepted++
		}
	}
	return accepted, len(state.Chapters)
}

func rewriteCheckCount(plan *RewritePlan) (int, int) {
	if plan == nil {
		return 0, 0
	}
	checked := 0
	passed := 0
	for _, ch := range plan.Chapters {
		check := latestRewriteCheckForChapter(plan, ch.Num)
		if check == nil {
			continue
		}
		checked++
		if check.Passed {
			passed++
		}
	}
	return checked, passed
}

func rewriteImpactMap(plan *RewritePlan) map[string]*RewriteRequestImpact {
	out := make(map[string]*RewriteRequestImpact)
	if plan == nil {
		return out
	}
	for i := range plan.RequestImpacts {
		impact := &plan.RequestImpacts[i]
		out[impact.RequestID] = impact
	}
	return out
}

func rewriteRequestAffectedChapters(plan *RewritePlan, state *Progress, req RewriteRequest, impacts map[string]*RewriteRequestImpact) []int {
	if impact := impacts[req.ID]; impact != nil && len(impact.AffectedChapters) > 0 {
		return sortedUniqueInts(impact.AffectedChapters)
	}
	return affectedChaptersForRewriteRequest(plan, state, req)
}

func rewriteRequestImplementationStatus(plan *RewritePlan, state *Progress, req RewriteRequest, impacts map[string]*RewriteRequestImpact, lang string) string {
	chapters := rewriteRequestAffectedChapters(plan, state, req, impacts)
	if len(chapters) == 0 {
		return reportText(lang, "未映射到具体章节，需人工复核", "No mapped chapters; manual review needed")
	}
	accepted := 0
	passed := 0
	checked := 0
	for _, num := range chapters {
		if ch := chapterStateByNum(state, num); ch != nil && ch.Status == StatusAccepted && strings.TrimSpace(ch.Content) != "" {
			accepted++
		}
		if check := latestRewriteCheckForChapter(plan, num); check != nil {
			checked++
			if check.Passed {
				passed++
			}
		}
	}
	if accepted == len(chapters) && passed == len(chapters) {
		return reportText(lang, "已落实，相关章节已确认且检查通过", "Implemented: related chapters are accepted and passed checks")
	}
	if accepted == len(chapters) && checked == len(chapters) {
		return reportText(lang, "已写完，但存在待复核检查项", "Written, but some checks need review")
	}
	if accepted > 0 {
		return reportText(lang, "部分落实，仍有章节未完成或未检查", "Partially implemented; some chapters remain unfinished or unchecked")
	}
	return reportText(lang, "待落实，相关章节尚未确认", "Pending implementation; related chapters are not accepted yet")
}

func rewriteRequestScopeText(req RewriteRequest, lang string) string {
	switch req.Type {
	case RewriteRequestTypeChapter:
		return fmt.Sprintf("%s %d", reportText(lang, "第", "Ch."), req.ChapterNum)
	case RewriteRequestTypeRange:
		return fmt.Sprintf("%s %d-%d", reportText(lang, "第", "Ch."), req.ChapterStart, req.ChapterEnd)
	case RewriteRequestTypeCharacter, RewriteRequestTypeSetting, RewriteRequestTypeRelationship:
		return defaultString(req.Object, req.Scope)
	default:
		return defaultString(req.Scope, reportText(lang, "全局", "Global"))
	}
}

func rewriteRequestTypeLabel(t, lang string) string {
	labelsZH := map[string]string{
		RewriteRequestTypeGlobal:       "全局",
		RewriteRequestTypeChapter:      "单章",
		RewriteRequestTypeRange:        "章节范围",
		RewriteRequestTypeCharacter:    "角色",
		RewriteRequestTypeSetting:      "设定",
		RewriteRequestTypeRelationship: "关系",
		RewriteRequestTypeEnding:       "结局",
		RewriteRequestTypeForbidden:    "禁止项",
	}
	labelsEN := map[string]string{
		RewriteRequestTypeGlobal:       "Global",
		RewriteRequestTypeChapter:      "Chapter",
		RewriteRequestTypeRange:        "Chapter range",
		RewriteRequestTypeCharacter:    "Character",
		RewriteRequestTypeSetting:      "Setting",
		RewriteRequestTypeRelationship: "Relationship",
		RewriteRequestTypeEnding:       "Ending",
		RewriteRequestTypeForbidden:    "Forbidden",
	}
	if NormalizeLanguage(lang) == LangEN {
		return defaultString(labelsEN[t], t)
	}
	return defaultString(labelsZH[t], t)
}

func rewriteMappingTypeLabel(t, lang string) string {
	switch t {
	case ChapterMappingMerge:
		return reportText(lang, "合并", "Merge")
	case ChapterMappingSplit:
		return reportText(lang, "拆分", "Split")
	default:
		return reportText(lang, "一对一", "One-to-one")
	}
}

func referenceAnalysisSummaryForReport(lang string, analysis *ReferenceAnalysis, nums []int) string {
	if analysis == nil || len(nums) == 0 {
		return ""
	}
	var parts []string
	for _, num := range nums {
		for _, ch := range analysis.Chapters {
			if ch.Num != num {
				continue
			}
			bits := []string{}
			if ch.Summary != "" {
				bits = append(bits, ch.Summary)
			}
			if ch.SceneFunction != "" {
				bits = append(bits, ch.SceneFunction)
			}
			if len(ch.KeyEvents) > 0 {
				bits = append(bits, strings.Join(ch.KeyEvents, "；"))
			}
			if len(bits) > 0 {
				parts = append(parts, fmt.Sprintf("%s %d: %s", reportText(lang, "原文第", "Source ch."), num, strings.Join(bits, " / ")))
			}
		}
	}
	return strings.Join(parts, "\n")
}

func latestRewriteCheckForChapter(plan *RewritePlan, chapterNum int) *RewriteCheckResult {
	if plan == nil {
		return nil
	}
	if _, ch := FindRewriteChapterPlan(plan, chapterNum); ch != nil && ch.LastCheckResult != nil {
		return ch.LastCheckResult
	}
	for i := len(plan.CheckResults) - 1; i >= 0; i-- {
		if plan.CheckResults[i].ChapterNum == chapterNum {
			return &plan.CheckResults[i]
		}
	}
	return nil
}

func chapterStateByNum(state *Progress, num int) *ChapterState {
	if state == nil {
		return nil
	}
	for i := range state.Chapters {
		if state.Chapters[i].Num == num {
			return &state.Chapters[i]
		}
	}
	return nil
}

func rewriteAICheckSummary(check RewriteAICheckResult, lang string) string {
	result := strings.TrimSpace(check.Result)
	if result == "" {
		result = reportText(lang, "未记录", "Not recorded")
	}
	var parts []string
	parts = append(parts, result)
	if len(check.Issues) > 0 {
		parts = append(parts, reportText(lang, "问题：", "Issues: ")+strings.Join(check.Issues, "；"))
	}
	if strings.TrimSpace(check.Notes) != "" {
		parts = append(parts, reportText(lang, "备注：", "Notes: ")+strings.TrimSpace(check.Notes))
	}
	return strings.Join(parts, " | ")
}

func writeSimilarityChapterList(sb *strings.Builder, lang string, plan *RewritePlan, chapters []RewriteChapterPlan) {
	if len(chapters) == 0 {
		sb.WriteString(reportEmptyLine(lang, "暂无。", "None."))
		return
	}
	for _, ch := range chapters {
		check := latestRewriteCheckForChapter(plan, ch.Num)
		if check == nil {
			continue
		}
		result := check.Closeness.Deterministic
		title := fmt.Sprintf("%s %d %s", reportText(lang, "第", "Ch."), ch.Num, strings.TrimSpace(ch.Title))
		sb.WriteString(reportHeading(lang, title, title, 3))
		writeReportBullet(sb, lang, "风险等级", "Risk level", reportRiskLabel(result.RiskLevel, lang))
		writeReportBullet(sb, lang, "8 字符片段重合", "8-char n-gram overlap", reportPercent(result.CharNGramOverlapRatio))
		writeReportBullet(sb, lang, "句子重合", "Sentence overlap", reportPercent(result.SentenceOverlapRatio))
		writeReportBullet(sb, lang, "最长连续公共片段", "Longest common fragment", fmt.Sprintf("%d", result.LongestCommonRunes))
		writeReportFragments(sb, lang, result.HighRiskFragments)
		sb.WriteString("\n")
	}
}

func writeReportFragments(sb *strings.Builder, lang string, fragments []SimilarityFragment) {
	if len(fragments) == 0 {
		return
	}
	sb.WriteString("- **")
	sb.WriteString(reportText(lang, "高风险片段", "High-risk fragments"))
	sb.WriteString("**:\n")
	for _, fragment := range fragments {
		source := strings.TrimSpace(fragment.Source)
		if source == "" {
			continue
		}
		sb.WriteString("  - ")
		sb.WriteString(fmt.Sprintf("[%s, %d] %s", fragment.Reason, fragment.Runes, source))
		sb.WriteString("\n")
	}
}

func writeRewriteChangeItems(sb *strings.Builder, lang string, items []RewriteChangeItem) {
	if len(items) == 0 {
		sb.WriteString(reportEmptyLine(lang, "暂无记录。", "No records."))
		return
	}
	for _, item := range items {
		title := strings.TrimSpace(item.Object)
		if title == "" {
			title = reportText(lang, "未命名对象", "Unnamed object")
		}
		sb.WriteString(reportHeading(lang, title, title, 3))
		writeReportBullet(sb, lang, "原设定", "Before", item.Before)
		writeReportBullet(sb, lang, "新设定", "After", item.After)
		writeReportBullet(sb, lang, "影响章节", "Affected chapters", reportChapterList(item.AffectedChaps))
		sb.WriteString("\n")
	}
}

func reportChapterList(values []int) string {
	values = sortedUniqueInts(values)
	if len(values) == 0 {
		return "-"
	}
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(parts, ", ")
}

func sortedUniqueInts(values []int) []int {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[int]bool)
	var out []int
	for _, v := range values {
		if v <= 0 || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Ints(out)
	return out
}

func reportRiskLabel(level, lang string) string {
	switch strings.ToLower(level) {
	case "high":
		return reportText(lang, "高", "High")
	case "medium":
		return reportText(lang, "中", "Medium")
	case "low":
		return reportText(lang, "低", "Low")
	default:
		return defaultString(level, reportText(lang, "未记录", "Not recorded"))
	}
}

func reportPercent(value float64) string {
	return fmt.Sprintf("%.1f%%", value*100)
}
