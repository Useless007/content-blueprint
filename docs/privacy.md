# Privacy and data handling

Content Blueprint เป็น local-first workbench ไม่มีระบบบัญชีของโครงการหรือฐานข้อมูลกลาง การใช้ AI, GitHub updater และ browser extension ยังสร้างเส้นทางข้อมูลที่ผู้ใช้ควรรู้ก่อนใส่ Brief จริง

## ข้อมูลที่เก็บในเครื่อง

ตำแหน่งโดยปริยาย:

```text
%AppData%\ContentBlueprint\settings.json
%AppData%\ContentBlueprint\projects\<project-id>.json
%AppData%\ContentBlueprint\FacebookCompanion\facebook-brief.json
%AppData%\ContentBlueprint\FacebookCompanion\facebook-pack.json
%AppData%\ContentBlueprint\GrowthWorkbench\growth-brief.json
%AppData%\ContentBlueprint\GrowthWorkbench\growth-pack.json
%AppData%\ContentBlueprint\GrowthWorkbench\growth-leads.json
%AppData%\ContentBlueprint\GrowthWorkbench\growth-experiments.json
```

| ข้อมูล | เหตุผลที่เก็บ | หมายเหตุ |
| --- | --- | --- |
| Provider settings และ SEO projects | จำ provider ที่เลือก, Brief, draft, sources และผลตรวจคุณภาพ | อยู่ใน `settings.json` และ `projects\<project-id>.json`; JSON ไม่ได้เข้ารหัสที่ระดับแอป |
| Brief และ Evidence Notes | สร้างงานและตรวจ revision | JSON ไม่ได้เข้ารหัสที่ระดับแอป |
| Content/Growth Pack | ตรวจ แก้ ส่งต่อ และกัน stale result | เก็บเฉพาะผลล่าสุดตาม contract ปัจจุบัน |
| Lead Cards | ติดตามโอกาสขายที่ผู้ใช้สร้างเอง | ไม่ได้ import จาก Facebook profile/inbox |
| Commission | แยกค่าประมาณการกับยอดที่เจ้าของงานยืนยัน | ไม่ใช่ระบบ payout หรือ invoice |
| Experiment records | บันทึก hypothesis, version และ observed metrics | โปรแกรมไม่ประกาศผู้ชนะเอง |
| Joyride state | จำภารกิจและขั้นที่เรียนจบ | ไม่เก็บ Brief หรือ Pack ใน onboarding state |
| WebView local storage | จำ workspace/tab, Growth draft ที่ตัด field ซึ่งทำเครื่องหมาย sensitive, สถานะ Joyride และเวลาที่ลองตรวจ update ล่าสุด | อยู่ใน WebView2 profile ของผู้ใช้จนกว่าจะล้างข้อมูลแอป |
| Extension local storage | จำ draft กับค่า `useGrounding` ของ side panel | อยู่ใน `chrome.storage.local`; ไม่ควรใส่ข้อมูลส่วนบุคคลที่ไม่จำเป็น |
| Extension session storage | เก็บ Gemini API key เมื่อผู้ใช้กรอก | อยู่ใน `chrome.storage.session` และสิ้นสุดตาม browser session |

ถอนโปรแกรมแล้วข้อมูลใน `%AppData%\ContentBlueprint` จะยังอยู่ เพื่อป้องกันการลบงานโดยไม่ตั้งใจ ผู้ใช้ต้องสำรองและลบโฟลเดอร์นี้เองเมื่อต้องการล้างข้อมูล

## ข้อมูลที่ออกจากเครื่อง

| เมื่อใช้ | ปลายทาง | ข้อมูลที่ส่ง |
| --- | --- | --- |
| Claude Code direct CLI | provider ตามบัญชี Claude ที่ล็อกอิน | Brief, Evidence Notes และ handoff ที่ stage ต้องใช้ |
| Codex direct CLI | provider ตามบัญชี Codex ที่ล็อกอิน | Brief, Evidence Notes และ handoff ที่ stage ต้องใช้ |
| Claude/Codex ผ่าน MCP | companion แลกข้อมูลกับ AI client ผ่าน local stdio; จากนั้น client ติดต่อ provider ตามบัญชีที่ล็อกอิน | Brief หรือ Pack ที่ tool ส่งคืน/บันทึกอาจอยู่ใน model context และถูกส่งตามนโยบายของ AI client/provider แม้ MCP server จะรันในเครื่อง |
| Gemini mode | Google Gemini endpoint | Prompt/Brief และเนื้อหาที่เลือกสร้าง |
| Update check | GitHub REST API และ GitHub Releases | เวอร์ชันแอปใน User-Agent และคำขอ release/asset ไม่มี Brief หรือ customer data |

Content Blueprint ไม่เปลี่ยนนโยบาย retention, training, billing หรือ privacy ของ provider ตรวจข้อกำหนดของบัญชีและองค์กรที่ใช้อยู่ก่อนส่งข้อมูลลับ

## Claude/Codex credentials

Direct-CLI route ใช้ session ที่ CLI จัดการอยู่ แอปไม่ขอให้คัดลอก API key ลง UI และไม่บันทึก credential ของ Claude/Codex ลง Brief/Pack

อย่าใส่ token ใน environment variable ชื่อที่ไม่จำเป็นหรือแนบ output การล็อกอินใน issue ถึงแม้ worker จะได้รับ environment allowlist แบบจำกัด การตั้งค่าและ credential lifecycle ยังเป็นหน้าที่ของ provider CLI

## Gemini credentials

Desktop SEO ใช้ `GEMINI_API_KEY` จาก environment ได้ หรือรับ key ที่ผู้ใช้กรอกในหน้าตั้งค่าแล้วเก็บไว้เฉพาะ React memory ของการเปิดแอปครั้งนั้น แอปไม่เขียน key จากช่องนี้ลง `settings.json` หรือ project และค่าจะหายเมื่อปิดแอป

Chrome/Brave extension แยกการจัดการ key ออกจาก desktop โดยเก็บ Gemini key ที่กรอกใน `chrome.storage.session` ซึ่งสิ้นสุดตาม browser session และไม่รวม key ใน Brief หรือ Pack

## Facebook and browser boundary

Extension:

- ไม่อ่าน cookie, token, inbox, DM, follower list, reaction list หรือ customer list
- ไม่ crawl Page, post, comment หรือ public profile
- ไม่ค้นหาหรือกด Post, Schedule, Send และ Ads controls
- รับ Pack ผ่าน Native Messaging จาก companion ที่ลงทะเบียนกับ extension origin เดียว
- แทรก plain text ลง editor ที่ผู้ใช้โฟกัสเอง

Public visibility ไม่ใช่ consent ให้นำข้อมูลรายบุคคลไปเก็บเป็นฐานลูกค้า โปรแกรมจึงไม่สร้าง scraper หรือ audience list จากเพจอื่น

การวาง audience plan ใช้ได้กับข้อมูลที่เจ้าของเพจมีสิทธิ์และอนุมัติ เช่น first-party customer list ที่มีฐานกฎหมายและ opt-out, engagement ของเพจตนเอง, website/app activity, lead forms และ aggregate observations ที่ไม่มีข้อมูลระบุตัวบุคคล ผู้ใช้ต้องตรวจนโยบาย Meta/Google และกฎหมายที่ใช้กับตนเอง

## Update privacy and integrity

Automatic check เกิดไม่เกินหนึ่งครั้งต่อ 24 ชั่วโมงต่อการติดตั้ง และ manual check เกิดเมื่อผู้ใช้กด แอปไม่ฝัง GitHub PAT เพื่อเพิ่ม rate limit

Download เริ่มหลังผู้ใช้ยืนยัน Backend รับ asset จาก repository คงที่ ตรวจ SHA-256 และเก็บไฟล์ใน temporary directory ที่สร้างสำหรับ update นั้น ผู้ใช้ต้องยืนยันอีกครั้งก่อนเปิด installer แบบมองเห็นได้

รุ่น `v0.3.0` ยังไม่ได้เซ็น Authenticode `SHA256SUMS.txt` ที่มากับ release ตรวจ digest ได้ แต่ไม่ได้สร้าง trust anchor แยกจากบัญชี GitHub เจ้าของ release

## สิ่งที่ไม่ควรใส่ในแอปหรือรายงานปัญหา

- password, session cookie, OAuth/access token, API key หรือ private key
- รายชื่อบุคคล เบอร์โทร อีเมล หรือข้อความลูกค้าที่ไม่จำเป็น
- customer export หรือ audience file จริง
- Brief ที่มีข้อมูลสัญญา ราคา หรือแผนการตลาดลับ หากบัญชี AI provider ไม่อนุญาตข้อมูลชนิดนั้น
- screenshot/log ที่ยังเห็น path ผู้ใช้ ข้อมูลบัญชี หรือข้อความลูกค้า

รายงานช่องโหว่ตาม [SECURITY.md](../SECURITY.md) และส่งตัวอย่างสังเคราะห์ที่ทำให้เกิดปัญหาได้เหมือนกัน
