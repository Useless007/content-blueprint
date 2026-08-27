# Content Blueprint Growth Copilot

พื้นที่ทำงานนี้ช่วยแอดมินเปลี่ยนข้อเท็จจริงทางธุรกิจเป็นงานขายและงาน SEO ที่ตรวจทานได้ โดย AI มีหน้าที่เสนอร่างและคำอธิบาย ส่วนมนุษย์เป็นผู้อนุมัติการนำไปใช้ภายนอกเสมอ

## Language

**Workspace**:
พื้นที่ระดับบนที่รวมงานตามช่องทางหรือเป้าหมายเดียวกัน เช่น Facebook, SEO หรือ Growth Copilot
_Avoid_: Page, screen

**Playbook**:
กระบวนงาน AI ที่กำหนดผลลัพธ์ ข้อมูลที่ต้องใช้ และข้อควรระวังไว้ล่วงหน้าอย่างชัดเจน
_Avoid_: Tool, template, workflow

**Growth Brief**:
ชุดข้อมูลที่ผู้ใช้จัดทำเพื่อเรียก Playbook หนึ่งรายการ รวมบริบทธุรกิจ โจทย์ และ Evidence Notes
_Avoid_: Prompt, command

**Evidence Note**:
ข้อเท็จจริงที่ผู้ใช้ตรวจแล้วและอนุญาตให้นำไปใช้ URL หรือชื่อแหล่งข้อมูลเพียงอย่างเดียวไม่ถือเป็น Evidence Note
_Avoid_: Source URL, grounding

**Growth Pack**:
ผลลัพธ์ที่ผ่านการตรวจรูปแบบและผูกกับ revision ของ Growth Brief ที่ใช้สร้าง
_Avoid_: AI response, output blob

**Deliverable**:
ชิ้นงาน plain text หนึ่งรายการใน Growth Pack ที่ผู้ใช้สามารถตรวจ แก้ คัดลอก หรือส่งต่อได้
_Avoid_: Message, post

**AI Inference**:
ข้อเสนอหรือการจัดประเภทที่โมเดลอนุมานจากข้อมูล ไม่ใช่ข้อเท็จจริงจากแพลตฟอร์มหรือ Evidence Note
_Avoid_: Insight, Google data

**Observed Metric**:
ค่าที่ผู้ใช้กรอกหรือ import จากรายงาน/API ทางการและมีที่มาระบุไว้
_Avoid_: AI estimate

**Experiment**:
บันทึกสมมติฐาน ตัวแปร เวอร์ชัน ตัวชี้วัด และผลที่ผู้ใช้ยืนยัน โปรแกรมไม่เป็นผู้เริ่มหรือประกาศผู้ชนะโดยอัตโนมัติ
_Avoid_: Campaign result, A/B winner

**Lead Card**:
บันทึก local-only ที่ผู้ใช้สร้างเองเพื่อช่วยติดตามโอกาสขาย การส่งต่องาน และการติดตามผล
_Avoid_: Facebook profile, contact scrape

**Estimated Commission**:
ค่าคำนวณจากยอดขายและอัตราส่วนที่ผู้ใช้กรอก ซึ่งยังไม่ถือเป็นยอดที่เจ้าของงานรับรอง
_Avoid_: Commission due, confirmed commission

**Confirmed Commission**:
ยอดส่วนแบ่งที่มีการบันทึกว่าเจ้าของงานยืนยันแล้ว แยกออกจาก Estimated Commission เสมอ
_Avoid_: Estimate

**Human Approval**:
การตรวจและการกระทำอย่างชัดเจนโดยผู้ใช้ก่อนเผยแพร่ ส่งข้อความ เปลี่ยนข้อมูลภายนอก หรือรับรองผลทางธุรกิจ
_Avoid_: Auto approval

**Companion Inbox**:
พื้นที่รับส่งข้อมูลภายในเครื่องระหว่าง Wails, MCP และ Browser Extension โดยไม่ใช่ Facebook Inbox และไม่เผยแพร่ข้อมูลเอง
_Avoid_: Inbox, Messenger

**Stale Pack**:
Growth Pack ที่ revision ไม่ตรงกับ Growth Brief ปัจจุบัน จึงต้องไม่ถูกนำเสนอว่าเป็นผลพร้อมใช้
_Avoid_: Old result
