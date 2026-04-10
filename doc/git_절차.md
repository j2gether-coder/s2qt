## 📤 GitHub에 올릴 때 (Push)
1. **상태 확인**
   ```bash
   git status
   ```
   → 어떤 파일이 변경되었는지 확인

2. **변경 파일 스테이징**
   ```bash
   git add <파일명>
   git add .   # 모든 변경 파일 추가
   ```

3. **커밋 생성**
   ```bash
   git commit -m "작업 내용 설명"
   ```

4. **원격 저장소 연결 (최초 1회만)**
   ```bash
   git remote add origin https://github.com/<사용자명>/<저장소명>.
   git remote add origin git@github.com:j2gether-coder/s2qt.git
   git
   ```

5. **푸시 (업로드)**
   ```bash
   git push -u origin main
   ```
   → 이후에는 `git push`만 입력해도 됩니다.  
   ⚠️ 여기서 **비밀번호 대신 Personal Access Token**을 입력해야 합니다.

---

## 📥 GitHub에서 내릴 때 (Pull)
1. **원격 저장소 최신 내용 가져오기**
   ```bash
   git pull origin main
   ```
   → GitHub에 있는 최신 커밋을 로컬 저장소로 가져옵니다.

2. **저장소 복제 (처음 받을 때만)**
   ```bash
   git clone https://github.com/<사용자명>/<저장소명>.git
   ```
   → 처음 프로젝트를 내려받을 때 사용합니다.

---

## 🔑 핵심 요약
- **올릴 때(push)**: `git add → git commit → git push`  
- **내릴 때(pull)**: `git pull` (처음이면 `git clone`)  
- 인증은 **GitHub 비밀번호 대신 Personal Access Token**을 사용해야 함  

---
