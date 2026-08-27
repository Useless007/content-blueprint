package quality

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"ContentBlueprint/internal/domain"
	"ContentBlueprint/internal/htmlutil"
)

const (
	StatusPass    = "pass"
	StatusWarning = "warning"
	StatusFail    = "fail"
)

var (
	wordPattern       = regexp.MustCompile(`[\p{L}\p{N}]+`)
	h2Pattern         = regexp.MustCompile(`(?i)<h2(?:\s|>)`)
	h3Pattern         = regexp.MustCompile(`(?i)<h3(?:\s|>)`)
	h1Pattern         = regexp.MustCompile(`(?i)<h1(?:\s|>)`)
	unsafeHTMLPattern = regexp.MustCompile(`(?is)<\s*(script|style|iframe|object|embed|form|svg|math)\b|\son[a-z]+\s*=|javascript\s*:`)
)

type weightedCheck struct {
	check  domain.QualityCheck
	weight int
}

func Evaluate(brief domain.ContentBrief, content domain.GeneratedContent) domain.QualityReport {
	plainContent := htmlutil.Text(content.MainContentHTML)
	wordCount := CountWords(plainContent)
	bodySourceIDs := htmlutil.SourceIDs(content.MainContentHTML)
	sourceCoverage, unknownSources := sourceCoverage(brief, content, bodySourceIDs)

	weighted := []weightedCheck{
		{completenessCheck(content), 18},
		{metadataCheck(content), 14},
		{keywordCheck(brief.Keyword, content, plainContent), 10},
		{structureCheck(content.MainContentHTML), 12},
		{depthCheck(wordCount), 10},
		{evidenceCheck(len(brief.Evidence), sourceCoverage, unknownSources), 18},
		{bodyCitationCheck(brief.Evidence, bodySourceIDs), 14},
		{takeawayCheck(content.KeyTakeaways), 8},
		{faqCheck(content.FAQData), 8},
		{safetyCheck(content.MainContentHTML), 12},
	}

	report := domain.QualityReport{
		Checks:         make([]domain.QualityCheck, 0, len(weighted)),
		WordCount:      wordCount,
		SourceCoverage: sourceCoverage,
	}
	var earned, total float64
	for _, item := range weighted {
		report.Checks = append(report.Checks, item.check)
		total += float64(item.weight)
		switch item.check.Status {
		case StatusPass:
			earned += float64(item.weight)
		case StatusWarning:
			earned += float64(item.weight) * 0.55
		}
	}
	if total > 0 {
		report.Score = int(math.Round(earned * 100 / total))
	}
	return report
}

func CountWords(value string) int {
	words := wordPattern.FindAllString(value, -1)
	count := 0
	for _, word := range words {
		thaiRunes := 0
		nonThai := false
		for _, character := range word {
			if unicode.In(character, unicode.Thai) {
				thaiRunes++
			} else {
				nonThai = true
			}
		}
		if thaiRunes > 0 {
			// Thai normally omits spaces. Four runes per lexical unit is a
			// deliberately conservative estimate for an editorial signal.
			count += (thaiRunes + 3) / 4
		}
		if nonThai {
			count++
		}
	}
	return count
}

func completenessCheck(content domain.GeneratedContent) domain.QualityCheck {
	missing := make([]string, 0, 4)
	if strings.TrimSpace(content.Title) == "" {
		missing = append(missing, "title")
	}
	if strings.TrimSpace(content.SummaryBox) == "" {
		missing = append(missing, "summaryBox")
	}
	if strings.TrimSpace(content.MainContentHTML) == "" {
		missing = append(missing, "mainContentHtml")
	}
	if strings.TrimSpace(content.Slug) == "" {
		missing = append(missing, "slug")
	}
	if len(missing) > 0 {
		return check("completeness", "ความครบถ้วน", StatusFail, "ยังขาดฟิลด์สำคัญ: "+strings.Join(missing, ", "))
	}
	return check("completeness", "ความครบถ้วน", StatusPass, "มีส่วนประกอบหลักของบทความครบ")
}

func metadataCheck(content domain.GeneratedContent) domain.QualityCheck {
	titleLength := utf8.RuneCountInString(strings.TrimSpace(content.MetaTitle))
	descriptionLength := utf8.RuneCountInString(strings.TrimSpace(content.MetaDescription))
	if titleLength == 0 || descriptionLength == 0 {
		return check("metadata", "ข้อมูลสำหรับผลการค้นหา", StatusFail, "ต้องมี meta title และ meta description")
	}
	if titleLength < 30 || titleLength > 60 || descriptionLength < 110 || descriptionLength > 160 {
		return check("metadata", "ข้อมูลสำหรับผลการค้นหา", StatusWarning,
			fmt.Sprintf("meta title %d ตัวอักษร และ meta description %d ตัวอักษร; ควรตรวจความกระชับในหน้าผลการค้นหา", titleLength, descriptionLength))
	}
	return check("metadata", "ข้อมูลสำหรับผลการค้นหา", StatusPass,
		fmt.Sprintf("ความยาว meta title (%d) และ description (%d) อยู่ในช่วงตรวจทานที่เหมาะสม", titleLength, descriptionLength))
}

func keywordCheck(keyword string, content domain.GeneratedContent, plainContent string) domain.QualityCheck {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return check("keyword_alignment", "ความสอดคล้องกับคีย์เวิร์ด", StatusFail, "brief ไม่มีคีย์เวิร์ด")
	}
	titleHasKeyword := strings.Contains(strings.ToLower(content.Title), keyword) || strings.Contains(strings.ToLower(content.MetaTitle), keyword)
	bodyHasKeyword := strings.Contains(strings.ToLower(plainContent), keyword)
	switch {
	case titleHasKeyword && bodyHasKeyword:
		return check("keyword_alignment", "ความสอดคล้องกับคีย์เวิร์ด", StatusPass, "คีย์เวิร์ดปรากฏอย่างชัดเจนในชื่อและเนื้อหา")
	case titleHasKeyword || bodyHasKeyword:
		return check("keyword_alignment", "ความสอดคล้องกับคีย์เวิร์ด", StatusWarning, "คีย์เวิร์ดปรากฏเพียงบางส่วน ควรตรวจว่าบทความตอบ intent ชัดเจน")
	default:
		return check("keyword_alignment", "ความสอดคล้องกับคีย์เวิร์ด", StatusFail, "ไม่พบคีย์เวิร์ดหลักในชื่อหรือเนื้อหา")
	}
}

func structureCheck(articleHTML string) domain.QualityCheck {
	if strings.TrimSpace(articleHTML) == "" {
		return check("structure", "โครงสร้างบทความ", StatusFail, "ยังไม่มีเนื้อหาหลัก")
	}
	if h1Pattern.MatchString(articleHTML) {
		return check("structure", "โครงสร้างบทความ", StatusWarning, "main content มี H1 ซ้ำกับชื่อบทความ ควรเริ่มโครงสร้างที่ H2")
	}
	if !h2Pattern.MatchString(articleHTML) {
		return check("structure", "โครงสร้างบทความ", StatusWarning, "ควรแบ่งหัวข้อหลักด้วย H2 ที่สื่อความหมาย")
	}
	if h3Pattern.MatchString(articleHTML) {
		return check("structure", "โครงสร้างบทความ", StatusPass, "มีลำดับหัวข้อ H2/H3 ที่ช่วยให้สแกนเนื้อหาได้")
	}
	return check("structure", "โครงสร้างบทความ", StatusPass, "มีหัวข้อ H2 แบ่งเนื้อหาอย่างชัดเจน")
}

func depthCheck(wordCount int) domain.QualityCheck {
	switch {
	case wordCount == 0:
		return check("editorial_depth", "ความลึกของเนื้อหา", StatusFail, "ยังไม่มีข้อความที่อ่านได้ในเนื้อหาหลัก")
	case wordCount < 150:
		return check("editorial_depth", "ความลึกของเนื้อหา", StatusFail, fmt.Sprintf("มีเนื้อหาประมาณ %d คำ อาจยังตอบคำถามหลักได้ไม่ครบ", wordCount))
	case wordCount < 350:
		return check("editorial_depth", "ความลึกของเนื้อหา", StatusWarning, fmt.Sprintf("มีเนื้อหาประมาณ %d คำ ควรตรวจความครบถ้วนตาม intent", wordCount))
	default:
		return check("editorial_depth", "ความลึกของเนื้อหา", StatusPass, fmt.Sprintf("มีเนื้อหาประมาณ %d คำ; ให้ยึดความครบถ้วนมากกว่าจำนวนคำ", wordCount))
	}
}

func evidenceCheck(sourceCount, coverage int, unknown []string) domain.QualityCheck {
	if len(unknown) > 0 {
		return check("source_coverage", "การรองรับด้วยแหล่งข้อมูล", StatusFail,
			"พบ sourceIds ที่ไม่มีใน evidence: "+strings.Join(unknown, ", "))
	}
	if sourceCount == 0 {
		return check("source_coverage", "การรองรับด้วยแหล่งข้อมูล", StatusWarning, "brief ยังไม่มี evidence จึงตรวจการรองรับข้อเท็จจริงไม่ได้")
	}
	switch {
	case coverage == 0:
		return check("source_coverage", "การรองรับด้วยแหล่งข้อมูล", StatusFail, "ยังไม่มี takeaway หรือ FAQ ที่อ้างถึง evidence")
	case coverage < 60:
		return check("source_coverage", "การรองรับด้วยแหล่งข้อมูล", StatusWarning, fmt.Sprintf("ใช้ evidence %d%% ควรตรวจว่าแหล่งที่เหลือจำเป็นหรือถูกมองข้าม", coverage))
	default:
		return check("source_coverage", "การรองรับด้วยแหล่งข้อมูล", StatusPass, fmt.Sprintf("ใช้ evidence %d%% โดย source ID ที่อ้างถึงมีอยู่จริง", coverage))
	}
}

func bodyCitationCheck(evidence []domain.EvidenceSource, bodySourceIDs []string) domain.QualityCheck {
	available := make(map[string]struct{}, len(evidence))
	for _, source := range evidence {
		if id := strings.TrimSpace(source.ID); id != "" {
			available[id] = struct{}{}
		}
	}
	if len(available) == 0 {
		if len(bodySourceIDs) > 0 {
			return check("body_citations", "การอ้างอิงในเนื้อหา", StatusFail, "พบ citation marker แต่ brief ไม่มี evidence ที่ตรงกัน")
		}
		return check("body_citations", "การอ้างอิงในเนื้อหา", StatusPass, "ไม่มี evidence ที่ต้องใส่ citation marker ในเนื้อหา")
	}
	if len(bodySourceIDs) == 0 {
		return check("body_citations", "การอ้างอิงในเนื้อหา", StatusFail,
			`มี evidence แต่ไม่พบ marker รูปแบบ <sup data-source-id="S1">[S1]</sup> ใน main content`)
	}
	unknown := make([]string, 0)
	for _, id := range bodySourceIDs {
		if _, exists := available[id]; !exists {
			unknown = append(unknown, id)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return check("body_citations", "การอ้างอิงในเนื้อหา", StatusFail, "citation marker ใช้ ID ที่ไม่มีใน evidence: "+strings.Join(unknown, ", "))
	}
	return check("body_citations", "การอ้างอิงในเนื้อหา", StatusPass,
		fmt.Sprintf("พบ citation marker ที่ตรงกับ evidence %d แหล่งใน main content", len(bodySourceIDs)))
}

func takeawayCheck(items []domain.KeyTakeaway) domain.QualityCheck {
	if len(items) == 0 {
		return check("takeaways", "ประเด็นสำคัญ", StatusFail, "ยังไม่มี key takeaways")
	}
	for _, item := range items {
		if strings.TrimSpace(item.Statement) == "" {
			return check("takeaways", "ประเด็นสำคัญ", StatusFail, "มี takeaway ที่ไม่มีข้อความ")
		}
	}
	if len(items) < 3 {
		return check("takeaways", "ประเด็นสำคัญ", StatusWarning, fmt.Sprintf("มี takeaway %d ข้อ ควรตรวจว่าสรุปสาระหลักครบหรือไม่", len(items)))
	}
	return check("takeaways", "ประเด็นสำคัญ", StatusPass, fmt.Sprintf("มี takeaway ที่อ่านแยกได้ %d ข้อ", len(items)))
}

func faqCheck(items []domain.FAQItem) domain.QualityCheck {
	if len(items) == 0 {
		return check("faq", "คำถามที่พบบ่อย", StatusFail, "ยังไม่มี FAQ")
	}
	for _, item := range items {
		if strings.TrimSpace(item.Question) == "" || strings.TrimSpace(item.Answer) == "" {
			return check("faq", "คำถามที่พบบ่อย", StatusFail, "มี FAQ ที่คำถามหรือคำตอบว่าง")
		}
	}
	if len(items) < 3 {
		return check("faq", "คำถามที่พบบ่อย", StatusWarning, fmt.Sprintf("มี FAQ %d ข้อ ควรเพิ่มเฉพาะคำถามที่ผู้อ่านต้องการจริง", len(items)))
	}
	return check("faq", "คำถามที่พบบ่อย", StatusPass, fmt.Sprintf("มี FAQ ที่ตอบได้โดยลำพัง %d ข้อ", len(items)))
}

func safetyCheck(articleHTML string) domain.QualityCheck {
	if unsafeHTMLPattern.MatchString(articleHTML) {
		return check("safe_html", "ความปลอดภัยของ HTML", StatusFail, "พบแท็กหรือ attribute ที่ไม่ปลอดภัย; exporter จะตัดออก แต่ควรตรวจต้นฉบับ")
	}
	return check("safe_html", "ความปลอดภัยของ HTML", StatusPass, "ไม่พบ active content หรือ inline event handler")
}

func sourceCoverage(brief domain.ContentBrief, content domain.GeneratedContent, bodySourceIDs []string) (int, []string) {
	available := make(map[string]struct{}, len(brief.Evidence))
	for _, source := range brief.Evidence {
		if id := strings.TrimSpace(source.ID); id != "" {
			available[id] = struct{}{}
		}
	}
	used := make(map[string]struct{})
	unknown := make(map[string]struct{})
	visit := func(ids []string) {
		for _, rawID := range ids {
			id := strings.TrimSpace(rawID)
			if id == "" {
				continue
			}
			if _, exists := available[id]; exists {
				used[id] = struct{}{}
			} else {
				unknown[id] = struct{}{}
			}
		}
	}
	visit(bodySourceIDs)
	for _, item := range content.KeyTakeaways {
		visit(item.SourceIDs)
	}
	for _, item := range content.FAQData {
		visit(item.SourceIDs)
	}
	unknownList := make([]string, 0, len(unknown))
	for id := range unknown {
		unknownList = append(unknownList, id)
	}
	sort.Strings(unknownList)
	if len(available) == 0 {
		return 0, unknownList
	}
	return int(math.Round(float64(len(used)) * 100 / float64(len(available)))), unknownList
}

func check(id, label, status, message string) domain.QualityCheck {
	return domain.QualityCheck{ID: id, Label: label, Status: status, Message: message}
}
