# Architecture and trust boundaries

Content Blueprint แยกการสร้างชิ้นงาน, การตรวจ contract, การเก็บในเครื่อง และการส่งไป browser ออกจากกัน แต่ละ boundary รับข้อมูลเท่าที่ต้องใช้

## Component map

```text
┌─────────────────────────────────────────────────────────────┐
│ Wails desktop                                               │
│ React UI → Go application façade → domain validators/store │
│       │              │                  │                   │
│       │              ├─ direct CLI ─────┼─► Claude / Codex │
│       │              │                  │                   │
│       │              └─ stage events ───┴─► AI Studio      │
└───────┼──────────────────────┬──────────────────────────────┘
        │ local JSON           │ revision-bound Pack
        ▼                      ▼
  AppData stores       Companion executable
                              ▲  ▲
                         MCP  │  │ Native Messaging
                              │  ▼
                     Claude/Codex  Chrome/Brave side panel
                                        │
                                        ▼ plain text only
                                  user-focused editor
```

## Contracts before prompts

Facebook ใช้ typed Brief และ Content Pack 9 ส่วน Growth Hub ใช้ Playbook catalog, Growth Brief และ Growth Pack ที่จำกัดชนิด block ทั้งสองทางคำนวณ `BriefRevision` จาก Brief ต้นฉบับ

Backend ปฏิเสธ Pack เมื่อ:

- JSON ไม่ตรง schema
- field หรือ block เกินขนาดที่กำหนด
- `briefRevision` ไม่ตรงกับ Brief ปัจจุบัน
- evidence basis หรือ `sourceIds` ไม่ตรง contract ของ Playbook
- AI ตั้งสถานะที่ต้องเป็นการตัดสินใจของมนุษย์

Structured output ลดความคลุมเครือของ handoff แต่ไม่รับรองว่าข้อความจริง ถูกกฎหมาย หรือขายได้ ผู้ใช้ต้องตรวจเนื้อหาก่อนใช้งานภายนอก

## Direct CLI workflow

Wails เปิด Claude Code หรือ Codex เป็น child process โดยส่ง argument แยกค่า ไม่ประกอบ shell command

- Quick draft เปิด worker หนึ่ง process
- AI Team เปิด Strategist, Producer/Copywriter และ Reviewer แยกตามลำดับ
- แต่ละ stage รับเฉพาะ Brief และ handoff ที่ผ่าน validator
- stage ไม่รับ session หรือสิทธิ์จาก worker ก่อนหน้า
- frontend ส่ง prompt, schema หรือ path ตามใจไปแทน trusted instructions ไม่ได้

AI Studio รับ stage events ที่จำกัดข้อมูลแล้ว ตัวละครเดิน นั่ง และส่งแฟ้มตามสถานะจริง ภาพเหล่านี้ไม่ใช่ agent runtime และไม่ให้สิทธิ์อ่านไฟล์ เปิด browser หรือส่งงานภายนอก

## MCP reverse workflow

Companion executable เป็น MCP stdio server ให้ Claude/Codex ที่ผู้ใช้เปิดเองอ่าน Brief และบันทึก Pack กลับมา มี tools 7 ตัวที่กำหนดชื่อและ schema ล่วงหน้า MCP ไม่สามารถ:

- รับ caller-defined prompt, schema หรือ filesystem path
- เริ่ม model process
- เปิด browser หรืออ่านหน้า Facebook
- scrape profile, comment, follower หรือ inbox
- โพสต์ ส่งข้อความ หรืออัปโหลด audience

การอ่านและเขียนทั้งหมดผ่าน revision checks เดียวกับ Wails

## Browser handoff

Installer ลงทะเบียน Native Messaging host ใน `HKCU` สำหรับ Chrome และ Brave พร้อมล็อก allowed origin ไว้ที่ extension ID `ppncejmpiekmkepaeccdnpnpgdcfafje`

Extension ใช้ Manifest V3 side panel Content script จดจำ element ที่ผู้ใช้โฟกัสจริงและแทรกด้วย text operation ไม่ตีความ model output เป็น HTML โค้ดไม่มี selector หรือ action สำหรับ Post, Schedule หรือ Send

Real-browser E2E ใช้ profile ชั่วคราวกับ HTTPS fixture ที่ map เป็น hostname ทดสอบ ไม่ล็อกอิน Facebook และยืนยันว่า Post button ไม่ถูกคลิก

## Local persistence

Stores ใช้ JSON ใน `%AppData%\ContentBlueprint` แยก Facebook Companion และ Growth Workbench การตั้ง `CONTENT_BLUEPRINT_DATA_DIR` มีไว้สำหรับ development/test เพื่อไม่แตะข้อมูลจริง

ไฟล์ไม่ได้เข้ารหัสที่ระดับแอป อย่าใส่ password, cookie, access token, private key หรือข้อมูลลูกค้าที่ไม่จำเป็น

## Update flow

Updater เชื่อถือ repository คงที่ `Useless007/content-blueprint` และทำงานสามจังหวะ:

```text
Check latest stable release
        │ user chooses
        ▼
Download exact installer + SHA256SUMS.txt to private temp directory
        │ digest matches
        ▼
User confirms → launch visible NSIS installer
```

Frontend ส่ง URL หรือ destination path เข้า updater ไม่ได้ Backend จำกัด response/file size, บังคับ HTTPS, ตรวจ stable SemVer, ตรวจ asset name/repository path และ hash ไฟล์อีกครั้งก่อนเปิด

Checksum จาก release account เดียวกันช่วยจับไฟล์เสียหรือไฟล์ที่ถูกแทนระหว่างทาง แต่ไม่ใช่ publisher identity รุ่น `v0.3.0` จึงเปิดเผยตรง ๆ ว่ายังไม่มี Authenticode signature

## Human approval boundary

การกระทำต่อไปนี้ต้องเกิดนอก agent workflow หรือจากการกดยืนยันของผู้ใช้:

- ยืนยันข้อเท็จจริงและสิทธิ์ใช้ Evidence Notes
- อนุมัติ Growth Pack และยอด Confirmed Commission
- เลือกข้อความที่จะคัดลอกหรือแทรก
- กด Post, Schedule, Send หรือสร้าง Ads audience ในระบบทางการ
- ดาวน์โหลดและเปิด update installer

รายละเอียดข้อมูลและ network destinations อยู่ใน [privacy.md](privacy.md)
