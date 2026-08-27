## ปัญหาและผลหลังแก้

<!-- อธิบาย user-visible behavior แบบสั้นและตรวจได้ -->

## Trust boundaries ที่แตะ

- [ ] Brief / Pack schema หรือ revision
- [ ] Provider CLI / process / environment
- [ ] MCP / Native Messaging / extension
- [ ] Local data / privacy
- [ ] Installer / updater / release
- [ ] ไม่แตะ boundary ข้างต้น

## การตรวจ

<!-- ระบุคำสั่งและผลจริง; อย่าเขียนเพียงว่า tests pass -->

- [ ] Focused tests
- [ ] `go test ./...`
- [ ] `go vet ./...`
- [ ] `frontend`: `npm run build`
- [ ] `facebook-extension`: `npm test`
- [ ] Browser/UI smoke เมื่อเกี่ยวข้อง
- [ ] Windows release build เมื่อเกี่ยวข้อง

## ภาพและคู่มือ

- [ ] แนบภาพก่อน/หลังสำหรับ UI change หรือระบุว่าไม่เกี่ยวข้อง
- [ ] อัปเดต README/docs เมื่อ behavior เปลี่ยน หรือระบุว่าไม่จำเป็น
- [ ] ใช้ข้อมูลสังเคราะห์และลบ secret, customer data, username และ local path ออกจากไฟล์แนบ

## Human approval

- [ ] การเปลี่ยนแปลงนี้ไม่ทำให้ AI โพสต์ ส่งข้อความ สร้าง audience รับรองยอดขาย หรือเปิด installer โดยไม่มีผู้ใช้ยืนยัน
