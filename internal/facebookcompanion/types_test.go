package facebookcompanion

import (
	"strings"
	"testing"
)

func validBrief() Brief {
	return Brief{
		Topic:          "เวิร์กช็อปวางแผนการตลาด",
		Audience:       "เจ้าของธุรกิจขนาดเล็ก",
		Objective:      "ให้ผู้อ่านทักมาขอรายละเอียด",
		Offer:          "เวิร์กช็อปหนึ่งวัน ราคา 2,900 บาท",
		BrandVoice:     "อาจารย์ที่เป็นกันเองและตรงไปตรงมา",
		Language:       "ไทย",
		ProductDetails: "เรียนสดและมีแบบฝึกหัด",
		Evidence: []EvidenceSource{
			{ID: "course-page", Title: "หน้าหลักสูตร", URL: "https://example.com/course", Notes: "ระบุราคา 2,900 บาท"},
		},
	}
}

func validPack() ContentPack {
	return ContentPack{
		Hooks:      []string{"การตลาดไม่ควรเริ่มจากการโพสต์", "วางแผนหนึ่งวันให้ชัดทั้งเดือน", "ติดตรงไหนเวลาคิดคอนเทนต์?"},
		LongPost:   "โพสต์ฉบับยาวที่ให้รายละเอียดครบถ้วน",
		ShortPost:  "โพสต์สั้นที่อ่านจบได้ไว",
		ReelScript: "เปิด: วางแผนก่อนโพสต์\nเนื้อหา: สามขั้นตอน\nปิด: ทักมาขอรายละเอียด",
		CarouselSlides: []CarouselSlide{
			{Headline: "เริ่มจากเป้าหมาย", Body: "กำหนดสิ่งที่อยากให้ผู้อ่านทำ"},
			{Headline: "รู้จักคนอ่าน", Body: "เลือกปัญหาที่เขากำลังเจอ"},
			{Headline: "วางสารหลัก", Body: "ทำให้แต่ละโพสต์มีหน้าที่เดียว"},
		},
		CTA:          "ทักข้อความเพื่อขอรายละเอียดหลักสูตร",
		FirstComment: "รายละเอียดที่ควรเตรียมก่อนเข้าเวิร์กช็อป",
		ReplyBank: []Reply{
			{Intent: "ถามราคา", Reply: "ราคา 2,900 บาทตามข้อมูลหลักสูตรครับ"},
			{Intent: "ถามว่าเหมาะกับใคร", Reply: "ออกแบบมาสำหรับเจ้าของธุรกิจขนาดเล็กครับ"},
			{Intent: "ยังไม่แน่ใจ", Reply: "เล่าปัญหาที่กำลังเจอมาได้ครับ จะช่วยดูว่าเนื้อหาตรงหรือไม่"},
		},
		ComplianceNotes: []string{},
	}
}

func TestBriefRevisionIsStableAfterWhitespaceNormalization(t *testing.T) {
	first := validBrief()
	second := validBrief()
	second.Topic = "  " + second.Topic + "\n"

	firstRevision, err := BriefRevision(first)
	if err != nil {
		t.Fatal(err)
	}
	secondRevision, err := BriefRevision(second)
	if err != nil {
		t.Fatal(err)
	}
	if firstRevision != secondRevision {
		t.Fatalf("normalized revisions differ: %s != %s", firstRevision, secondRevision)
	}
}

func TestValidateBriefRejectsUnsafeEvidenceURL(t *testing.T) {
	brief := validBrief()
	brief.Evidence[0].URL = "file:///C:/secret.txt"
	if err := ValidateBrief(brief); err == nil || !strings.Contains(err.Error(), "url is invalid") {
		t.Fatalf("expected invalid URL error, got %v", err)
	}
}

func TestValidateContentPackRejectsDuplicateHooks(t *testing.T) {
	pack := validPack()
	pack.Hooks[2] = strings.ToUpper(pack.Hooks[0])
	if err := ValidateContentPack(pack); err == nil || !strings.Contains(err.Error(), "distinct") {
		t.Fatalf("expected distinct hook error, got %v", err)
	}
}

func TestNormalizeGroundingSourcesDeduplicatesCanonicalURL(t *testing.T) {
	sources, err := NormalizeGroundingSources([]GroundingSource{
		{URL: "https://EXAMPLE.com:443/article", Title: ""},
		{URL: "https://example.com/article", Title: "บทความ"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 1 || sources[0].Title != "บทความ" || sources[0].URL != "https://example.com/article" {
		t.Fatalf("unexpected normalized sources: %#v", sources)
	}
}
