export type OnboardingMode = 'facebook' | 'growth' | 'seo'

export type GrowthTourTab = 'playbooks' | 'leads' | 'utm' | 'experiments'

export interface TourDestination {
  mode: OnboardingMode
  growthTab?: GrowthTourTab
  playbookId?: string
  seoView?: 'brief' | 'prompt' | 'editor'
}

export interface MissionStepDefinition {
  id: string
  target: string
  title: string
  body: string
  destination: TourDestination
  placement?: 'top' | 'top-start' | 'bottom' | 'bottom-start' | 'left' | 'right' | 'center'
  blockTargetInteraction?: boolean
}

export interface OnboardingMission {
  id: string
  title: string
  description: string
  duration: string
  steps: MissionStepDefinition[]
}

export const ONBOARDING_VERSION = 3

export const ONBOARDING_MISSIONS: OnboardingMission[] = [
  {
    id: 'suite-overview',
    title: 'รู้จักพื้นที่ทำงาน',
    description: 'แยกงานโพสต์ งานขาย และบทความ SEO ให้เลือกได้ถูกที่',
    duration: 'ประมาณ 2 นาที',
    steps: [
      {
        id: 'workspace-switcher',
        target: '[data-tour="suite-mode-switcher"]',
        title: 'สามพื้นที่ งานคนละแบบ',
        body: 'Facebook ใช้ทำ Content Pack, Growth Hub ใช้วางแผนขายและวัดผล, SEO Blueprint ใช้เขียนบทความจากหลักฐาน งานแต่ละส่วนเก็บแยกกันเพื่อลดความสับสน',
        destination: {mode: 'growth'},
        placement: 'bottom',
        blockTargetInteraction: true,
      },
      {
        id: 'growth-home',
        target: '[data-tour="growth-shell"]',
        title: 'เริ่มงานขายที่ Growth Hub',
        body: 'เลือก Playbook ตามงานที่ต้องทำ กรอกเฉพาะข้อมูลจริง แล้วให้ Claude หรือ Codex ช่วยร่าง ผลลัพธ์ยังต้องผ่านคนตรวจก่อนนำไปใช้',
        destination: {mode: 'growth', growthTab: 'playbooks'},
        placement: 'center',
      },
      {
        id: 'ai-office',
        target: '[data-tour="growth-ai-studio"]',
        title: 'ดูทีม AI เดินงานในห้องเดียวกัน',
        body: 'ตัวละคร Strategist, Producer และ Reviewer จะเดินไปยังโต๊ะตามช่วงงาน สีและบันทึกกิจกรรมบอกว่ากำลังรอ ทำเสร็จ หรือมีปัญหา ภาพนี้แสดงสถานะ CLI เท่านั้น AI ไม่มีสิทธิ์โพสต์ ส่งข้อความ หรือยืนยันยอดขายแทนคุณ',
        destination: {mode: 'growth', growthTab: 'playbooks'},
        placement: 'bottom',
        blockTargetInteraction: true,
      },
    ],
  },
  {
    id: 'facebook-content',
    title: 'ทำ Facebook Content Pack',
    description: 'จาก brief และหลักฐาน ไปจนถึงโพสต์ รีล คารูเซล และคลังคำตอบ',
    duration: 'ประมาณ 4 นาที',
    steps: [
      {
        id: 'facebook-brief',
        target: '[data-tour="facebook-brief"]',
        title: 'บอกงานให้ชัดก่อน',
        body: 'ใส่สินค้า กลุ่มลูกค้า และผลลัพธ์ที่ต้องการให้ครบ ข้อมูลสามส่วนนี้เป็นขอบเขตที่ AI ใช้ร่างเนื้อหา จึงควรเขียนให้เฉพาะเจาะจงกว่าคำว่า “ช่วยขายของ”',
        destination: {mode: 'facebook'},
        placement: 'right',
      },
      {
        id: 'facebook-evidence',
        target: '[data-tour="facebook-evidence"]',
        title: 'แยกข้อเท็จจริงออกจากคำขาย',
        body: 'เพิ่มราคา เงื่อนไข สเปก รีวิวที่ได้รับอนุญาต หรือลิงก์อ้างอิงเป็นหลักฐาน AI ต้องผูกข้อกล่าวอ้างกับข้อมูลส่วนนี้และแจ้งเมื่อข้อมูลไม่พอ',
        destination: {mode: 'facebook'},
        placement: 'right',
      },
      {
        id: 'facebook-provider',
        target: '[data-tour="facebook-provider"]',
        title: 'ใช้บัญชี CLI ที่มีอยู่',
        body: 'เลือก Claude CLI หรือ Codex CLI เมื่อต้องการให้แอปสั่ง worker ในเครื่อง หรือเลือก MCP เมื่อจะส่งงานให้ agent ภายนอกมารับและบันทึกผลกลับมา',
        destination: {mode: 'facebook'},
        placement: 'bottom',
        blockTargetInteraction: true,
      },
      {
        id: 'facebook-workflow',
        target: '[data-tour="facebook-workflow"]',
        title: 'เลือกความละเอียดของงาน',
        body: 'Quick draft เหมาะกับงานประจำที่แก้เองได้เร็ว ส่วน AI Team เหมาะกับแคมเปญหรือโพสต์ที่ต้องตรวจมุมขายและความเสี่ยงเพิ่ม',
        destination: {mode: 'facebook'},
        placement: 'bottom',
        blockTargetInteraction: true,
      },
      {
        id: 'facebook-run',
        target: '[data-tour="facebook-run"]',
        title: 'สร้างเมื่อ brief พร้อม',
        body: 'กดสร้างแล้วดูสถานะ worker ในห้องทำงาน AI ระหว่างรอ แอปไม่ควบคุม Facebook และไม่โพสต์อะไรเอง',
        destination: {mode: 'facebook'},
        placement: 'top',
        blockTargetInteraction: true,
      },
      {
        id: 'facebook-output',
        target: '[data-tour="facebook-output"]',
        title: 'ตรวจแล้วค่อยคัดลอก',
        body: 'อ่านโพสต์ CTA และ compliance notes ก่อน ใช้ปุ่มคัดลอกเฉพาะชิ้นที่ต้องการ แล้วแก้ให้ตรงน้ำเสียงอาจารย์ก่อนนำไปลงจริง',
        destination: {mode: 'facebook'},
        placement: 'top',
      },
    ],
  },
  {
    id: 'competitor-ads',
    title: 'ศึกษาคู่แข่งโดยไม่ดูดฐานลูกค้า',
    description: 'เปลี่ยนสิ่งที่เห็นใน Meta Ads Library เป็น creative gap และแผน audience ที่มีสิทธิ์ใช้',
    duration: 'ประมาณ 4 นาที',
    steps: [
      {
        id: 'audience-safety',
        target: '[data-tour="growth-audience-safety"]',
        title: 'เส้นแบ่งที่ต้องรู้ก่อน',
        body: 'ใช้โฆษณาสาธารณะเพื่อศึกษามุมสื่อสารได้ แต่ไม่ดึงรายชื่อผู้ติดตาม คนคอมเมนต์ inbox หรือลูกค้าของเพจอื่นมาเป็นฐานยิงแอด ระบบนี้จึงรับเฉพาะบันทึกจาก Ads Library และข้อมูลที่เพจมีสิทธิ์ใช้',
        destination: {mode: 'growth', growthTab: 'playbooks', playbookId: 'facebook-campaign'},
        placement: 'bottom',
      },
      {
        id: 'campaign-playbook',
        target: '[data-tour="growth-playbook-catalog"]',
        title: 'เลือก Facebook Campaign',
        body: 'Playbook นี้รวมงานที่มักทำซ้ำ ได้แก่ สรุปข้อเสนอ แตกมุมโฆษณา วางลำดับคอนเทนต์ และเลือกแหล่ง audience ที่ตรวจสิทธิ์ได้',
        destination: {mode: 'growth', growthTab: 'playbooks', playbookId: 'facebook-campaign'},
        placement: 'right',
        blockTargetInteraction: true,
      },
      {
        id: 'competitor-notes',
        target: '[data-tour="growth-playbook-form"]',
        title: 'จดรูปแบบ ไม่จดรายชื่อคน',
        body: 'วาง headline, offer, CTA, format และสิ่งที่โฆษณาคู่แข่งยังตอบไม่ชัด จากนั้นระบุสินทรัพย์ที่คุณมีสิทธิ์ เช่น คนดูวิดีโอ คนมีส่วนร่วมกับเพจ เว็บไซต์ หรือลีดที่ยินยอม',
        destination: {mode: 'growth', growthTab: 'playbooks', playbookId: 'facebook-campaign'},
        placement: 'right',
      },
      {
        id: 'audience-plan-output',
        target: '[data-tour="growth-output"]',
        title: 'รับแผนที่ย้อนตรวจได้',
        body: 'ผลลัพธ์ควรแยก creative gap, สมมติฐาน, กลุ่ม retargeting, กลุ่ม lookalike และสิ่งที่ต้องขอเจ้าของเพจยืนยัน หากที่มาของข้อมูลไม่ชัด ให้หยุดที่คำถามแทนการเดา',
        destination: {mode: 'growth', growthTab: 'playbooks', playbookId: 'facebook-campaign'},
        placement: 'left',
      },
      {
        id: 'campaign-review',
        target: '[data-tour="growth-output"]',
        title: 'คนเป็นผู้อนุมัติ',
        body: 'ตรวจสิทธิ์ของ audience, ข้อเสนอ, งบ และข้อความก่อนกดรับรอง การรับรองในแอปเป็นบันทึกการตรวจ ไม่ได้ส่งข้อมูลหรือสร้างแคมเปญใน Meta Ads Manager',
        destination: {mode: 'growth', growthTab: 'playbooks', playbookId: 'facebook-campaign'},
        placement: 'top',
        blockTargetInteraction: true,
      },
    ],
  },
  {
    id: 'lead-commission',
    title: 'ตามลีดและค่าคอม',
    description: 'บันทึกสถานะขาย แยกยอดคาดการณ์จากยอดที่เจ้าของเพจยืนยัน',
    duration: 'ประมาณ 3 นาที',
    steps: [
      {
        id: 'lead-board',
        target: '[data-tour="growth-leads"]',
        title: 'หนึ่งลีด หนึ่งสถานะล่าสุด',
        body: 'บันทึกชื่ออ้างอิง ช่องทาง สินค้าที่สนใจ และขั้นตอนถัดไปเท่าที่จำเป็น หลีกเลี่ยงการคัดลอกบทสนทนาหรือข้อมูลส่วนตัวเกินงานขาย',
        destination: {mode: 'growth', growthTab: 'leads'},
        placement: 'top',
      },
      {
        id: 'commission-ledger',
        target: '[data-tour="growth-commission"]',
        title: 'ยอดประมาณการไม่ใช่ยอดยืนยัน',
        body: 'แอปคำนวณค่าคอมจากยอดและอัตราที่กำหนด แต่จะแสดงเป็นประมาณการจนกว่าเจ้าของเพจจะยืนยัน เพื่อไม่ให้รายงานรายได้จากข้อมูลที่ยังไม่จบ',
        destination: {mode: 'growth', growthTab: 'leads'},
        placement: 'top',
      },
      {
        id: 'lead-review-cycle',
        target: '[data-tour="growth-tabs"]',
        title: 'ใช้เป็นรอบงานประจำวัน',
        body: 'เปิดดูรายการที่ต้องติดตาม อัปเดตเฉพาะสถานะที่เปลี่ยน และทบทวนยอดกับเจ้าของเพจเป็นรอบ วิธีนี้ลดการไล่ค้นแชตซ้ำโดยไม่ต้องให้บอตตอบลูกค้าเอง',
        destination: {mode: 'growth', growthTab: 'leads'},
        placement: 'bottom',
        blockTargetInteraction: true,
      },
    ],
  },
  {
    id: 'measurement',
    title: 'ทำ UTM และบันทึกการทดลอง',
    description: 'ตั้งชื่อลิงก์ให้สม่ำเสมอ แล้วแยกผลที่วัดได้ออกจากข้อสรุป',
    duration: 'ประมาณ 3 นาที',
    steps: [
      {
        id: 'utm-builder',
        target: '[data-tour="growth-utm"]',
        title: 'สร้างลิงก์จากกติกาเดียวกัน',
        body: 'ใส่ URL ต้นทาง source, medium และ campaign ให้เป็นชื่อที่ทีมอ่านรู้เรื่อง แอปจะจัดการ encoding และเก็บพารามิเตอร์เดิมของ URL ให้',
        destination: {mode: 'growth', growthTab: 'utm'},
        placement: 'top',
      },
      {
        id: 'experiment-log',
        target: '[data-tour="growth-experiment"]',
        title: 'กำหนดสิ่งที่จะเปลี่ยนก่อนยิง',
        body: 'เขียนสมมติฐาน ตัวแปรที่เปลี่ยน ตัวชี้วัด และช่วงเวลาให้ครบก่อนเริ่ม เพื่อป้องกันการเลือกคำอธิบายหลังเห็นผลแล้ว',
        destination: {mode: 'growth', growthTab: 'experiments'},
        placement: 'top',
      },
      {
        id: 'experiment-reading',
        target: '[data-tour="growth-experiment"]',
        title: 'บันทึกสิ่งที่เห็น ไม่เติมเหตุผลเอง',
        body: 'กรอกยอดแสดงผล คลิก ลีด หรือยอดขายตามที่มีจริง ถ้าทดลองหลายตัวแปรพร้อมกันหรือข้อมูลน้อย ระบบจะระบุว่าเป็นทิศทาง ไม่ใช่เหตุและผลที่ยืนยันแล้ว',
        destination: {mode: 'growth', growthTab: 'experiments'},
        placement: 'top',
      },
    ],
  },
  {
    id: 'growth-seo',
    title: 'วางแผน Google SEO',
    description: 'ทำ topic map, content brief, on-page review และงานจาก Search Console',
    duration: 'ประมาณ 4 นาที',
    steps: [
      {
        id: 'seo-playbooks',
        target: '[data-tour="growth-playbook-catalog"]',
        title: 'เลือกงาน SEO ตามโจทย์',
        body: 'เริ่มจาก Topic Map เมื่อต้องวางภาพรวม ใช้ Content Brief ก่อนเขียน ใช้ On-page Review กับหน้าที่มีอยู่ และใช้ Search Console Opportunities กับไฟล์ CSV ที่คุณนำเข้าเอง',
        destination: {mode: 'growth', growthTab: 'playbooks', playbookId: 'seo-topic-map'},
        placement: 'right',
        blockTargetInteraction: true,
      },
      {
        id: 'seo-inputs',
        target: '[data-tour="growth-playbook-form"]',
        title: 'ให้ข้อมูลธุรกิจก่อน keyword',
        body: 'ระบุสินค้า ลูกค้า พื้นที่ให้บริการ หน้าเว็บที่มี และหลักฐานความเชี่ยวชาญ แล้วจึงใส่หัวข้อหรือข้อมูล Search Console วิธีนี้ช่วยให้แผนผูกกับธุรกิจจริง ไม่ใช่ลิสต์คำค้นทั่วไป',
        destination: {mode: 'growth', growthTab: 'playbooks', playbookId: 'seo-topic-map'},
        placement: 'right',
      },
      {
        id: 'seo-output',
        target: '[data-tour="growth-output"]',
        title: 'ตรวจหลักฐานและคำถามค้าง',
        body: 'ดู basis ของแต่ละส่วนว่าเป็นข้อมูลจากคุณ หลักฐาน ตัวเลขที่นำเข้า หรือข้อเสนอของ AI แล้วตอบ open questions ก่อนนำแผนไปเขียนหรือแก้หน้าเว็บ',
        destination: {mode: 'growth', growthTab: 'playbooks', playbookId: 'seo-topic-map'},
        placement: 'left',
      },
      {
        id: 'seo-review',
        target: '[data-tour="growth-output"]',
        title: 'อนุมัติงานที่พร้อมเท่านั้น',
        body: 'เช็ก search intent, ความตรงกับสินค้า, แหล่งอ้างอิง และงานถัดไปก่อนรับรอง ระบบไม่สร้างหน้าจำนวนมาก ไม่แก้อันดับ และไม่เผยแพร่เว็บแทนคุณ',
        destination: {mode: 'growth', growthTab: 'playbooks', playbookId: 'seo-topic-map'},
        placement: 'top',
        blockTargetInteraction: true,
      },
    ],
  },
  {
    id: 'seo-article',
    title: 'เขียนบทความ SEO ฉบับเต็ม',
    description: 'ใช้พื้นที่ SEO Blueprint เดิมตั้งแต่ brief ถึงตรวจคุณภาพและส่งออก',
    duration: 'ประมาณ 4 นาที',
    steps: [
      {
        id: 'seo-brief',
        target: '[data-tour="seo-brief"]',
        title: 'เริ่มจาก brief ที่ตรวจได้',
        body: 'ใส่ keyword, กลุ่มผู้อ่าน, เป้าหมาย และหลักฐาน เนื้อหาที่ไม่มีแหล่งรองรับควรถูกเขียนเป็นข้อเสนอหรือคำถาม ไม่ใช่ข้อเท็จจริง',
        destination: {mode: 'seo', seoView: 'brief'},
        placement: 'right',
      },
      {
        id: 'seo-workflow',
        target: '[data-tour="seo-workflow"]',
        title: 'เดินทีละขั้น',
        body: 'Brief เก็บข้อมูลต้นทาง Prompt ให้ตรวจคำสั่งก่อนส่ง และ Editor ใช้แก้ผลลัพธ์ คุณย้อนกลับมาแก้ brief ได้เสมอเมื่อพบว่าข้อมูลยังไม่พอ',
        destination: {mode: 'seo', seoView: 'brief'},
        placement: 'bottom',
        blockTargetInteraction: true,
      },
      {
        id: 'seo-generate',
        target: '[data-tour="seo-generate"]',
        title: 'สร้างเมื่อ API พร้อม',
        body: 'พื้นที่นี้ใช้ Gemini API ตามการตั้งค่าของโปรเจกต์ ตรวจ provider และ grounding ก่อนสร้าง เพราะค่าเหล่านี้มีผลต่อข้อมูลที่โมเดลได้รับ',
        destination: {mode: 'seo', seoView: 'brief'},
        placement: 'bottom',
        blockTargetInteraction: true,
      },
      {
        id: 'seo-quality',
        target: '[data-tour="seo-quality"]',
        title: 'คะแนนเป็นตัวช่วยตรวจ',
        body: 'ดู coverage ของแหล่งอ้างอิง ความครบของ metadata และคำเตือน คะแนนไม่รับประกันอันดับ จึงต้องอ่านบทความและตรวจความถูกต้องอีกครั้ง',
        destination: {mode: 'seo', seoView: 'editor'},
        placement: 'left',
      },
    ],
  },
  {
    id: 'mcp-browser-handoff',
    title: 'ส่งงานผ่าน MCP แล้วใช้ใน Chrome/Brave',
    description: 'ใช้ Claude หรือ Codex ที่ล็อกอินอยู่ โดยไม่ต้องซื้อ API เพิ่ม แล้วรับร่างกลับมาใช้แบบกดเอง',
    duration: 'ประมาณ 4 นาที',
    steps: [
      {
        id: 'mcp-provider',
        target: '[data-tour="growth-provider"]',
        title: 'เลือก MCP เมื่อจะสั่งจาก CLI ภายนอก',
        body: 'MCP ไม่ได้เปิด Claude หรือ Codex ให้เอง แอปจะบันทึก Brief ไว้ในเครื่อง แล้ว agent ที่คุณเปิดอยู่เรียก get_growth_brief เพื่อรับโจทย์และ contract ที่เชื่อถือได้',
        destination: {mode: 'growth', growthTab: 'playbooks', playbookId: 'offer-audience'},
        placement: 'bottom',
        blockTargetInteraction: true,
      },
      {
        id: 'mcp-sync',
        target: '[data-tour="growth-run"]',
        title: 'ซิงก์ Brief ก่อนสั่ง agent',
        body: 'กรอกช่องบังคับแล้วกด “ซิงก์ Brief” จากนั้นบอก Claude หรือ Codex ให้ใช้ Content Blueprint MCP: อ่าน brief ล่าสุด สร้าง Growth Pack ตาม schema และบันทึกด้วย save_growth_pack โดยใช้ revision เดิมทุกตัวอักษร',
        destination: {mode: 'growth', growthTab: 'playbooks', playbookId: 'offer-audience'},
        placement: 'top',
        blockTargetInteraction: true,
      },
      {
        id: 'mcp-ai-office',
        target: '[data-tour="growth-ai-studio"]',
        title: 'แยกภาพสถานะออกจากสิทธิ์ทำงาน',
        body: 'ห้องตัวละครแสดงขั้นตอนของ worker ที่แอปสั่งโดยตรง ส่วนรอบ MCP ที่คุณสั่งจากอีกหน้าต่างจะรับผลผ่านไฟล์ local จึงอาจไม่เห็นตัวละครเดินครบทุกโต๊ะ แต่ข้อห้ามเรื่อง scrape, post และ DM ยังเหมือนเดิม',
        destination: {mode: 'growth', growthTab: 'playbooks', playbookId: 'offer-audience'},
        placement: 'bottom',
      },
      {
        id: 'mcp-fetch',
        target: '[data-tour="growth-output"]',
        title: 'รับผลกลับมาและตรวจ revision',
        body: 'เมื่อ agent บันทึกเสร็จ กด “รับผลล่าสุด” แอปจะปฏิเสธ pack ที่รูปแบบผิด และติดป้าย stale ถ้า brief ถูกแก้หลังสร้าง คุณยังต้องอ่าน risk flags, คำถามค้าง และเหตุผลของแต่ละส่วนก่อนอนุมัติ',
        destination: {mode: 'growth', growthTab: 'playbooks', playbookId: 'offer-audience'},
        placement: 'left',
      },
      {
        id: 'browser-extension',
        target: '[data-tour="growth-output"]',
        title: 'Chrome/Brave รับเฉพาะร่างที่ผ่าน guard',
        body: 'เปิด side panel ของ extension แล้วกดรับ Growth Pack ล่าสุด Pack ที่ rejected หรือ stale แทรกไม่ได้ ส่วน needs_review ต้องติ๊กยืนยันก่อน Extension เติมข้อความลง composer ที่คุณโฟกัสเท่านั้น และไม่กด Post หรือส่งข้อความแทนคุณ',
        destination: {mode: 'growth', growthTab: 'playbooks', playbookId: 'offer-audience'},
        placement: 'left',
        blockTargetInteraction: true,
      },
    ],
  },
]

export function getMission(id: string): OnboardingMission | undefined {
  return ONBOARDING_MISSIONS.find((mission) => mission.id === id)
}
