@echo off
chcp 65001 >nul
setlocal

cd /d "%~dp0" || exit /b 1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0build-release.ps1"
exit /b %ERRORLEVEL%
