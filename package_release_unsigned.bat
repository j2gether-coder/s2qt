@echo off
setlocal enabledelayedexpansion

REM ==================================================
REM S2QT Unsigned Release Package
REM - 인증서 없는 배포/테스트용
REM - Inno Setup 빌드는 다른 PC에서 수행
REM - 기존 산출물 복사 + SHA256 해시 생성
REM ==================================================

set "ROOT=%~dp0"
cd /d "%ROOT%"

set "RELEASE_DIR=%ROOT%release"
set "DOC_DIR=%ROOT%var\doc"

set "APP_EXE=%ROOT%build\bin\s2qt.exe"
set "SETUP_EXE=%ROOT%build\s2qt_setup.exe"

echo.
echo [1/4] Prepare folders...
if not exist "%RELEASE_DIR%" mkdir "%RELEASE_DIR%"
if not exist "%DOC_DIR%" mkdir "%DOC_DIR%"

echo.
echo [2/4] Check build artifacts...
if not exist "%APP_EXE%" (
  echo ERROR: s2qt.exe not found:
  echo %APP_EXE%
  exit /b 1
)

if not exist "%SETUP_EXE%" (
  echo ERROR: s2qt_setup.exe not found:
  echo %SETUP_EXE%
  echo.
  echo Inno Setup build is expected to be done on another PC.
  exit /b 1
)

echo.
echo [3/4] Copy release files...
copy /Y "%APP_EXE%" "%RELEASE_DIR%\s2qt.exe" >nul
if errorlevel 1 (
  echo ERROR: failed to copy s2qt.exe
  exit /b 1
)

copy /Y "%SETUP_EXE%" "%RELEASE_DIR%\s2qt_setup.exe" >nul
if errorlevel 1 (
  echo ERROR: failed to copy s2qt_setup.exe
  exit /b 1
)

echo.
echo [4/4] Generate SHA256 hashes...
powershell -NoProfile -ExecutionPolicy Bypass -Command ^
  "$files = @('%RELEASE_DIR%\s2qt.exe','%RELEASE_DIR%\s2qt_setup.exe');" ^
  "$out = '%RELEASE_DIR%\release_integrity.md';" ^
  "'# S2QT Release Integrity' | Out-File $out -Encoding utf8;" ^
  "'' | Out-File $out -Append -Encoding utf8;" ^
  "'## Files' | Out-File $out -Append -Encoding utf8;" ^
  "foreach($f in $files) { if(Test-Path $f) { $h = Get-FileHash $f -Algorithm SHA256; '- ' + (Split-Path $f -Leaf) + ' SHA256: ' + $h.Hash | Out-File $out -Append -Encoding utf8 } }" ^
  "'' | Out-File $out -Append -Encoding utf8;" ^
  "'## Signing Status' | Out-File $out -Append -Encoding utf8;" ^
  "'- Code signing: Not signed' | Out-File $out -Append -Encoding utf8;" ^
  "'- Build type: unsigned release package' | Out-File $out -Append -Encoding utf8;" ^
  "'' | Out-File $out -Append -Encoding utf8;" ^
  "'## Note' | Out-File $out -Append -Encoding utf8;" ^
  "'- Inno Setup installer is generated on a separate PC.' | Out-File $out -Append -Encoding utf8;"

if errorlevel 1 (
  echo ERROR: failed to generate release_integrity.md
  exit /b 1
)

copy /Y "%RELEASE_DIR%\release_integrity.md" "%DOC_DIR%\release_integrity.md" >nul

echo.
echo Done.
echo Release folder:
echo %RELEASE_DIR%
echo.
echo Generated:
echo - %RELEASE_DIR%\s2qt.exe
echo - %RELEASE_DIR%\s2qt_setup.exe
echo - %RELEASE_DIR%\release_integrity.md
echo.

endlocal