@echo off
set PORT=8080

:: Parse APP_PORT from .env if it exists
if exist .env (
    for /f "usebackq tokens=1,2 delims==" %%i in (".env") do (
        if "%%i"=="APP_PORT" set PORT=%%j
    )
)

echo Stopping NovaFlow Go Server on port %PORT%...

set FOUND=0
for /f "tokens=5" %%a in ('netstat -aon ^| findstr :%PORT% ^| findstr LISTENING') do (
    taskkill /f /pid %%a
    echo Server process PID %%a stopped successfully.
    set FOUND=1
)

if %FOUND%==0 (
    echo No active NovaFlow server process found running on port %PORT%.
)

pause
