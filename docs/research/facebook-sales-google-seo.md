# Research: Facebook Sales Assistant + Google SEO Copilot

วันที่ตรวจสอบแหล่งข้อมูล: 2026-08-28

## ขอบเขตและข้อสรุป

เอกสารนี้คัดเฉพาะ workflow ที่เหมาะกับเครื่องมือใช้เองของแอดมิน Facebook Page และผู้ช่วย Google SEO โดยยึดเอกสารทางการของ Meta และ Google เท่านั้น แนวทางที่ปลอดภัยและให้คุณค่าเร็วที่สุดคือ **AI ช่วยคิด ร่าง ตรวจ และบันทึกงาน แต่ผู้ใช้เป็นคนเลือกบริบท ตรวจคำตอบ และกดส่ง/เผยแพร่เอง**

เหตุผลสำคัญคือ Meta นิยาม scraping ว่าเป็นการเก็บข้อมูลจากเว็บไซต์หรือส่วนติดต่อที่สร้างไว้ให้มนุษย์ใช้ด้วยระบบอัตโนมัติ และระบุว่าการ scraping ที่ไม่ได้รับอนุญาตอาจขัดกับข้อกำหนดของบริการ ([Facebook Help: Data scraping](https://www.facebook.com/help/463983701520800)) ขณะที่สิทธิ์ของผู้ดูแลเพจแบ่งตามงาน เช่น content, messages/community activity, ads และ insights จึงควรให้เครื่องมือทำงานไม่เกินสิทธิ์ที่เจ้าของเพจมอบให้ ([Facebook Help: Page access](https://www.facebook.com/help/289207354498410/r.php/))

ฝั่ง Google ไม่ควรทำเครื่องมือยิงคำค้นหรือ scrape หน้า Search เพื่อเช็กอันดับ เพราะ Google ระบุว่า automated queries รวมถึง scraping ผลการค้นหาเพื่อ rank checking เป็น machine-generated traffic ที่ขัดกับ spam policies ([Google Search spam policies](https://developers.google.com/search/docs/essentials/spam-policies)) ใช้ข้อมูลที่ผู้ใช้ป้อน, ไฟล์ export, URL ที่ผู้ใช้สั่งตรวจอย่างเจาะจง และ API ทางการแทน

## ฟังก์ชัน MVP: ไม่ต้องมี OAuth หรือสิทธิ์แพลตฟอร์ม

### 1. Offer & Audience Workspace

สร้างฐานข้อมูลกลางสำหรับสินค้า/บริการของอาจารย์ ได้แก่ กลุ่มเป้าหมาย ปัญหา ผลลัพธ์ที่เสนอ ราคา เงื่อนไข หลักฐาน คำถามพบบ่อย ข้อห้ามในการกล่าวอ้าง CTA และน้ำเสียง จากนั้น AI สร้าง:

- value proposition และ message hierarchy;
- angle matrix แยก pain, aspiration, proof, urgency และ education;
- ข้อเสนอหลัก/โบนัส/เงื่อนไข/คำเตือนที่ต้องแสดง;
- checklist ให้แอดมินกับเจ้าของเพจอนุมัติก่อนนำไปใช้.

ฟังก์ชันนี้เป็น local-first และไม่ต้องอ่านข้อมูล Facebook ผู้ใช้เป็นคนป้อน facts และหลักฐานเอง AI ต้องติดป้ายข้อความที่ยังไม่มีหลักฐานว่า `ต้องตรวจสอบ` ไม่แต่งยอดขาย รีวิว คุณสมบัติ หรือความเร่งด่วนปลอม

### 2. Facebook Content Campaign Builder

จาก Offer Workspace ให้สร้าง campaign brief และ Content Pack ที่ประกอบด้วย:

- เป้าหมาย แคมเปญ กลุ่มเป้าหมาย single-minded message และ CTA;
- post แบบให้ความรู้, case/proof, objection, offer และ follow-up;
- hook หลายแบบ, caption, Reel script, Story sequence และ creative brief;
- เวอร์ชันสั้น/ยาวและโทนต่างกัน โดยเก็บ revision ชัดเจน;
- pre-publish checklist: facts, ราคา, วันหมดเขต, link, UTM, disclosure และ owner approval.

Extension ควรแทรก plain text เฉพาะ editor ที่ผู้ใช้คลิกเองเหมือนขอบเขตปัจจุบัน ไม่อ่าน Feed/Inbox/Comments และไม่กด Post/Schedule ให้ ผู้ใช้ที่มี Page access อาจจัดการ content, messages, comments, ads และ insights ได้ตามระดับสิทธิ์ แต่เครื่องมือต้องไม่ถือว่าผู้ใช้มี full control โดยอัตโนมัติ ([Facebook Help: Page access](https://www.facebook.com/help/289207354498410/r.php/))

### 3. Sales Reply Copilot แบบ user-in-the-loop

ผู้ใช้ paste ข้อความหรือคอมเมนต์ที่ต้องการตอบเอง แล้วระบบสร้างคำตอบโดยไม่ scrape DOM:

- สรุป intent: สนใจ, ถามราคา, เปรียบเทียบ, กังวล, ขอหลักฐาน, พร้อมซื้อ หรือควรส่งต่อเจ้าของ;
- คำตอบ 2–3 โทน พร้อมเหตุผลและข้อมูลที่ยังขาด;
- objection handling จาก fact/FAQ ที่อนุมัติแล้ว;
- qualification questions แบบสั้น และ next-best-action;
- handoff summary สำหรับส่งให้อาจารย์หรือฝ่ายขาย;
- quick-reply library ที่ผู้ใช้แก้และ copy/insert เอง.

ควรมีปุ่ม `ล้างข้อมูลส่วนบุคคลก่อนส่งให้ AI` และเตือนผู้ใช้ไม่ให้ paste ข้อมูลอ่อนไหวเกินจำเป็น ระบบไม่ควรตอบอัตโนมัติหรือส่งข้อความจำนวนมาก Meta มี Instant Reply ใน Meta Business Suite อยู่แล้วสำหรับ greeting/การตอบครั้งแรก จึงให้โปรแกรมช่วย “ร่างข้อความ” แล้วผู้ใช้ไปตั้งค่าผ่านช่องทางทางการ ([Facebook Help: automated Messenger greeting](https://www.facebook.com/help/1698046970464236/))

### 4. Lead Qualification & Handoff Board

ทำบอร์ด local-only ที่ผู้ใช้กรอกเอง:

- lead source, offer, stage, needs, objections, consent/contact preference;
- owner/admin assignee, next follow-up date และ handoff note;
- status เช่น New → Qualified → Owner review → Won/Lost;
- attribution ระหว่าง content ID, campaign ID, UTM และยอดขายที่ผู้ใช้ยืนยัน;
- commission ledger แบบประมาณการและ `confirmed by owner` แยกกัน.

ไม่ควร import รายชื่อ/โปรไฟล์หรือเก็บ Facebook user identifiers จากหน้าเว็บอัตโนมัติ และไม่ควรให้ AI ตัดสินใจเรื่องที่มีผลสูงกับบุคคลโดยไม่มีคนตรวจ

### 5. UTM Builder และ Campaign Naming

สร้าง URL พร้อม preset เช่น `utm_source=facebook`, `utm_medium=organic_social|paid_social`, `utm_campaign`, `utm_content` และ campaign ID พร้อมตรวจ:

- lowercase/case consistency;
- ช่องว่าง อักขระพิเศษ และ duplicate naming;
- preview URL และ copy button;
- ผูก UTM กับ Content Pack และ experiment โดยไม่แก้ destination URL เอง.

Google Analytics ระบุว่า UTM ทำให้เห็นว่าแคมเปญใดพาทราฟฟิกเข้ามา และแนะนำการตั้งชื่อที่สม่ำเสมอ เพราะค่าต่างตัวพิมพ์ถือเป็นคนละค่า ([Google Analytics: URL builders and UTM best practices](https://support.google.com/analytics/answer/10917952?hl=en))

### 6. Experiment Log ที่ไม่อ้างผลเกินข้อมูล

เก็บ hypothesis, variable เดียวที่เปลี่ยน, เวอร์ชัน A/B, ช่วงเวลา, audience/channel, KPI และผลที่ผู้ใช้กรอก เช่น impressions, clicks, leads, sales, revenue:

- คำนวณ CTR, lead rate, close rate, revenue/content และ commission;
- ระบุผลเป็น `directional` เมื่อไม่ได้สุ่ม audience หรือช่วงเวลาไม่เทียบเคียง;
- ห้าม AI ประกาศ winner หากข้อมูลไม่ครบหรือทดสอบหลายตัวแปรพร้อมกัน;
- เก็บ learning และ decision พร้อมชื่อผู้อนุมัติ.

### 7. SEO Topic & Intent Map

รับ seed topics จากสินค้า คำถามลูกค้า expertise ของอาจารย์ และ keyword CSV ที่ผู้ใช้นำเข้า แล้วจัดกลุ่มเป็น:

- topic → subtopic → user question → proposed page;
- intent label เช่น learn, compare, evaluate, buy/contact โดยระบุว่าเป็น **AI inference** ไม่ใช่ข้อมูล Google;
- audience stage, business relevance, evidence owner และ conversion path;
- cannibalization warning เมื่อหลาย page brief ตั้งใจตอบคำถามเดียวกัน.

ใน MVP ห้ามแสดง search volume, competition หรือ rank เป็นข้อมูลจริงถ้าไม่มีแหล่งข้อมูลทางการ ให้แสดงเพียง `ไม่ทราบ` หรือ `ผู้ใช้กรอก` Google แนะนำให้ใช้คำที่ผู้ใช้จะใช้ค้นหาในตำแหน่งสำคัญ เช่น title, main heading, alt text และ link text แต่ไม่ได้รับประกันการ crawl/index/ranking ([Google Search Essentials](https://developers.google.com/search/docs/essentials))

### 8. People-first SEO Content Brief

สร้าง brief จาก topic + intent + facts ที่ผู้ใช้ให้:

- reader goal และคำตอบหลัก;
- outline/H1/H2, questions to answer และ CTA ที่เหมาะกับ intent;
- evidence slots: ประสบการณ์จริง ตัวอย่าง รูป ข้อมูลอ้างอิง และผู้ตรวจข้อเท็จจริง;
- originality checklist และ “อะไรที่หน้านี้เพิ่มคุณค่าจากข้อมูลทั่วไป”;
- content completeness/reliability review ก่อน export;
- Facebook repurpose pack จากบทความ และ landing-page brief จาก Facebook campaign.

Google แนะนำ content ที่ helpful, reliable และ people-first และเตือนว่าการใช้ automation/AI เพื่อสร้างเนื้อหาจำนวนมากโดยมีเป้าหมายหลักเพื่อ manipulate ranking ผิด spam policies ([Google: helpful, reliable, people-first content](https://developers.google.com/search/docs/fundamentals/creating-helpful-content), [Google: generative AI content guidance](https://developers.google.com/search/docs/fundamentals/using-gen-ai-content)) ดังนั้นระบบควรเป็น copilot ที่มี evidence/review gate ไม่ใช่ mass-page generator

### 9. Title, Meta Description และ On-page Review

จากข้อความ/HTML ที่ผู้ใช้ paste หรือไฟล์ที่ผู้ใช้เลือก ให้ตรวจและเสนอ:

- title หลายแบบที่ unique, descriptive, concise และตรงกับเนื้อหาหน้า;
- meta description แบบสรุปจริง ไม่สร้างข้อมูลใหม่;
- H1/H2 consistency, missing sections, CTA และ readability;
- snippet preview ที่ติดป้ายว่า `ตัวอย่าง` เพราะ Google อาจสร้าง title/snippet ใหม่;
- checklist สิ่งที่ตรวจไม่ได้ เช่น index status, Core Web Vitals หรือ rendered DOM.

Google ใช้หลายแหล่งในการสร้าง title link และอาจไม่ใช้ `<title>` ที่เสนอ ส่วน snippet มักมาจากเนื้อหาและบางครั้งใช้ meta description ([Google: title links](https://developers.google.com/search/docs/appearance/title-link), [Google SEO Starter Guide](https://developers.google.com/search/docs/fundamentals/seo-starter-guide)) จึงห้ามเขียนว่า “ตั้งค่านี้แล้ว Google จะแสดงแน่นอน”

### 10. Internal Link Planner

รับ URL inventory/sitemap/CSV ที่ผู้ใช้นำเข้า พร้อม title และ summary แล้ว AI เสนอ:

- source page → target page → contextual anchor → เหตุผล;
- orphan candidates: หน้าที่ผู้ใช้ระบุว่าสำคัญแต่ไม่มี source link ในชุดข้อมูล;
- duplicate/over-optimized anchor warning;
- export เป็น task list เท่านั้น ไม่แก้เว็บไซต์โดยอัตโนมัติ.

Google ระบุว่า internal links และ anchor text ช่วยให้ทั้งคนและ Google เข้าใจและค้นพบหน้าอื่น และหน้าสำคัญควรมีลิงก์จากอย่างน้อยหนึ่งหน้า โดยลิงก์ที่ crawlable ควรเป็น `<a href>` ([Google: link best practices](https://developers.google.com/search/docs/crawling-indexing/links-crawlable))

### 11. Structured Data Assistant

ให้ผู้ใช้เลือกชนิดหน้าก่อน เช่น Article, Organization, Product หรือ LocalBusiness แล้วสร้าง JSON-LD draft จากข้อมูลที่มองเห็นได้บนหน้านั้นเท่านั้น พร้อม:

- required/recommended property checklist ตามชนิด;
- source mapping ว่าแต่ละ property มาจาก field ใด;
- เตือน property ที่ไม่มีหลักฐานหรือไม่ปรากฏต่อผู้ใช้;
- copy/export และลิงก์ไป Rich Results Test;
- สถานะ `eligible draft` ไม่ใช่ “ได้ rich result แน่นอน”.

Google แนะนำ JSON-LD ในหลายกรณี แต่ markup ต้องตรงกับเนื้อหาที่มองเห็นได้ ปฏิบัติตาม guideline เฉพาะชนิด และแม้ถูกต้องก็ไม่รับประกันว่าจะได้ rich result ([Google: structured data introduction](https://developers.google.com/search/docs/appearance/structured-data/intro-structured-data), [Google: general structured data guidelines](https://developers.google.com/search/docs/appearance/structured-data/sd-policies))

### 12. Manual Search Console CSV Opportunity Finder

ก่อนทำ OAuth ให้ผู้ใช้ export Search Console แล้วนำ CSV เข้าเอง ระบบหา:

- high-impression/low-CTR query-page pairs สำหรับตรวจ title/snippet/content match;
- rising/falling pages ระหว่างสองช่วงที่เทียบกัน;
- queries ที่หน้าเดียวกันตอบ และหลายหน้าที่แย่ง topic ใกล้กัน;
- refresh backlog พร้อม hypothesis ไม่ใช่คำรับประกัน ranking.

Search Console แนะนำให้วิเคราะห์ query/page และให้ความสำคัญกับแนวโน้ม clicks/impressions มากกว่าตำแหน่งอย่างเดียว อีกทั้งข้อมูลบาง query ถูก anonymize และตารางถูก truncate จึงต้องแสดง data-limit notice ([Search Console: common tasks](https://support.google.com/webmasters/answer/17010961?hl=en), [Search Console: dimensions and limitations](https://support.google.com/webmasters/answer/17011259?hl=en))

### 13. User-triggered PageSpeed Audit

ให้ผู้ใช้กดตรวจ URL ทีละรายการที่ตนมีสิทธิ์ตรวจ แล้วแสดง performance/accessibility/SEO diagnostics เป็นงานแก้ไขพร้อมหลักฐานดิบจาก PageSpeed Insights ไม่ crawl ทั้งเว็บไซต์และไม่ตรวจ URL เองเบื้องหลัง API ใช้งานได้โดยไม่มี key สำหรับการทดลองหรือปริมาณต่ำ แต่ Google แนะนำให้ใช้ key เมื่อต้องเรียกแบบอัตโนมัติหรือบ่อยขึ้น ([PageSpeed Insights API: get started](https://developers.google.com/speed/docs/insights/v5/get-started))

ผลควรติดป้าย `measurement from PageSpeed` แยกจากคำอธิบายของ AI และไม่สรุปว่าคะแนนเดียวเป็นตัวแทน ranking หรือ conversion ทั้งหมด

## ระยะถัดไป: ต้องใช้ OAuth/API หรือการอนุมัติจากแพลตฟอร์ม

### A. Search Console read-only connector — แนะนำเป็น API แรก

ใช้ OAuth 2.0 scope `webmasters.readonly` เพื่อดึง query/page/country/device พร้อม clicks, impressions, CTR และ average position แล้วสร้าง opportunity queue, trend comparison และ content refresh brief ตัว API จำกัดผลลัพธ์ไว้ที่ top rows และไม่รับประกันว่าจะคืนทุก row จึงต้องเก็บข้อจำกัดนี้ไว้ใน UI ([Search Console API: authorization](https://developers.google.com/webmaster-tools/v1/how-tos/authorizing), [Search Analytics query](https://developers.google.com/webmaster-tools/v1/searchanalytics/query))

หลักการ: read-only เป็นค่าเริ่มต้น, OAuth consent ชัดเจน, token เก็บใน OS credential store, disconnect/delete ได้ และไม่ส่ง query dataไปยัง AI จนผู้ใช้เลือกชุดข้อมูลและยืนยัน

### B. Google Ads Keyword Planner connector

ใช้ `KeywordPlanIdeaService` เพื่อขอ keyword ideas และ historical metrics จาก seed keyword/URL พร้อม location/language ที่ผู้ใช้กำหนด ต้องมี Google Ads customer, OAuth และ developer token ([Google Ads API: generate keyword ideas](https://developers.google.com/google-ads/api/docs/keyword-planning/generate-keyword-ideas), [Google Ads API method/scopes](https://developers.google.com/google-ads/api/reference/rpc/v22/KeywordPlanIdeaService/GenerateKeywordIdeas))

ตัวเลขเหล่านี้เป็น Ads planning data ไม่ใช่ organic-ranking promise และไม่ควรผสมกับ AI-estimated volume

### C. Google Business Profile connector

มีประโยชน์เมื่อธุรกิจของอาจารย์มีสถานที่/บริการท้องถิ่น:

- อ่าน performance เช่น searches, views, calls, directions และ website clicks ([Business Profile: performance metrics](https://support.google.com/business/answer/9918094?hl=en));
- ร่าง replies ต่อ reviews และ posts/offers/events;
- ตรวจ completeness ของข้อมูลธุรกิจ;
- วาง local content/offer experiment.

Local results พิจารณา relevance, distance และ prominence และไม่มีวิธีจ่ายเงินเพื่อขออันดับ local ที่ดีขึ้น ([Google Business Profile: local ranking](https://support.google.com/business/answer/7091?hl=en)) การใช้ API ต้องมี eligibility, Cloud project, business website และการอนุมัติ ([Business Profile API overview](https://developers.google.com/my-business/content/overview), [Basic setup](https://developers.google.com/my-business/content/basic-setup)) ที่สำคัญ policy ระบุว่าห้าม trigger review replies, Q&A, listing edits หรือ action อื่นโดยไม่มี prior specific and express consent ของผู้ใช้ จึงต้องใช้ confirm-per-action และ audit log ([Business Profile API policies](https://developers.google.com/my-business/content/policies))

### D. Meta official API connector — ทำหลังผ่าน permission/app-review design

พิจารณาเฉพาะเมื่อ manual workflow ไม่พอ และต้องออกแบบตาม Page access/permissions ปัจจุบันของ Meta ใหม่ในเวลานำไปทำจริง use cases ที่ควรขอเท่าที่จำเป็นคือ read-only insights import หรือ explicit user-approved publishing/reply action ไม่ควรใช้ browser scraping แทน API และไม่ควรเก็บ credentials/session cookies

หากต้องเชื่อม Inbox จริง ให้ใช้ official Messenger Platform เท่านั้น: Send API ใช้ Page access token และ `pages_messaging`; Conversations API ต้องใช้ permissions ที่เกี่ยวข้องและ Advanced Access เมื่อเข้าถึงผู้ใช้นอก app roles ส่วน Webhooks ใช้รับ event หลังเชื่อม App กับ Page และ subscribe field ที่ได้รับอนุญาต ([Meta official Messenger Send API collection](https://www.postman.com/meta/messenger-platform-api/folder/vilwbh4/send-api), [Meta Conversations API collection](https://www.postman.com/meta/messenger-platform-api/folder/22794852-255610cd-47f5-4f4d-b3fa-71aec360be9a), [Meta Webhooks collection](https://www.postman.com/meta/messenger-platform-api/folder/22794852-b5d97624-14d8-4e67-a2e4-529add49ca58)) ต้องตรวจ permission, messaging window และ Graph API version ปัจจุบันอีกครั้งตอน implement

ก่อนเปิด action ที่เขียนข้อมูล ต้องมี:

- least-privilege permission และแสดงว่าเพจ/งานใดได้รับอนุญาต;
- draft → owner/admin review → explicit confirm ต่อ action;
- immutable audit log, idempotency และ duplicate-post guard;
- rate/error handling และ revoke/disconnect;
- policy/app-review revalidation ก่อน release.

### E. GA4 read-only connector

ใช้ Google Analytics Data API เพื่อผูก UTM/landing page กับ sessions, conversions และ revenue ที่บัญชีอนุญาต แล้วแสดง funnel Facebook → website → lead/sale โดยไม่อ้างว่า UTM อย่างเดียวพิสูจน์ causal lift API รองรับ credential ของ user account หรือ service account ตามรูปแบบ deployment ([Google Analytics Data API](https://developers.google.com/analytics/devguides/reporting/data/v1)) ควรเริ่มด้วย read-only, เลือก property อย่างชัดเจน และไม่ส่งข้อมูลระดับบุคคลให้ AI

## ฟังก์ชันที่ไม่ควรสร้าง

- scrape Feed, comments, member lists, profiles, Inbox หรือ Google SERP;
- auto-like, auto-comment, auto-DM, bulk unsolicited messages หรือ auto-post โดยไม่มี confirmation;
- rank checker ที่ยิง automated queries เข้า Google Search;
- mass SEO page generation/doorway pages หรือ rewrite เนื้อหาคนอื่นเป็นจำนวนมาก;
- ปุ่ม “Instant Index” สำหรับบทความ/หน้าสินค้าทั่วไป เพราะ Google Indexing API จำกัดไว้ที่หน้า `JobPosting` หรือ livestream ที่มี `BroadcastEvent` ([Google Indexing API](https://developers.google.com/search/apis/indexing-api/v3/quickstart));
- fake testimonials, fake scarcity, fabricated sales/credentials หรือ schema ที่ไม่ตรงกับเนื้อหาจริง;
- แจกส่วนลด ของฟรี หรือสิ่งจูงใจแลกกับ Google review; Google ห้าม incentivized reviews ([Google Business Profile review policy](https://support.google.com/business/answer/3474122));
- AI ประเมิน search volume/rank แล้วแสดงเหมือนเป็นข้อมูล Google;
- เก็บ Facebook/Google cookies, passwords หรือ browser session tokens.

## ลำดับส่งมอบที่แนะนำ

1. **Sales Workspace MVP:** Offer, Content Campaign, Reply Copilot, Lead/Handoff, UTM และ Experiment Log
2. **SEO Copilot MVP:** Topic Map, Content Brief, Title/Meta Review, Internal Links, Structured Data และ Search Console CSV
3. **Cross-channel intelligence:** ผูก Content ID ↔ UTM ↔ landing page ↔ manually confirmed lead/sale และสร้าง learning report
4. **Search Console OAuth read-only:** เป็น connector แรก เพราะข้อมูลตรงกับ SEO outcome และ scope จำกัดได้
5. **GA4 read-only:** เชื่อม UTM กับ website outcomes เมื่อ tracking พร้อม
6. **Google Ads Keyword Planner / Business Profile:** เพิ่มเมื่อมีบัญชีและ use case จริง
7. **Meta API:** ทำเฉพาะ capability ที่ manual workflow พิสูจน์แล้วว่าคุ้มกับ permission/app-review burden

## Joyride ที่ควรสอนผู้ใช้อย่างละเอียด

เพื่อให้ผู้ใช้เรียนรู้โดยลงมือทำ tour ไม่ควรเป็นกล่องข้อความยาวชุดเดียว แต่แบ่งเป็นภารกิจที่กลับมาเปิดซ้ำได้:

1. **ตั้งฐานข้อเสนอ:** ชี้ Product facts → Audience → Proof → Claims guardrail → Save revision
2. **ทำ Facebook campaign แรก:** Objective → Angle → Content Pack → Review checklist → Sync extension → Click editor → Insert → user publishes
3. **ตอบลูกค้าอย่างปลอดภัย:** Paste selected message → Redact PII → Choose intent → Generate options → Verify facts → Copy/insert → Mark handoff
4. **ติดตามยอดขาย:** Create lead manually → Attach campaign/UTM → Update stage → Owner confirms sale → Commission becomes confirmed
5. **ทำ SEO brief แรก:** Seed topic → Intent inference → Evidence slots → Outline → Title/meta → People-first review → Export
6. **ตรวจบทความ:** Paste text/select file or explicitly fetch one URL → Review issues → Accept/reject recommendations → Generate internal-link tasks → Schema draft/test
7. **อ่านข้อมูล Search Console:** Import CSV → Explain clicks/impressions/CTR/position and limitations → Select opportunity → Create refresh experiment

ทุก tour ควรมี `ทำตัวอย่างให้ดู`, `ข้าม`, `ย้อนกลับ`, `จบภายหลัง`, progress และ reset ใน Help Center ขั้นที่มีผลภายนอกต้องบอกตรง ๆ ว่า Joyride จะไม่กดเผยแพร่หรือส่งข้อความแทนผู้ใช้

## Acceptance guardrails

- ทุก generated artifact มี source facts, revision, model/provider, timestamp และ reviewer status;
- ทุก suggestion แยก `fact`, `user input`, `AI inference` และ `metric from official import/API`;
- Facebook insertion เป็น plain text และเกิดหลัง user gesture;
- external write/send/publish ไม่มีใน MVP;
- user-triggered URL audit แสดง URL ที่จะ fetch และขอ action แบบเฉพาะเจาะจง;
- SEO UI ไม่รับประกัน ranking/indexing/rich result;
- API connector ใช้ least-privilege OAuth, secure token storage, disconnect และ audit log;
- experiment report แยก observed result ออกจาก causal claim;
- sensitive data มี redact/delete controls และ retention ที่ผู้ใช้กำหนดได้.

## React Joyride integration

ตรวจ React Joyride จากเอกสารและ source ของโครงการโดยตรง ณ วันที่เดียวกับงานวิจัยนี้ รุ่นปัจจุบันคือ `3.2.0` และประกาศ peer compatibility กับ React 16.8–19 จึงตรงกับ React 19 ของ Wails frontend นี้ ([official package source](https://github.com/gilbarbara/react-joyride/blob/main/package.json), [npm package](https://www.npmjs.com/package/react-joyride))

แนวทางที่เหมาะกับแอปนี้:

- ใช้ `useJoyride()` ซึ่งเป็น interface หลักของ v3 และ render ค่า `Tour` ที่ hook คืนมา;
- ใช้ uncontrolled mode เป็นค่าเริ่มต้น และใช้ async `before` hook เพื่อสลับ workspace/tab ก่อนแสดงขั้นถัดไป แทนการขับ `stepIndex` เอง;
- เก็บเฉพาะ mission ID, step ล่าสุด และสถานะ finished/skipped ใน local storage เพื่อเปิดต่อหรือเริ่มใหม่ได้;
- ใช้ `onEvent` บันทึกความคืบหน้าและจัดการ `error:target_not_found`; ตั้ง `targetWaitTimeout` สำหรับส่วน UI ที่ mount หลังสลับ workspace;
- เปิด focus trap, keyboard navigation และ ARIA ของตัว library ไว้ ใช้ปุ่มย้อนกลับ/ข้าม/จบภายหลังภาษาไทย และคืน focus หลังปิด;
- ปิด overlay click เพื่อไม่ให้ทัวร์หายเพราะคลิกพลาด และตั้ง `blockTargetInteraction` ในขั้นที่มีปุ่มสร้าง/บันทึก;
- เมื่อ `prefers-reduced-motion: reduce` ให้ตั้ง scroll duration เป็นศูนย์และปิด beacon animation;
- แบ่งเป็นภารกิจสั้นที่เปิดซ้ำได้ ไม่ทำทัวร์ยาวบังคับตั้งแต่ครั้งแรก.

React Joyride v3 ระบุว่า hook เหมาะกับทัวร์ที่ต้องประสานกับ state ภายนอก, `before` hook รองรับ dynamic UI, target รองรับ selector/ref/function และระบบจะรอ target ตาม timeout ที่กำหนด ([useJoyride](https://react-joyride.com/docs/hook), [How it works](https://react-joyride.com/docs/how-it-works), [Events](https://react-joyride.com/docs/events), [New in v3](https://react-joyride.com/docs/new-in-v3))

## Competitor research และแผน audience ที่ใช้ข้อมูลอย่างถูกสิทธิ์

### ศึกษาโฆษณาคู่แข่งจาก Ads Library แบบ manual

Meta Ads Library ให้ค้นหาโฆษณาที่กำลังรันบนผลิตภัณฑ์ของ Meta และดูข้อความ สื่อ วันที่เริ่มรัน แพลตฟอร์ม Ad ID และลิงก์โฆษณาได้ สำหรับโฆษณาทั่วไป Library ไม่ได้เปิดเผย audience จริง ยอดขาย หรือผลลัพธ์ของแคมเปญ ส่วนข้อมูลโฆษณาที่หยุดรัน ค่าใช้จ่าย ช่วง impressions และ demographics เพิ่มเติมมีไว้สำหรับโฆษณาเรื่องประเด็นสังคม การเลือกตั้ง หรือการเมืองตามเงื่อนไขของ Meta ([Meta Ads Library Help](https://www.facebook.com/help/259468828226154), [Meta Ads Library](https://www.facebook.com/ads/library/))

ฟังก์ชันที่ทำได้ใน MVP คือให้ผู้ใช้เปิด Ads Library เอง แล้วบันทึกลิงก์โฆษณา เพจ วันที่สังเกต รูปแบบ creative, hook, offer, CTA และ landing page ที่มองเห็น จากนั้น AI ช่วย:

- แยก `สิ่งที่เห็นในโฆษณา` ออกจาก `ข้อสันนิษฐานของ AI`;
- เทียบ message, proof, format และ CTA กับ offer ที่เจ้าของเพจอนุมัติแล้ว;
- ทำ creative gap และ test hypothesis โดยไม่สรุปว่าโฆษณาคู่แข่งขายดี;
- เก็บภาพอ้างอิงหรือลิงก์ที่ผู้ใช้เลือกเอง พร้อมวันที่ตรวจ เพราะโฆษณาอาจหยุดหรือเปลี่ยนภายหลัง.

โปรแกรมไม่ควร crawl Ads Library, ดึงโฆษณาจำนวนมาก หรือเก็บ followers, reactions, comments, profiles และรายชื่อลูกค้าของเพจอื่น Meta นิยาม scraping ว่าเป็นการเก็บข้อมูลอัตโนมัติจากเว็บไซต์หรือ interface ที่สร้างไว้ให้คนใช้ และดำเนินมาตรการกับการ scraping ที่ไม่ได้รับอนุญาต ([Facebook Help: Data scraping](https://www.facebook.com/help/463983701520800))

### ตรวจแหล่งที่มาของ audience ก่อนวางแผนยิงโฆษณา

Custom Audience สร้างได้จากแหล่งข้อมูลของผู้ลงโฆษณาเอง หรือข้อมูล engagement ที่ Meta รองรับ เช่น website, app, catalog, customer list, offline activity, video, lead form, Instagram account และ Facebook Page engagement ([Meta: About custom audiences](https://www.facebook.com/business/help/744354708981227), [Meta: About engagement custom audiences](https://www.facebook.com/business/help/1090330204367211)) ขั้นตอนทางการสำหรับ Facebook Page engagement ให้ผู้ลงโฆษณาเลือกเพจของตนใน Ads Manager ดังนั้น MVP เก็บเพียงการตั้งค่าแผนและไม่รับรายชื่อบุคคล ([Meta: Create a Facebook Page engagement Custom Audience](https://www.facebook.com/business/help/221146184973131))

ถ้าใช้ customer list ข้อกำหนดของ Meta ระบุว่าผู้ลงโฆษณาต้องมีสิทธิ การอนุญาต และฐานกฎหมายที่จำเป็นต่อการเปิดเผยและใช้ข้อมูล ต้องเคารพ opt-out และลบบุคคลที่ถอนความยินยอมหรือคัดค้านตามสิทธิที่รับรองไว้ ห้ามใช้ข้อมูลเด็กอายุต่ำกว่า 13 ปี identifiers ที่ Meta ไม่อนุญาต หรือข้อมูลอ่อนไหวตามหมวดที่กำหนด แม้ Meta จะ hash ข้อมูลติดต่อก่อนรับข้อมูล ข้อกำหนดเหล่านี้ยังคงใช้ และ Meta ไม่เปิดเผยว่าใครบ้างอยู่ใน audience ที่จับคู่ได้ ([Customer List Custom Audiences Terms](https://www.facebook.com/legal/terms/customaudience))

Audience Source Checker ควรรับเฉพาะ metadata ไม่รับรายชื่อบุคคลดิบ แล้วตรวจ:

- แหล่งข้อมูลและผู้ควบคุมข้อมูล เช่น CRM ของเจ้าของเพจ, website, app หรือ engagement กับเพจของตน;
- วัตถุประสงค์การใช้ สิทธิหรือฐานกฎหมาย วิธีรับ opt-out และอายุข้อมูล;
- การมีข้อมูลเด็ก ข้อมูลอ่อนไหว หรือ identifier ที่ไม่ควรใช้;
- สถานะ `ใช้วางแผนได้`, `ต้องให้เจ้าของตรวจ` หรือ `ห้ามใช้` พร้อมเหตุผล;
- การอนุมัติของเจ้าของเพจก่อนนำแผนไปตั้งค่าใน Ads Manager.

เครื่องมือนี้ช่วยตรวจข้อมูลภายในงานเท่านั้น ผู้ใช้ต้องตรวจข้อกำหนด Meta และกฎหมายที่ใช้กับธุรกิจจริงอีกครั้งก่อน upload.

### Retargeting, Lookalike และ Advantage+ plan

Meta รองรับ retargeting จาก customer list ที่ธุรกิจเก็บอย่างถูกสิทธิ์, website activity ผ่าน Meta Pixel และ app activity ผ่าน SDK รวมถึง Custom Audience จาก engagement ที่เกิดกับธุรกิจของผู้ลงโฆษณาเอง ([Meta: Retargeting](https://www.facebook.com/business/goals/retargeting)) Engagement Custom Audience ใช้เป็น source สำหรับ Lookalike Audience ได้ ส่วน Lookalike ใช้ source audience เพื่อหา “คนใหม่” ที่มีลักษณะคล้ายกัน แผนจึงอ้างอิงคุณสมบัติของ source audience โดยไม่ใช้รายชื่อ followers หรือลูกค้าของเพจอื่น ([Meta: About Lookalike Audiences](https://www.facebook.com/business/help/164749007013531))

Advantage+ audience ใช้ข้อมูล เช่น past conversions, Meta Pixel และ interactions กับโฆษณาก่อนหน้าเพื่อช่วยหา audience และอาจขยายออกนอก audience suggestion เมื่อระบบคาดว่าจะช่วยผลลัพธ์ เอกสาร Meta แนะนำให้ A/B test Advantage+ กับแคมเปญเกือบทุกประเภท ยกเว้น retargeting ดังนั้นแผนต้องบันทึกให้ชัดว่าจะจำกัดอยู่ใน first-party audience หรือยอมให้ระบบขยาย และให้เจ้าของอนุมัติตัวเลือกนี้ก่อนตั้งค่า ([Meta: About Advantage+ Audience](https://www.facebook.com/business/help/273363992030035), [Meta: About Advantage+ custom audience](https://www.facebook.com/business/help/414975413946182))

ผลลัพธ์ของฟังก์ชันควรเป็นแผนที่มี source audience, inclusion/exclusion, retention window, campaign objective, creative angle, KPI, ข้อจำกัด และช่อง owner approval โปรแกรมยังไม่สร้างหรือ upload audience, ไม่เปิด Ads Manager แทนผู้ใช้ และไม่กด publish แคมเปญ.
