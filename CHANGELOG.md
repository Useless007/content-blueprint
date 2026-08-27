# Changelog

การเปลี่ยนแปลงที่ผู้ใช้เห็นจะบันทึกไว้ในไฟล์นี้ โครงการใช้ Semantic Versioning สำหรับ tag และ GitHub Release

## [0.3.0] - 2026-08-28

### Added

- Windows desktop app สำหรับ Facebook Content Pack, Growth Hub และ SEO Workspace
- Growth Playbook 10 แบบ พร้อม Lead Card, Estimated/Confirmed Commission, UTM Builder และ Experiment Log
- Claude Code และ Codex direct-CLI workflows แบบ Quick draft และ AI Team
- AI Studio แสดง Strategist, Producer/Copywriter, Reviewer และ Browser Courier จาก stage events จริง
- Joyride ภาษาไทย 8 ภารกิจ พร้อมบันทึกเฉพาะสถานะการเรียนใน browser storage
- MCP companion 7 tools สำหรับ Facebook และ Growth contracts
- Chrome/Brave Manifest V3 side panel และ Native Messaging handoff แบบ plain text
- GitHub Releases updater ที่แยกการตรวจ, ดาวน์โหลด และเปิด installer พร้อมตรวจ SHA-256
- GitHub Actions สำหรับ CI, browser smoke และ Windows release

### Security and privacy boundaries

- ทุก Pack ผูกกับ `BriefRevision`; ผล revision เก่าถูกระบุว่า stale
- Extension ไม่ scrape Facebook, ไม่อ่าน token/cookie และไม่กด Post, Schedule หรือ Send
- MCP ไม่มี tool เปิด browser, ส่งข้อความ, เผยแพร่ หรืออัปโหลด audience
- ข้อมูลในเครื่องยังไม่ได้เข้ารหัสที่ระดับแอป
- Windows executable และ installer รุ่นนี้ยังไม่ได้เซ็น Authenticode

### Known limitations

- Browser DOM ที่เปลี่ยนอาจทำให้ plain-text insertion ต้องปรับตาม
- AI output ยังต้องตรวจข้อเท็จจริง สิทธิ์ใช้งาน และ compliance โดยมนุษย์
- Checksum ที่อยู่ใน GitHub Release เดียวกับ installer ตรวจความเสียหายของไฟล์ได้ แต่ไม่ใช่หลักฐานตัวตนของ publisher แบบ Authenticode

[0.3.0]: https://github.com/Useless007/content-blueprint/releases/tag/v0.3.0
