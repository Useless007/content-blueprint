export namespace domain {

	export class ProjectSummary {
	    id: string;
	    name: string;
	    keyword: string;
	    score: number;
	    updatedAt: string;

	    static createFrom(source: any = {}) {
	        return new ProjectSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.keyword = source["keyword"];
	        this.score = source["score"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class ProviderSettings {
	    provider: string;
	    model: string;
	    useGrounding: boolean;
	    baseUrl: string;

	    static createFrom(source: any = {}) {
	        return new ProviderSettings(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.model = source["model"];
	        this.useGrounding = source["useGrounding"];
	        this.baseUrl = source["baseUrl"];
	    }
	}
	export class BootstrapData {
	    settings: ProviderSettings;
	    projects: ProjectSummary[];
	    apiKeyFromEnvironment: boolean;

	    static createFrom(source: any = {}) {
	        return new BootstrapData(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.settings = this.convertValues(source["settings"], ProviderSettings);
	        this.projects = this.convertValues(source["projects"], ProjectSummary);
	        this.apiKeyFromEnvironment = source["apiKeyFromEnvironment"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class EvidenceSource {
	    id: string;
	    title: string;
	    url: string;
	    notes: string;

	    static createFrom(source: any = {}) {
	        return new EvidenceSource(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.url = source["url"];
	        this.notes = source["notes"];
	    }
	}
	export class ContentBrief {
	    keyword: string;
	    audience: string;
	    intent: string;
	    objective: string;
	    brandVoice: string;
	    language: string;
	    additionalInstructions: string;
	    evidence: EvidenceSource[];

	    static createFrom(source: any = {}) {
	        return new ContentBrief(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.keyword = source["keyword"];
	        this.audience = source["audience"];
	        this.intent = source["intent"];
	        this.objective = source["objective"];
	        this.brandVoice = source["brandVoice"];
	        this.language = source["language"];
	        this.additionalInstructions = source["additionalInstructions"];
	        this.evidence = this.convertValues(source["evidence"], EvidenceSource);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class FAQItem {
	    question: string;
	    answer: string;
	    sourceIds: string[];

	    static createFrom(source: any = {}) {
	        return new FAQItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.question = source["question"];
	        this.answer = source["answer"];
	        this.sourceIds = source["sourceIds"];
	    }
	}
	export class KeyTakeaway {
	    statement: string;
	    sourceIds: string[];

	    static createFrom(source: any = {}) {
	        return new KeyTakeaway(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.statement = source["statement"];
	        this.sourceIds = source["sourceIds"];
	    }
	}
	export class GeneratedContent {
	    title: string;
	    slug: string;
	    metaTitle: string;
	    metaDescription: string;
	    summaryBox: string;
	    mainContentHtml: string;
	    keyTakeaways: KeyTakeaway[];
	    faqData: FAQItem[];

	    static createFrom(source: any = {}) {
	        return new GeneratedContent(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.slug = source["slug"];
	        this.metaTitle = source["metaTitle"];
	        this.metaDescription = source["metaDescription"];
	        this.summaryBox = source["summaryBox"];
	        this.mainContentHtml = source["mainContentHtml"];
	        this.keyTakeaways = this.convertValues(source["keyTakeaways"], KeyTakeaway);
	        this.faqData = this.convertValues(source["faqData"], FAQItem);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GenerationRequest {
	    brief: ContentBrief;
	    settings: ProviderSettings;
	    apiKey?: string;

	    static createFrom(source: any = {}) {
	        return new GenerationRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.brief = this.convertValues(source["brief"], ContentBrief);
	        this.settings = this.convertValues(source["settings"], ProviderSettings);
	        this.apiKey = source["apiKey"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Usage {
	    inputTokens: number;
	    outputTokens: number;
	    thoughtTokens: number;
	    totalTokens: number;

	    static createFrom(source: any = {}) {
	        return new Usage(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.inputTokens = source["inputTokens"];
	        this.outputTokens = source["outputTokens"];
	        this.thoughtTokens = source["thoughtTokens"];
	        this.totalTokens = source["totalTokens"];
	    }
	}
	export class PromptPreview {
	    system: string;
	    user: string;
	    schemaJson: string;

	    static createFrom(source: any = {}) {
	        return new PromptPreview(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.system = source["system"];
	        this.user = source["user"];
	        this.schemaJson = source["schemaJson"];
	    }
	}
	export class QualityCheck {
	    id: string;
	    label: string;
	    status: string;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new QualityCheck(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.status = source["status"];
	        this.message = source["message"];
	    }
	}
	export class QualityReport {
	    score: number;
	    checks: QualityCheck[];
	    wordCount: number;
	    sourceCoverage: number;

	    static createFrom(source: any = {}) {
	        return new QualityReport(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.score = source["score"];
	        this.checks = this.convertValues(source["checks"], QualityCheck);
	        this.wordCount = source["wordCount"];
	        this.sourceCoverage = source["sourceCoverage"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GroundingSource {
	    title: string;
	    url: string;

	    static createFrom(source: any = {}) {
	        return new GroundingSource(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.url = source["url"];
	    }
	}
	export class GenerationResult {
	    content: GeneratedContent;
	    groundingSources: GroundingSource[];
	    quality: QualityReport;
	    prompt: PromptPreview;
	    usage: Usage;
	    model: string;

	    static createFrom(source: any = {}) {
	        return new GenerationResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.content = this.convertValues(source["content"], GeneratedContent);
	        this.groundingSources = this.convertValues(source["groundingSources"], GroundingSource);
	        this.quality = this.convertValues(source["quality"], QualityReport);
	        this.prompt = this.convertValues(source["prompt"], PromptPreview);
	        this.usage = this.convertValues(source["usage"], Usage);
	        this.model = source["model"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}


	export class Project {
	    id: string;
	    name: string;
	    brief: ContentBrief;
	    content?: GeneratedContent;
	    groundingSources: GroundingSource[];
	    quality?: QualityReport;
	    settings: ProviderSettings;
	    createdAt: string;
	    updatedAt: string;

	    static createFrom(source: any = {}) {
	        return new Project(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.brief = this.convertValues(source["brief"], ContentBrief);
	        this.content = this.convertValues(source["content"], GeneratedContent);
	        this.groundingSources = this.convertValues(source["groundingSources"], GroundingSource);
	        this.quality = this.convertValues(source["quality"], QualityReport);
	        this.settings = this.convertValues(source["settings"], ProviderSettings);
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}






}

export namespace facebookcompanion {

	export class EvidenceSource {
	    id: string;
	    title: string;
	    url?: string;
	    notes?: string;

	    static createFrom(source: any = {}) {
	        return new EvidenceSource(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.url = source["url"];
	        this.notes = source["notes"];
	    }
	}
	export class Brief {
	    topic: string;
	    audience: string;
	    objective: string;
	    offer?: string;
	    brandVoice?: string;
	    language: string;
	    productDetails?: string;
	    evidence?: EvidenceSource[];
	    additionalInstructions?: string;

	    static createFrom(source: any = {}) {
	        return new Brief(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.topic = source["topic"];
	        this.audience = source["audience"];
	        this.objective = source["objective"];
	        this.offer = source["offer"];
	        this.brandVoice = source["brandVoice"];
	        this.language = source["language"];
	        this.productDetails = source["productDetails"];
	        this.evidence = this.convertValues(source["evidence"], EvidenceSource);
	        this.additionalInstructions = source["additionalInstructions"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CarouselSlide {
	    headline: string;
	    body: string;

	    static createFrom(source: any = {}) {
	        return new CarouselSlide(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.headline = source["headline"];
	        this.body = source["body"];
	    }
	}
	export class Reply {
	    intent: string;
	    reply: string;

	    static createFrom(source: any = {}) {
	        return new Reply(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.intent = source["intent"];
	        this.reply = source["reply"];
	    }
	}
	export class ContentPack {
	    hooks: string[];
	    longPost: string;
	    shortPost: string;
	    reelScript: string;
	    carouselSlides: CarouselSlide[];
	    cta: string;
	    firstComment: string;
	    replyBank: Reply[];
	    complianceNotes: string[];

	    static createFrom(source: any = {}) {
	        return new ContentPack(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hooks = source["hooks"];
	        this.longPost = source["longPost"];
	        this.shortPost = source["shortPost"];
	        this.reelScript = source["reelScript"];
	        this.carouselSlides = this.convertValues(source["carouselSlides"], CarouselSlide);
	        this.cta = source["cta"];
	        this.firstComment = source["firstComment"];
	        this.replyBank = this.convertValues(source["replyBank"], Reply);
	        this.complianceNotes = source["complianceNotes"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class GroundingSource {
	    title?: string;
	    url: string;

	    static createFrom(source: any = {}) {
	        return new GroundingSource(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.url = source["url"];
	    }
	}
	export class PackSnapshot {
	    version: number;
	    briefRevision: string;
	    pack: ContentPack;
	    groundingSources?: GroundingSource[];
	    generatedBy?: string;
	    // Go type: time
	    updatedAt: any;

	    static createFrom(source: any = {}) {
	        return new PackSnapshot(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.briefRevision = source["briefRevision"];
	        this.pack = this.convertValues(source["pack"], ContentPack);
	        this.groundingSources = this.convertValues(source["groundingSources"], GroundingSource);
	        this.generatedBy = source["generatedBy"];
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace main {

	export class FacebookProviderState {
	    id: string;
	    label: string;
	    available: boolean;
	    authenticationChecked: boolean;
	    authenticated: boolean;
	    version?: string;
	    message?: string;

	    static createFrom(source: any = {}) {
	        return new FacebookProviderState(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.available = source["available"];
	        this.authenticationChecked = source["authenticationChecked"];
	        this.authenticated = source["authenticated"];
	        this.version = source["version"];
	        this.message = source["message"];
	    }
	}
	export class FacebookBootstrapData {
	    providers: FacebookProviderState[];
	    latest?: facebookcompanion.PackSnapshot;

	    static createFrom(source: any = {}) {
	        return new FacebookBootstrapData(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.providers = this.convertValues(source["providers"], FacebookProviderState);
	        this.latest = this.convertValues(source["latest"], facebookcompanion.PackSnapshot);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FacebookBriefReceipt {
	    briefRevision: string;
	    updatedAt: string;

	    static createFrom(source: any = {}) {
	        return new FacebookBriefReceipt(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.briefRevision = source["briefRevision"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class FacebookGenerationRequest {
	    runId: string;
	    provider: string;
	    workflow: string;
	    model?: string;
	    brief: facebookcompanion.Brief;
	    timeoutSec?: number;

	    static createFrom(source: any = {}) {
	        return new FacebookGenerationRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.runId = source["runId"];
	        this.provider = source["provider"];
	        this.workflow = source["workflow"];
	        this.model = source["model"];
	        this.brief = this.convertValues(source["brief"], facebookcompanion.Brief);
	        this.timeoutSec = source["timeoutSec"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FacebookLatestResult {
	    found: boolean;
	    stale: boolean;
	    snapshot?: facebookcompanion.PackSnapshot;

	    static createFrom(source: any = {}) {
	        return new FacebookLatestResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.found = source["found"];
	        this.stale = source["stale"];
	        this.snapshot = this.convertValues(source["snapshot"], facebookcompanion.PackSnapshot);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class GrowthBootstrapData {
	    catalog: workbench.Playbook[];
	    providers: FacebookProviderState[];
	    latest?: workbench.PackSnapshot;
	    latestStale: boolean;
	    leads: salesops.Lead[];
	    experiments: measurement.ExperimentView[];

	    static createFrom(source: any = {}) {
	        return new GrowthBootstrapData(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.catalog = this.convertValues(source["catalog"], workbench.Playbook);
	        this.providers = this.convertValues(source["providers"], FacebookProviderState);
	        this.latest = this.convertValues(source["latest"], workbench.PackSnapshot);
	        this.latestStale = source["latestStale"];
	        this.leads = this.convertValues(source["leads"], salesops.Lead);
	        this.experiments = this.convertValues(source["experiments"], measurement.ExperimentView);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GrowthBriefReceipt {
	    briefRevision: string;
	    updatedAt: string;

	    static createFrom(source: any = {}) {
	        return new GrowthBriefReceipt(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.briefRevision = source["briefRevision"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class GrowthGenerationRequest {
	    runId: string;
	    provider: string;
	    workflow: string;
	    model?: string;
	    brief: workbench.GrowthBrief;
	    timeoutSec?: number;

	    static createFrom(source: any = {}) {
	        return new GrowthGenerationRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.runId = source["runId"];
	        this.provider = source["provider"];
	        this.workflow = source["workflow"];
	        this.model = source["model"];
	        this.brief = this.convertValues(source["brief"], workbench.GrowthBrief);
	        this.timeoutSec = source["timeoutSec"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GrowthLatestResult {
	    found: boolean;
	    stale: boolean;
	    snapshot?: workbench.PackSnapshot;

	    static createFrom(source: any = {}) {
	        return new GrowthLatestResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.found = source["found"];
	        this.stale = source["stale"];
	        this.snapshot = this.convertValues(source["snapshot"], workbench.PackSnapshot);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace measurement {

	export class DerivedRates {
	    ctr: number;
	    leadRate: number;
	    closeRate: number;

	    static createFrom(source: any = {}) {
	        return new DerivedRates(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ctr = source["ctr"];
	        this.leadRate = source["leadRate"];
	        this.closeRate = source["closeRate"];
	    }
	}
	export class VariantMetrics {
	    impressions: number;
	    clicks: number;
	    leads: number;
	    sales: number;
	    revenueSatang: number;

	    static createFrom(source: any = {}) {
	        return new VariantMetrics(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.impressions = source["impressions"];
	        this.clicks = source["clicks"];
	        this.leads = source["leads"];
	        this.sales = source["sales"];
	        this.revenueSatang = source["revenueSatang"];
	    }
	}
	export class Experiment {
	    id: string;
	    title: string;
	    hypothesis: string;
	    variable: string;
	    variantA: string;
	    variantB: string;
	    startDate?: string;
	    endDate?: string;
	    audience?: string;
	    channel?: string;
	    primaryMetric: string;
	    guardrail?: string;
	    comparable: boolean;
	    metricsA: VariantMetrics;
	    metricsB: VariantMetrics;
	    learning?: string;
	    decision?: string;
	    approvedBy?: string;
	    createdAt: string;
	    updatedAt: string;

	    static createFrom(source: any = {}) {
	        return new Experiment(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.hypothesis = source["hypothesis"];
	        this.variable = source["variable"];
	        this.variantA = source["variantA"];
	        this.variantB = source["variantB"];
	        this.startDate = source["startDate"];
	        this.endDate = source["endDate"];
	        this.audience = source["audience"];
	        this.channel = source["channel"];
	        this.primaryMetric = source["primaryMetric"];
	        this.guardrail = source["guardrail"];
	        this.comparable = source["comparable"];
	        this.metricsA = this.convertValues(source["metricsA"], VariantMetrics);
	        this.metricsB = this.convertValues(source["metricsB"], VariantMetrics);
	        this.learning = source["learning"];
	        this.decision = source["decision"];
	        this.approvedBy = source["approvedBy"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ExperimentView {
	    experiment: Experiment;
	    ratesA: DerivedRates;
	    ratesB: DerivedRates;
	    analysisLabel: string;
	    winner: string;

	    static createFrom(source: any = {}) {
	        return new ExperimentView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.experiment = this.convertValues(source["experiment"], Experiment);
	        this.ratesA = this.convertValues(source["ratesA"], DerivedRates);
	        this.ratesB = this.convertValues(source["ratesB"], DerivedRates);
	        this.analysisLabel = source["analysisLabel"];
	        this.winner = source["winner"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UTMRequest {
	    destinationUrl: string;
	    source: string;
	    medium: string;
	    campaign: string;
	    content?: string;
	    term?: string;

	    static createFrom(source: any = {}) {
	        return new UTMRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.destinationUrl = source["destinationUrl"];
	        this.source = source["source"];
	        this.medium = source["medium"];
	        this.campaign = source["campaign"];
	        this.content = source["content"];
	        this.term = source["term"];
	    }
	}
	export class UTMResult {
	    url: string;
	    campaignId: string;

	    static createFrom(source: any = {}) {
	        return new UTMResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.campaignId = source["campaignId"];
	    }
	}

}

export namespace salesops {

	export class Lead {
	    id: string;
	    label: string;
	    source?: string;
	    offer?: string;
	    stage: string;
	    needs?: string;
	    objections?: string;
	    assignee?: string;
	    nextFollowUp?: string;
	    handoffNote?: string;
	    campaignId?: string;
	    utm?: string;
	    saleAmountSatang: number;
	    commissionRateBps: number;
	    estimatedCommissionSatang: number;
	    commissionConfirmed: boolean;
	    confirmedCommissionSatang: number;
	    confirmedBy?: string;
	    confirmedAt?: string;
	    createdAt: string;
	    updatedAt: string;

	    static createFrom(source: any = {}) {
	        return new Lead(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.source = source["source"];
	        this.offer = source["offer"];
	        this.stage = source["stage"];
	        this.needs = source["needs"];
	        this.objections = source["objections"];
	        this.assignee = source["assignee"];
	        this.nextFollowUp = source["nextFollowUp"];
	        this.handoffNote = source["handoffNote"];
	        this.campaignId = source["campaignId"];
	        this.utm = source["utm"];
	        this.saleAmountSatang = source["saleAmountSatang"];
	        this.commissionRateBps = source["commissionRateBps"];
	        this.estimatedCommissionSatang = source["estimatedCommissionSatang"];
	        this.commissionConfirmed = source["commissionConfirmed"];
	        this.confirmedCommissionSatang = source["confirmedCommissionSatang"];
	        this.confirmedBy = source["confirmedBy"];
	        this.confirmedAt = source["confirmedAt"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}

}

export namespace updater {

	export class Info {
	    currentVersion: string;
	    latestVersion: string;
	    state: string;
	    releaseUrl?: string;
	    publishedAt?: string;
	    releaseNotes?: string;

	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.state = source["state"];
	        this.releaseUrl = source["releaseUrl"];
	        this.publishedAt = source["publishedAt"];
	        this.releaseNotes = source["releaseNotes"];
	    }
	}

}

export namespace workbench {

	export class BlockItem {
	    label: string;
	    value: string;
	    note?: string;

	    static createFrom(source: any = {}) {
	        return new BlockItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.value = source["value"];
	        this.note = source["note"];
	    }
	}
	export class FieldSpec {
	    key: string;
	    label: string;
	    help: string;
	    placeholder: string;
	    inputType: string;
	    required: boolean;
	    maxChars: number;
	    sensitive: boolean;

	    static createFrom(source: any = {}) {
	        return new FieldSpec(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.label = source["label"];
	        this.help = source["help"];
	        this.placeholder = source["placeholder"];
	        this.inputType = source["inputType"];
	        this.required = source["required"];
	        this.maxChars = source["maxChars"];
	        this.sensitive = source["sensitive"];
	    }
	}
	export class GrowthBlock {
	    id: string;
	    title: string;
	    purpose: string;
	    kind: string;
	    body: string;
	    items: BlockItem[];
	    columns: string[];
	    rows: string[][];
	    code: string;
	    evidenceBasis: string;
	    sourceIds: string[];

	    static createFrom(source: any = {}) {
	        return new GrowthBlock(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.purpose = source["purpose"];
	        this.kind = source["kind"];
	        this.body = source["body"];
	        this.items = this.convertValues(source["items"], BlockItem);
	        this.columns = source["columns"];
	        this.rows = source["rows"];
	        this.code = source["code"];
	        this.evidenceBasis = source["evidenceBasis"];
	        this.sourceIds = source["sourceIds"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GrowthBrief {
	    playbookId: string;
	    language: string;
	    brandVoice?: string;
	    inputs: Record<string, string>;
	    evidence: domain.EvidenceSource[];

	    static createFrom(source: any = {}) {
	        return new GrowthBrief(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.playbookId = source["playbookId"];
	        this.language = source["language"];
	        this.brandVoice = source["brandVoice"];
	        this.inputs = source["inputs"];
	        this.evidence = this.convertValues(source["evidence"], domain.EvidenceSource);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ReviewCheck {
	    status: string;
	    label: string;
	    reason: string;

	    static createFrom(source: any = {}) {
	        return new ReviewCheck(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.label = source["label"];
	        this.reason = source["reason"];
	    }
	}
	export class GrowthPack {
	    title: string;
	    summary: string;
	    blocks: GrowthBlock[];
	    openQuestions: string[];
	    riskFlags: string[];
	    reviewChecks: ReviewCheck[];

	    static createFrom(source: any = {}) {
	        return new GrowthPack(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.summary = source["summary"];
	        this.blocks = this.convertValues(source["blocks"], GrowthBlock);
	        this.openQuestions = source["openQuestions"];
	        this.riskFlags = source["riskFlags"];
	        this.reviewChecks = this.convertValues(source["reviewChecks"], ReviewCheck);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PackSnapshot {
	    version: number;
	    briefRevision: string;
	    playbookId: string;
	    evidenceSourceIds: string[];
	    pack: GrowthPack;
	    generatedBy: string;
	    // Go type: time
	    updatedAt: any;
	    reviewStatus: string;
	    reviewerNote?: string;
	    // Go type: time
	    reviewUpdatedAt?: any;

	    static createFrom(source: any = {}) {
	        return new PackSnapshot(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.briefRevision = source["briefRevision"];
	        this.playbookId = source["playbookId"];
	        this.evidenceSourceIds = source["evidenceSourceIds"];
	        this.pack = this.convertValues(source["pack"], GrowthPack);
	        this.generatedBy = source["generatedBy"];
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.reviewStatus = source["reviewStatus"];
	        this.reviewerNote = source["reviewerNote"];
	        this.reviewUpdatedAt = this.convertValues(source["reviewUpdatedAt"], null);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Playbook {
	    id: string;
	    category: string;
	    title: string;
	    summary: string;
	    outcome: string;
	    fields: FieldSpec[];

	    static createFrom(source: any = {}) {
	        return new Playbook(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.category = source["category"];
	        this.title = source["title"];
	        this.summary = source["summary"];
	        this.outcome = source["outcome"];
	        this.fields = this.convertValues(source["fields"], FieldSpec);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}
