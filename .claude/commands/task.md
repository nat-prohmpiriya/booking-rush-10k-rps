Role: คุณคือ Engineering Manager / Tech Lead
Goal: นำ Technical Plan ด้านบน มาแตกเป็น "Development Tasks" ขนาดเล็กที่สามารถ Implement และ Test ได้จบในตัวเอง (Isolation)

Requirement:
- แต่ละ Task ต้องเล็กพอที่จะรีวิวและทดสอบได้ง่าย (Small, Reviewable Chunks)
- หลีกเลี่ยงงานชิ้นใหญ่ (เช่น "สร้างระบบ Login") แต่ให้แตกย่อย (เช่น "สร้าง DB Schema สำหรับ User", "สร้าง API รับค่า Registration", "สร้างหน้า UI กรอก Form")
- เรียงลำดับความสำคัญ (Dependency Order)

Task:
สร้างรายการ Tasks (To-do list) โดยแต่ละรายการต้องประกอบด้วย:
1. Task ID & Name: ชื่อที่สื่อความหมาย
2. Description: สิ่งที่ต้องทำโดยละเอียด
3. Technical Context: ไฟล์ไหนที่เกี่ยวข้อง หรือ Function ไหนที่ต้องสร้าง
4. Acceptance Criteria / Definition of Done: เกณฑ์ที่บอกว่างานนี้เสร็จสมบูรณ์และทดสอบผ่าน (เช่น "ยิง API แล้วได้ Response 200", "Validate Email Format ได้ถูกต้อง")

Format: ขอให้อยู่ในรูปแบบ Checklist หรือ Markdown Table เพื่อให้ฉันติ๊กเมื่องานเสร็จ