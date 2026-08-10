@echo off
setlocal

cd /d "%~dp0" || exit /b 1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0build-release.ps1"
exit /b %ERRORLEVEL%
