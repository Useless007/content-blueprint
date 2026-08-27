package cliprovider

// contentPackSchema is intentionally strict at every object boundary. Provider
// structured-output validation is useful, but callers still perform their own
// strict JSON decoding and domain validation after the process exits.
const contentPackSchema = `{
  "type":"object",
  "additionalProperties":false,
  "properties":{
    "hooks":{"type":"array","minItems":3,"maxItems":3,"items":{"type":"string","minLength":1,"maxLength":1000}},
    "longPost":{"type":"string","minLength":1,"maxLength":30000},
    "shortPost":{"type":"string","minLength":1,"maxLength":5000},
    "reelScript":{"type":"string","minLength":1,"maxLength":15000},
    "carouselSlides":{"type":"array","minItems":3,"maxItems":10,"items":{"type":"object","additionalProperties":false,"properties":{"headline":{"type":"string","minLength":1,"maxLength":500},"body":{"type":"string","minLength":1,"maxLength":2000}},"required":["headline","body"]}},
    "cta":{"type":"string","minLength":1,"maxLength":3000},
    "firstComment":{"type":"string","minLength":1,"maxLength":5000},
    "replyBank":{"type":"array","minItems":3,"maxItems":12,"items":{"type":"object","additionalProperties":false,"properties":{"intent":{"type":"string","minLength":1,"maxLength":500},"reply":{"type":"string","minLength":1,"maxLength":4000}},"required":["intent","reply"]}},
    "complianceNotes":{"type":"array","maxItems":12,"items":{"type":"string","minLength":1,"maxLength":2000}}
  },
  "required":["hooks","longPost","shortPost","reelScript","carouselSlides","cta","firstComment","replyBank","complianceNotes"]
}`

func ContentPackSchema() string { return contentPackSchema }

const teamStrategySchema = `{
  "type":"object",
  "additionalProperties":false,
  "properties":{
    "audienceInsight":{"type":"string","minLength":1,"maxLength":2000},
    "positioning":{"type":"string","minLength":1,"maxLength":2000},
    "primaryPromise":{"type":"string","minLength":1,"maxLength":2000},
    "angles":{"type":"array","minItems":3,"maxItems":5,"items":{"type":"object","additionalProperties":false,"properties":{"name":{"type":"string","minLength":1,"maxLength":300},"hookApproach":{"type":"string","minLength":1,"maxLength":1000},"rationale":{"type":"string","minLength":1,"maxLength":1500}},"required":["name","hookApproach","rationale"]}},
    "narrativeFlow":{"type":"array","minItems":3,"maxItems":10,"items":{"type":"string","minLength":1,"maxLength":1000}},
    "evidenceUse":{"type":"array","maxItems":30,"items":{"type":"object","additionalProperties":false,"properties":{"sourceId":{"type":"string","minLength":1,"maxLength":128},"allowedClaim":{"type":"string","minLength":1,"maxLength":2000}},"required":["sourceId","allowedClaim"]}},
    "complianceRisks":{"type":"array","maxItems":12,"items":{"type":"string","minLength":1,"maxLength":1500}}
  },
  "required":["audienceInsight","positioning","primaryPromise","angles","narrativeFlow","evidenceUse","complianceRisks"]
}`

func TeamStrategySchema() string { return teamStrategySchema }
