@echo off
set PORT=8080

:: Parse APP_PORT from .env if it exists
if exist .env (
    for /f "usebackq tokens=1,2 delims==" %%i in (".env") do (
        if "%%i"=="APP_PORT" set PORT=%%j
    )
)

echo Starting NovaFlow Go Server on port %PORT%...
echo Open http://localhost:%PORT% in your browser.
echo You can stop the server by running stop.bat or pressing Ctrl+C in this terminal.
go run main.go
pause
