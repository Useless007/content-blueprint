package workbench

import "strings"

type FieldSpec struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Help        string `json:"help"`
	Placeholder string `json:"placeholder"`
	InputType   string `json:"inputType"`
	Required    bool   `json:"required"`
	MaxChars    int    `json:"maxChars"`
	Sensitive   bool   `json:"sensitive"`
}

type Playbook struct {
	ID       string      `json:"id"`
	Category string      `json:"category"`
	Title    string      `json:"title"`
	Summary  string      `json:"summary"`
	Outcome  string      `json:"outcome"`
	Fields   []FieldSpec `json:"fields"`
}

type playbookDefinition struct {
	playbook     Playbook
	instructions string
}

func field(key, label, help, placeholder, inputType string, required bool, max int, sensitive bool) FieldSpec {
	return FieldSpec{Key: key, Label: label, Help: help, Placeholder: placeholder, InputType: inputType, Required: required, MaxChars: max, Sensitive: sensitive}
}

var catalog = []playbookDefinition{
	{Playbook{ID: "offer-audience", Category: "sales", Title: "Offer & Audience", Summary: "Clarify an evidence-backed offer and audience.", Outcome: "Value proposition, message hierarchy, angle matrix, and approval checks.", Fields: []FieldSpec{
		field("offer", "Offer", "Verified product, service, price, and conditions.", "Describe only verified offer facts", "textarea", true, 12000, false),
		field("audience", "Audience", "Specific intended customer group.", "Who is this for?", "textarea", true, 4000, false),
		field("problems", "Problems", "Observed customer problems without invented claims.", "Known problems and questions", "textarea", true, 8000, false),
		field("proof", "Proof", "Available proof and its limitations.", "Evidence, credentials, or leave blank", "textarea", false, 8000, false),
		field("constraints", "Claims guardrails", "Claims, disclosures, or terms that must be respected.", "What must not be claimed?", "textarea", false, 6000, false),
	}}, "Build a truthful offer and audience system. Separate verified facts, user assumptions, and AI inference. Never invent outcomes, urgency, testimonials, price, credentials, or proof."},
	{Playbook{ID: "facebook-campaign", Category: "facebook", Title: "Facebook Campaign", Summary: "Plan a human-reviewed Facebook content campaign.", Outcome: "Campaign spine, content sequence, creative briefs, CTA, and pre-publish checks.", Fields: []FieldSpec{
		field("objective", "Objective", "One measurable communication objective.", "What should the audience do?", "text", true, 2000, false),
		field("audience", "Audience", "The intended audience for this campaign.", "Specific audience", "textarea", true, 4000, false),
		field("offer", "Offer", "Verified offer facts and conditions.", "Offer, price, dates, conditions", "textarea", true, 10000, false),
		field("channels", "Formats", "Requested Facebook formats.", "Posts, Reel, Story, carousel", "text", false, 1000, false),
		field("cta", "CTA", "The human-approved next step.", "Request details, visit page, register", "text", true, 1500, false),
		field("competitorAdsNotes", "Competitor ad observations", "Optional notes the user personally observed or copied from Meta Ads Library; not scraped data.", "Observed messages, formats, and creative patterns", "textarea", false, 10000, false),
		field("ownedAudienceAssets", "Owned audience assets", "Only lawful first-party data or assets the page is authorized to use.", "Permitted engagement data, website audience, or consented lead list", "textarea", false, 10000, true),
	}}, "Create a bounded Facebook campaign for human review. Find a creative gap from user-supplied observations and include an Audience Source Plan limited to lawfully controlled first-party assets. Do not scrape, identify people, build lists of another page's followers, commenters, inbox users, or customers, recommend uploading data without rights and lawful basis, post, schedule, message, or infer platform metrics. Include fact, link, date, disclosure, UTM, and owner-review checks."},
	{Playbook{ID: "sales-reply", Category: "sales", Title: "Sales Reply Copilot", Summary: "Draft replies from text deliberately pasted by the user.", Outcome: "Intent summary, reply options, missing facts, qualification questions, and owner handoff.", Fields: []FieldSpec{
		field("message", "Selected message", "Paste only the minimum message needed; redact personal data first.", "Redacted customer message", "textarea", true, 12000, true),
		field("offerFacts", "Approved offer facts", "Facts the reply may use.", "Price, terms, FAQ, availability", "textarea", true, 10000, false),
		field("intentHint", "Intent hint", "Optional user interpretation.", "Question, objection, ready to buy", "text", false, 1000, false),
		field("handoffRules", "Handoff rules", "When the reply must be escalated to the owner.", "Escalation conditions", "textarea", false, 4000, false),
	}}, "Draft options only from pasted text and approved facts. Do not identify people, send messages, make high-impact decisions, or retain unnecessary personal data. Flag missing facts and owner handoff."},
	{Playbook{ID: "seo-topic-map", Category: "seo", Title: "SEO Topic Map", Summary: "Map supplied topics and questions to people-first pages.", Outcome: "Topic clusters, inferred intent labels, page proposals, evidence gaps, and cannibalization warnings.", Fields: []FieldSpec{
		field("seedTopics", "Seed topics", "Topics supplied by the user.", "Topics, questions, expertise", "textarea", true, 12000, false),
		field("audience", "Audience", "Who the site serves.", "Audience and journey", "textarea", true, 4000, false),
		field("existingPages", "Existing pages", "Known page titles or URLs; no crawling occurs.", "One page or URL per line", "textarea", false, 12000, false),
		field("businessGoals", "Business goals", "Desired business relevance and conversion path.", "Goals and approved CTAs", "textarea", true, 4000, false),
	}}, "Create a people-first topic map from supplied inputs. Label search intent as AI inference. Never fabricate search volume, competition, rankings, or Google data."},
	{Playbook{ID: "seo-content-brief", Category: "seo", Title: "SEO Content Brief", Summary: "Create an evidence-aware brief for one useful page.", Outcome: "Audience need, intent hypothesis, outline, evidence slots, title/meta drafts, and review checklist.", Fields: []FieldSpec{
		field("topic", "Topic", "The page topic or question.", "One focused topic", "text", true, 1000, false),
		field("audience", "Audience", "The intended reader.", "Who needs this page?", "textarea", true, 4000, false),
		field("pageGoal", "Page goal", "The useful outcome and approved conversion path.", "What should this page help accomplish?", "textarea", true, 3000, false),
		field("knownFacts", "Known facts", "Verified facts available to the writer.", "Facts and evidence gaps", "textarea", false, 12000, false),
	}}, "Create a people-first content brief, not bulk generated copy. Mark intent as inference, require evidence for factual claims, and never promise ranking or indexing."},
	{Playbook{ID: "seo-onpage-review", Category: "seo", Title: "SEO On-page Review", Summary: "Review user-supplied page text without crawling or rank checking.", Outcome: "Prioritized clarity, title/meta, structure, helpfulness, and factual-risk recommendations.", Fields: []FieldSpec{
		field("pageUrl", "Page URL", "Optional exact page URL supplied by the user.", "https://example.com/page", "url", false, 4096, false),
		field("pageText", "Page text", "Text deliberately pasted by the user.", "Paste title, headings, and body", "textarea", true, 50000, false),
		field("pageGoal", "Page goal", "Intended audience outcome.", "What should the page do?", "textarea", true, 3000, false),
		field("targetTopic", "Target topic", "The user-declared topic; not rank data.", "Primary topic", "text", true, 1000, false),
	}}, "Review only supplied text and declared URL. Do not crawl, fetch, rank-check, or claim access to Google. Prioritize helpfulness and factual accuracy over keyword repetition."},
	{Playbook{ID: "seo-internal-links", Category: "seo", Title: "SEO Internal Links", Summary: "Propose internal-link tasks from a supplied page inventory.", Outcome: "Source-target suggestions, contextual anchors, rationale, and verification tasks.", Fields: []FieldSpec{
		field("pages", "Page inventory", "User-supplied titles, URLs, and short summaries.", "One page per line", "textarea", true, 30000, false),
		field("focusPage", "Focus page", "Optional page to prioritize.", "Title or URL", "text", false, 4096, false),
		field("constraints", "Constraints", "Pages or anchors to avoid.", "Editorial constraints", "textarea", false, 4000, false),
	}}, "Suggest internal links only from the supplied inventory. Do not claim pages were crawled. Avoid forced exact-match anchors and require human context verification."},
	{Playbook{ID: "seo-structured-data", Category: "seo", Title: "SEO Structured Data", Summary: "Draft schema from facts visible in supplied page content.", Outcome: "Candidate JSON-LD, field-to-page mapping, missing required facts, and validation checklist.", Fields: []FieldSpec{
		field("pageType", "Page type", "The user-declared page type.", "Article, Product, Event, FAQ", "text", true, 500, false),
		field("visibleContent", "Visible page content", "Facts visibly present on the page.", "Paste relevant visible content", "textarea", true, 30000, false),
		field("canonicalUrl", "Canonical URL", "Absolute URL for the page.", "https://example.com/page", "url", true, 4096, false),
		field("knownIdentifiers", "Known identifiers", "Optional verified identifiers.", "SKU, author URL, organization URL", "textarea", false, 4000, false),
	}}, "Draft only structured data supported by visible supplied content. Never invent reviews, ratings, prices, availability, identifiers, or eligibility; never promise rich results."},
	{Playbook{ID: "seo-search-console-opportunities", Category: "seo", Title: "Search Console Opportunities", Summary: "Interpret metrics explicitly imported by the user.", Outcome: "Observed patterns, bounded opportunity hypotheses, refresh experiments, and measurement cautions.", Fields: []FieldSpec{
		field("importedMetrics", "Imported Search Console metrics", "User-imported query/page/date/click/impression/CTR/position data.", "Paste a bounded CSV excerpt or summary", "textarea", true, 50000, true),
		field("dateRange", "Date range", "The exact imported reporting period.", "YYYY-MM-DD to YYYY-MM-DD", "text", true, 100, false),
		field("property", "Property", "User-declared property label; omit credentials.", "Domain or URL-prefix label", "text", true, 1000, true),
		field("goal", "Goal", "What opportunity should be investigated.", "Refresh, CTR, content gap", "textarea", true, 3000, false),
	}}, "Treat every metric as imported user data, preserve its date range and property context, and separate observation from hypothesis. Do not claim causal lift, live API access, or current rankings."},
	{Playbook{ID: "cross-channel-repurpose", Category: "cross-channel", Title: "Cross-channel Repurpose", Summary: "Adapt one approved source asset across channels.", Outcome: "Channel-specific derivatives, traceability, UTM/content IDs, and human review checks.", Fields: []FieldSpec{
		field("sourceAsset", "Approved source asset", "The user-supplied source content.", "Paste approved source content", "textarea", true, 40000, false),
		field("sourceFacts", "Source facts", "Verified facts that must remain unchanged.", "Claims, dates, prices, disclosures", "textarea", true, 10000, false),
		field("targetChannels", "Target channels", "Channels and formats to adapt for.", "Facebook, website, email", "text", true, 2000, false),
		field("cta", "Approved CTA", "The approved next step.", "One CTA", "text", true, 1500, false),
	}}, "Repurpose only the approved source asset and facts. Adapt format without inventing claims. Preserve traceability, disclosures, and human approval for every external action."},
}

func Catalog() []Playbook {
	result := make([]Playbook, len(catalog))
	for index, definition := range catalog {
		result[index] = definition.playbook
		result[index].Fields = append([]FieldSpec(nil), definition.playbook.Fields...)
	}
	return result
}

func LookupPlaybook(id string) (Playbook, bool) {
	id = strings.TrimSpace(id)
	for _, definition := range catalog {
		if definition.playbook.ID == id {
			playbook := definition.playbook
			playbook.Fields = append([]FieldSpec(nil), playbook.Fields...)
			return playbook, true
		}
	}
	return Playbook{}, false
}

// TrustedInstructions is intentionally separate from the UI-safe catalog DTO.
func TrustedInstructions(id string) (string, bool) {
	id = strings.TrimSpace(id)
	for _, definition := range catalog {
		if definition.playbook.ID == id {
			return definition.instructions, true
		}
	}
	return "", false
}
