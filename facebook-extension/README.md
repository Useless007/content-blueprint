# Content Blueprint for Facebook — Chrome/Brave Extension

Manifest V3 side-panel extension v0.3.0 สำหรับรับ Facebook Content Pack และ Growth Pack จาก Wails หรือ Claude/Codex ผ่าน MCP รวมถึงสร้าง Facebook Content Pack ด้วย Gemini โดยตรง แล้วช่วยแทรกข้อความธรรมดาลงใน Facebook editor ที่ผู้ใช้เลือกเอง

Extension นี้เป็นผู้ช่วยเตรียมร่าง ไม่ใช่ Facebook bot: ไม่อ่านหน้าเพจ, ไม่ scrape, ไม่เปิดโพสต์/คอมเมนต์/DM, ไม่คลิก Post และไม่ส่งข้อความแทนผู้ใช้

## เลือก workflow

### 1. Wails → Claude/Codex CLI → Extension

เหมาะกับการทำงานประจำ:

1. สร้าง Content Pack ใน Facebook workspace ของ Wails
2. เลือก AI Team หรือ Quick draft และเลือก Claude Code/Codex CLI ที่ล็อกอินแล้ว
3. เมื่อ Browser Courier แสดงว่างานพร้อม เปิด extension แล้วกด **รับผลล่าสุด**
4. ตรวจร่าง เลือก output แล้วแทรกลง Facebook

ไม่ต้องใส่ API key เพิ่มใน extension แต่ยังใช้สิทธิ์และโควตาของบัญชี CLI

### 2. Extension → MCP → Claude/Codex ที่ผู้ใช้เปิดเอง

เหมาะเมื่ออยากควบคุมบทสนทนาใน Claude/Codex:

1. กรอก Brief ใน side panel แล้วกด **ส่งโจทย์เข้า MCP**
2. เปิด Claude Code/Codex และสั่งให้ server `content-blueprint-facebook` เรียก `get_facebook_brief`
3. ให้ AI สร้างครบทุก field แล้วเรียก `save_facebook_pack` ด้วย revision ที่ได้รับ
4. กลับมา extension แล้วกด **รับผลล่าสุด**

### 3. Growth Hub → Extension

ส่วน Growth Pack เป็นทางรับผลที่สร้างไว้แล้ว ไม่ได้สร้างหรือแก้ Growth Brief ใน extension:

1. สร้าง Growth Brief/Pack ใน Wails หรือบันทึกผลกลับด้วย MCP
2. ใน side panel กด **รับ Growth Pack ล่าสุด** ผ่าน Native Messaging
3. ตรวจ finite blocks, evidence basis, risk flags และสถานะ review
4. ผล `stale` หรือ `rejected` อ่านและคัดลอกได้แต่แทรกไม่ได้; `needs_review` ต้องติ๊กยืนยันก่อนแทรก ส่วน `approved` แทรกได้ตามปกติ

Service worker ตรวจ typed snapshot และ block schema ก่อนส่งให้หน้าจอ HTML จากโมเดลแสดงเป็นข้อความ ไม่ถูก render เป็น DOM และการแทรกยังจำกัด 50,000 ตัวอักษร

### 4. Gemini API ใน extension

เปิดการตั้งค่า ใส่ Gemini API key แล้วกดสร้างจาก side panel ได้โดยตรง Key ถูกเก็บใน `chrome.storage.session` ของ browser session ปัจจุบัน ไม่ถูกเขียนรวมกับ Draft หรือ Content Pack โหมดนี้แยกจาก Claude/Codex CLI และอาจมีค่าใช้จ่าย/โควตาตามบัญชี Gemini

## ติดตั้งบน Windows

วิธีพร้อมใช้คือรัน `build\bin\content-blueprint-amd64-installer.exe` ตัวติดตั้งจะวาง companion, Extension runtime และลงทะเบียน Native Messaging แบบ per-user ให้แล้ว จากนั้นข้ามไปขั้นตอน **Load unpacked extension** โดยเลือก `%LOCALAPPDATA%\Programs\Content Blueprint\facebook-extension` ส่วนขั้นตอนสร้าง binary/ลงทะเบียนด้านล่างใช้สำหรับ development จาก source tree

### 1. สร้าง companion executable

จาก root ของโปรเจกต์:

```powershell
Set-Location C:\path\to\content-blueprint
go build -trimpath -o build\bin\content-blueprint-companion.exe .\cmd\content-blueprint-companion
```

Companion binary ตัวเดียวทำได้สองโหมด:

- ไม่มี argument หรือ `mcp` — MCP server ผ่าน stdio
- เมื่อ Chrome/Brave ส่ง extension origin — Native Messaging host ผ่าน length-prefixed JSON

### 2. ลงทะเบียน Native Messaging host

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\facebook-extension\native-host\install-native-host.ps1
```

Installer จะ:

- สร้าง host manifest ที่ `%LocalAppData%\ContentBlueprint\NativeMessaging\com.contentblueprint.facebook.json`
- ลงทะเบียน `com.contentblueprint.facebook` ใน `HKCU` สำหรับ Google Chrome และ Brave
- อนุญาตเฉพาะ `chrome-extension://ppncejmpiekmkepaeccdnpnpgdcfafje/`
- ชี้ host manifest ไปยัง companion executable ที่ resolve เป็น absolute path

ไม่ต้องใช้สิทธิ์ Administrator เพราะลงทะเบียนเฉพาะผู้ใช้ปัจจุบัน อย่าย้าย companion executable หลังขั้นตอนนี้ หรือรัน installer ใหม่เมื่อ path เปลี่ยน

ถ้า binary อยู่ที่อื่น ให้ระบุ path ชัดเจน:

```powershell
.\facebook-extension\native-host\install-native-host.ps1 `
  -CompanionExecutable "D:\Tools\content-blueprint-companion.exe"
```

### 3. Load unpacked extension

Chrome:

1. เปิด `chrome://extensions`
2. เปิด **Developer mode**
3. กด **Load unpacked**
4. เลือก `C:\path\to\content-blueprint\facebook-extension` หรือโฟลเดอร์
   `%LOCALAPPDATA%\Programs\Content Blueprint\facebook-extension` หลังติดตั้งแอป

Brave:

1. เปิด `brave://extensions`
2. เปิด **Developer mode**
3. กด **Load unpacked**
4. เลือกโฟลเดอร์เดียวกัน

ตรวจว่า extension ID เป็น:

```text
ppncejmpiekmkepaeccdnpnpgdcfafje
```

ค่า `key` ใน `manifest.json` ทำให้ unpacked extension มี ID คงที่ซึ่งต้องตรงกับ Native Host ห้ามแก้หรือลบ key โดยไม่อัปเดต host allowlist และติดตั้ง host ใหม่ ดูรายละเอียดจาก [Chrome manifest key](https://developer.chrome.com/docs/extensions/reference/manifest/key)

หลังติดตั้ง Native Host ให้กด **Reload** ที่หน้า extensions แล้วกดไอคอน Content Blueprint เพื่อเปิด side panel เอกสารของ Brave ระบุว่าสามารถติดตั้ง extension จาก Chrome ecosystem ได้ ดู [Using Chrome extensions in Brave](https://brave.com/learn/using-chrome-extensions-in-brave/)

## ติดตั้ง MCP

สร้าง companion ก่อน แล้วเลือก client:

```powershell
# ลงทะเบียนทั้งสองตัว
.\facebook-extension\native-host\install-mcp.ps1 -Target Both

# หรือเฉพาะตัวที่ติดตั้งอยู่
.\facebook-extension\native-host\install-mcp.ps1 -Target Codex
.\facebook-extension\native-host\install-mcp.ps1 -Target Claude
```

Script เรียกคำสั่งทางการในรูปแบบนี้:

```powershell
codex mcp add content-blueprint-facebook -- <absolute-path-to-companion.exe>
claude mcp add --transport stdio --scope user content-blueprint-facebook -- <absolute-path-to-companion.exe>
```

ตรวจการเชื่อมต่อใน client และอ้างอิง [Codex MCP](https://learn.chatgpt.com/docs/extend/mcp) หรือ [Claude Code MCP](https://code.claude.com/docs/en/mcp)

MCP tools ที่เปิดให้ใช้:

| Tool | ผลกระทบ |
| --- | --- |
| `get_facebook_brief` | อ่าน Brief และ revision ล่าสุดจาก local store |
| `save_facebook_pack` | ตรวจ schema กับ revision แล้วเขียน Content Pack ใน local store |
| `get_latest_facebook_pack` | อ่าน Content Pack ล่าสุดและสถานะ stale |
| `list_growth_playbooks` | อ่าน catalog ของ Growth Playbook ที่เชื่อถือได้ |
| `get_growth_brief` | อ่าน Growth Brief, exact `briefRevision`, trusted task contract และ schema ตาม Brief |
| `save_growth_pack` | strict decode, ตรวจ GrowthPack/revision แล้วบันทึกลง local store |
| `get_latest_growth_pack` | อ่าน Growth Pack ล่าสุดพร้อมสถานะ stale/review |

ทั้งเจ็ด tools ไม่รับ prompt/schema/path ตามใจผู้เรียก ไม่รัน AI ไม่เชื่อม Facebook และไม่มีคำสั่ง publish, schedule, scrape, message, เปิด browser หรืออัปโหลด audience

## วิธีแทรกข้อความใน Facebook

1. เปิดหน้า Facebook ที่บัญชีของคุณมีสิทธิ์ใช้งาน
2. คลิกในช่องสร้างโพสต์ คอมเมนต์ หรือ editor ที่ต้องการให้เกิด focus จริง
3. เปิด side panel แล้วเลือก output เช่นโพสต์ยาว โพสต์สั้น hook หรือ reply
4. กด **แทรกใน Facebook**
5. ตรวจข้อความใน editor และกดโพสต์เองเมื่อพร้อม

Content script จำเฉพาะ element ที่ได้รับ trusted `focusin`/`pointerdown` ล่าสุด มันไม่อ่านข้อความเดิมใน element และไม่ค้นหาปุ่ม Post ถ้าไม่มี editor ที่ผู้ใช้โฟกัส การแทรกจะถูกปฏิเสธ

Growth Pack ใช้ editor เดียวกัน แต่มี review guard เพิ่ม: `stale` และ `rejected` ห้ามแทรก, `needs_review` ต้องยืนยันด้วย checkbox ก่อน และการติ๊กยืนยันไม่ทำให้เกิดการแทรกเอง ทุกกรณียังต้องกดปุ่มแทรกและกดโพสต์ด้วยตัวเอง

ผู้ใช้ต้องมี [Facebook Page access](https://www.facebook.com/help/289207354498410/) ที่เหมาะสม Extension ไม่ข้าม permission ของ Facebook

## Permissions และข้อมูลที่เข้าถึง

| Permission / host | ใช้ทำอะไร |
| --- | --- |
| `sidePanel` | เปิด workspace ด้านข้าง browser |
| `storage` | เก็บ Draft/settings ใน local storage และ Gemini key ใน session storage |
| `nativeMessaging` | ส่ง Facebook Brief และรับ Facebook/Growth Pack กับ companion ในเครื่อง |
| `https://*.facebook.com/*` | โหลด content script เพื่อแทรก plain text หลังผู้ใช้โฟกัส editor |
| `https://generativelanguage.googleapis.com/*` | เรียก Gemini เมื่อผู้ใช้เลือกโหมด API โดยตรง |

Chrome อธิบายกลไกที่เกี่ยวข้องไว้ใน [Side Panel API](https://developer.chrome.com/docs/extensions/reference/api/sidePanel), [extension messaging](https://developer.chrome.com/docs/extensions/develop/concepts/messaging), [Storage API](https://developer.chrome.com/docs/extensions/reference/api/storage/) และ [Native Messaging](https://developer.chrome.com/docs/extensions/develop/concepts/native-messaging)

## Security guarantees และขอบเขต

- `manifest.json` เป็น Manifest V3 และ Content Security Policy อนุญาต script จากตัว extension เองเท่านั้น
- Service worker รับคำสั่งสำคัญจาก trusted extension context และตรวจ schema/ขนาดของ Facebook Brief, Content Pack และ typed Growth Pack snapshot
- Native Host ตรวจ exact extension origin; wildcard origin ไม่ได้รับอนุญาต
- Facebook Pack ที่ revision ไม่ตรง Brief ปัจจุบันถูกระบุเป็น stale และ side panel จะไม่แสดงเป็นผลพร้อมใช้ ส่วน Growth Pack ที่ stale/rejected ยังอ่านหรือคัดลอกได้ แต่ guard ปิดการแทรก
- การแทรกจำกัด 50,000 ตัวอักษร ปฏิเสธ NUL และสร้าง text node/insert text เท่านั้น
- ไม่มีการอ่าน Facebook DOM content, cookies, tokens, inbox หรือข้อมูลลูกค้า
- ไม่มี `.click()` หรือ selector สำหรับ Post/Send/Schedule ใน content script
- โปรแกรมไม่ทำ automated data collection; ดูนโยบายช่วยเหลือของ Meta เรื่อง [automated data collection](https://www.facebook.com/help/463983701520800)
- การที่ profile, Page, post หรือ comment เป็นสาธารณะไม่ใช่ consent ให้เก็บข้อมูลรายบุคคลหรือเพิ่มบุคคลนั้นเข้า audience โปรแกรมไม่เก็บ followers, reactions, comments, profiles, inbox หรือลูกค้าของเพจอื่น
- แผน retargeting/Lookalike/Advantage+ ใช้ได้เฉพาะ first-party/authorized sources ที่เจ้าของเพจมีสิทธิ์และอนุมัติ เช่น engagement ของเพจตน, website/app activity, lead forms และ customer lists ที่มีฐานกฎหมายกับทาง opt-out; extension ไม่อัปโหลด audience

สิ่งที่ผู้ใช้ยังต้องทำเองคือยืนยันข้อเท็จจริง สิทธิ์ใช้สื่อ ราคา เงื่อนไขโปรโมชัน การปฏิบัติตามกฎหมาย/นโยบาย และการกดเผยแพร่

## ตำแหน่งข้อมูล

Companion store:

```text
%AppData%\ContentBlueprint\FacebookCompanion\facebook-brief.json
%AppData%\ContentBlueprint\FacebookCompanion\facebook-pack.json
%AppData%\ContentBlueprint\GrowthWorkbench\growth-brief.json
%AppData%\ContentBlueprint\GrowthWorkbench\growth-pack.json
```

Native Host manifest:

```text
%LocalAppData%\ContentBlueprint\NativeMessaging\com.contentblueprint.facebook.json
```

Draft ของ side panel อยู่ใน `chrome.storage.local`; Gemini key อยู่ใน `chrome.storage.session` ข้อมูลเหล่านี้ไม่ได้เข้ารหัสโดย extension อย่าใส่ password, Facebook cookie/access token หรือข้อมูลส่วนบุคคลที่ไม่จำเป็น

## ถอน Native Host

```powershell
.\facebook-extension\native-host\uninstall-native-host.ps1
```

Script ลบเฉพาะ registry entries และ host manifest ที่ติดตั้งไว้ ไม่ลบ companion executable, Brief หรือ Content Pack จากนั้นลบ extension จาก `chrome://extensions`/`brave://extensions` ได้ตามปกติ

## ทดสอบหลังแก้ extension

```powershell
Set-Location C:\path\to\content-blueprint\facebook-extension
npm ci
npm test
node --check service-worker.js
node --check sidepanel.js
node --check content-script.js
node --check src\core.js
node --check src\growth.js
npm run test:e2e
```

ตรวจ Go boundary ที่ extension ใช้ร่วมด้วย:

```powershell
Set-Location C:\path\to\content-blueprint
go test ./internal/facebookcompanion ./internal/companionmcp ./internal/workbench ./internal/cliprovider
```

`npm run test:e2e` เปิด Chrome และ Brave ที่ติดตั้งจริงด้วย profile ชั่วคราว ทดสอบ Native Messaging, Sync/Fetch, Facebook stale guard, การแสดง finite Growth blocks, Growth stale/rejected/needs_review guards และการแทรกข้อความธรรมดากับ HTTPS Facebook editor fixture ในเครื่อง โดยไม่ใช้บัญชี Facebook และยืนยันว่าไม่มีการกด Post

ก่อนใช้งานกับเพจจริงควรตรวจซ้ำด้วยตนเอง:

1. Companion status เป็นพร้อมใช้งาน
2. Sync Brief แล้วได้ revision
3. รับ Pack ที่ revision ตรงกันได้
4. Pack เก่าถูกแจ้ง stale หลังแก้ Brief
5. Insert ล้มเหลวเมื่อยังไม่โฟกัส editor
6. Insert สำเร็จหลังโฟกัส editor และไม่มีการกด Post
7. Growth Pack ที่ stale/rejected แทรกไม่ได้ แต่ยังอ่านและคัดลอกได้
8. Growth Pack ที่ `needs_review` แทรกไม่ได้จนกว่าจะติ๊กยืนยัน และการติ๊กไม่แทรกให้อัตโนมัติ

## แก้ปัญหา

### ไม่พบ Content Blueprint Companion

- ตรวจว่า `build\bin\content-blueprint-companion.exe` มีอยู่
- รัน `install-native-host.ps1` ใหม่ โดยเฉพาะหลังย้าย binary
- ตรวจ extension ID ว่าตรง `ppncejmpiekmkepaeccdnpnpgdcfafje`
- Reload extension และเปิด side panel ใหม่

### ผลล่าสุดเป็น stale

Brief เปลี่ยนหลังผลนั้นถูกสร้าง ให้ Claude/Codex อ่าน Brief ล่าสุดและสร้างใหม่ อย่าแก้ revision ด้วยมือ เพราะ guard นี้มีไว้กันการใช้ร่างผิดโจทย์

### แทรกข้อความไม่ได้

เปิด `facebook.com`, คลิกใน editor ด้วยตัวเอง แล้วลองอีกครั้ง Content script ไม่เลือก editor อัตโนมัติและไม่ทำงานบนเว็บอื่น

### Wails ไม่พบ Claude/Codex

ตรวจจาก PowerShell แล้วเปิด Wails ใหม่หลังแก้ `PATH` หรือสถานะ login:

```powershell
codex --version
codex login status
claude --version
claude auth status --json
```

คำสั่งและโหมด headless อาจเปลี่ยนตามรุ่น CLI ควรตรวจเอกสาร [Codex CLI commands](https://learn.chatgpt.com/docs/developer-commands?surface=cli) และ [Claude Code headless mode](https://code.claude.com/docs/en/headless) เมื่ออัปเกรด

## ข้อจำกัดด้านโควตาและข้อกำหนด

- Wails/MCP ไม่ขายหรือแถม model usage; งานยังนับตามแผน โควตา rate limit และข้อกำหนดของบัญชี Claude/Codex
- Facebook AI Team ใช้สาม model invocations และ Quick draft ใช้หนึ่งครั้ง ส่วน Growth ใช้สาม/หนึ่งครั้งเป็น baseline แต่ GrowthPack stage อาจเรียก semantic repair เพิ่ม stage ละไม่เกินหนึ่งครั้ง
- Direct CLI workflow ปิดเครื่องมือเข้าถึงเครื่องและ browser/web จึงไม่เปิด URL หลักฐานเอง ให้ใส่ข้อเท็จจริงที่ตรวจแล้วใน evidence notes
- Gemini direct mode ใช้ API key และโควตาของบัญชี Gemini แยกจาก subscription ของ Claude/Codex
- Facebook UI และ policy เปลี่ยนได้ การแทรกข้อความอาจต้องปรับในอนาคต และไม่รับประกัน reach, conversion หรือการอนุมัติโฆษณา

## เอกสารทางการ

- [Chrome Manifest V3](https://developer.chrome.com/docs/extensions/develop/migrate/what-is-mv3)
- [Chrome Native Messaging](https://developer.chrome.com/docs/extensions/develop/concepts/native-messaging)
- [Chrome Side Panel API](https://developer.chrome.com/docs/extensions/reference/api/sidePanel)
- [Chrome Storage API](https://developer.chrome.com/docs/extensions/reference/api/storage/)
- [Chrome extension messaging](https://developer.chrome.com/docs/extensions/develop/concepts/messaging)
- [Using Chrome extensions in Brave](https://brave.com/learn/using-chrome-extensions-in-brave/)
