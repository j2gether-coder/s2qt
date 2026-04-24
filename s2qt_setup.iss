#define MyAppName "S2QT"
#define MyAppVersion "1.0.0"
#define MyAppPublisher "S2QT"
#define MyAppExeName "s2qt.exe"
#define MyAppIconName "s2qt.ico"
#define SourceRoot "C:\Users\COADMIN\Documents\s2qt"

[Setup]
AppId={{8F4D8B58-7A73-4B84-9E7B-2F7A6E9A1001}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
DefaultDirName={userdocs}\S2QT
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes
LicenseFile={#SourceRoot}\var\doc\license.md
OutputDir={#SourceRoot}\_installer
OutputBaseFilename=S2QT_Setup_{#MyAppVersion}
SetupIconFile={#SourceRoot}\bin\{#MyAppIconName}
UninstallDisplayIcon={app}\bin\{#MyAppExeName}
Compression=lzma
SolidCompression=yes
WizardStyle=modern
PrivilegesRequired=admin

[Languages]
Name: "korean"; MessagesFile: "compiler:Languages\Korean.isl"

[Tasks]
Name: "desktopicon"; Description: "바탕화면 바로가기 생성"; GroupDescription: "추가 작업:"; Flags: unchecked
Name: "quicklaunchicon"; Description: "빠른 실행 바로가기 생성"; GroupDescription: "추가 작업:"; Flags: unchecked; OnlyBelowVersion: 6.1

[Dirs]
Name: "{app}\var\data"
Name: "{app}\var\db"
Name: "{app}\var\log"
Name: "{app}\var\temp"

; 설치 경로가 보호된 경로일 때를 대비해 runtime 폴더에 쓰기 권한 부여
Name: "{app}\var"; Permissions: users-modify

[Files]
; --- bin ---
Source: "{#SourceRoot}\bin\ggml.dll"; DestDir: "{app}\bin"; Flags: ignoreversion
Source: "{#SourceRoot}\bin\ggml-base.dll"; DestDir: "{app}\bin"; Flags: ignoreversion
Source: "{#SourceRoot}\bin\ggml-cpu.dll"; DestDir: "{app}\bin"; Flags: ignoreversion
Source: "{#SourceRoot}\bin\SDL2.dll"; DestDir: "{app}\bin"; Flags: ignoreversion
Source: "{#SourceRoot}\bin\whisper.dll"; DestDir: "{app}\bin"; Flags: ignoreversion
Source: "{#SourceRoot}\bin\whisper-cli.exe"; DestDir: "{app}\bin"; Flags: ignoreversion
Source: "{#SourceRoot}\bin\{#MyAppExeName}"; DestDir: "{app}\bin"; Flags: ignoreversion
Source: "{#SourceRoot}\bin\{#MyAppIconName}"; DestDir: "{app}\bin"; Flags: ignoreversion

; --- var/conf ---
; 사용자 설정/커스텀 보존 목적이면 onlyifdoesntexist 유지
Source: "{#SourceRoot}\var\conf\app.yaml"; DestDir: "{app}\var\conf"; Flags: ignoreversion onlyifdoesntexist
Source: "{#SourceRoot}\var\conf\footer.json"; DestDir: "{app}\var\conf"; Flags: ignoreversion onlyifdoesntexist
Source: "{#SourceRoot}\var\conf\prompt_qt_json.md"; DestDir: "{app}\var\conf"; Flags: ignoreversion onlyifdoesntexist
Source: "{#SourceRoot}\var\conf\security.json"; DestDir: "{app}\var\conf"; Flags: ignoreversion onlyifdoesntexist
Source: "{#SourceRoot}\var\conf\style_qt_html.css"; DestDir: "{app}\var\conf"; Flags: ignoreversion onlyifdoesntexist
Source: "{#SourceRoot}\var\conf\style_qt_pdf.css"; DestDir: "{app}\var\conf"; Flags: ignoreversion onlyifdoesntexist
Source: "{#SourceRoot}\var\conf\util_ver.json"; DestDir: "{app}\var\conf"; Flags: ignoreversion onlyifdoesntexist

; --- var/doc ---
Source: "{#SourceRoot}\var\doc\license.md"; DestDir: "{app}\var\doc"; Flags: ignoreversion
Source: "{#SourceRoot}\var\doc\template_guide.md"; DestDir: "{app}\var\doc"; Flags: ignoreversion
Source: "{#SourceRoot}\var\doc\user_guide.md"; DestDir: "{app}\var\doc"; Flags: ignoreversion

; --- var/image ---
Source: "{#SourceRoot}\var\image\s2qt_link.png"; DestDir: "{app}\var\image"; Flags: ignoreversion

; --- var/model ---
Source: "{#SourceRoot}\var\model\ggml-tiny.bin"; DestDir: "{app}\var\model"; Flags: ignoreversion

; --- var/template ---
Source: "{#SourceRoot}\var\template\*"; DestDir: "{app}\var\template"; Flags: ignoreversion recursesubdirs createallsubdirs

[Icons]
Name: "{autoprograms}\{#MyAppName}"; Filename: "{app}\bin\{#MyAppExeName}"; WorkingDir: "{app}\bin"; IconFilename: "{app}\bin\{#MyAppIconName}"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\bin\{#MyAppExeName}"; WorkingDir: "{app}\bin"; IconFilename: "{app}\bin\{#MyAppIconName}"; Tasks: desktopicon
Name: "{userappdata}\Microsoft\Internet Explorer\Quick Launch\{#MyAppName}"; Filename: "{app}\bin\{#MyAppExeName}"; WorkingDir: "{app}\bin"; IconFilename: "{app}\bin\{#MyAppIconName}"; Tasks: quicklaunchicon

[Run]
Filename: "{app}\bin\{#MyAppExeName}"; Description: "{#MyAppName} 실행"; Flags: nowait postinstall skipifsilent

[UninstallDelete]
; runtime 생성 파일/로그 정리
Type: filesandordirs; Name: "{app}\var\log"
Type: filesandordirs; Name: "{app}\var\temp"