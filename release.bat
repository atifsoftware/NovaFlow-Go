@echo off
setlocal enabledelayedexpansion

echo =========================================================
echo       NovaFlow-Go Enterprise Release Builder 🚀
echo =========================================================
echo.

echo [1/4] Running automated tests to ensure stability...
go test -count=1 ./...
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Test suite failed! Aborting release build.
    pause
    exit /b %ERRORLEVEL%
)
echo [OK] All tests passed successfully.
echo.

echo [2/4] Compiling optimized standalone binary (MyERP.exe)...
go build -ldflags="-s -w" -o MyERP.exe .
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Compilation failed!
    pause
    exit /b %ERRORLEVEL%
)
echo [OK] MyERP.exe compiled successfully.
echo.

echo [3/4] Preparing MyERP_Release distribution folder...
if not exist "MyERP_Release" mkdir "MyERP_Release"
if not exist "MyERP_Release\app" mkdir "MyERP_Release\app"
if not exist "MyERP_Release\public" mkdir "MyERP_Release\public"
if not exist "MyERP_Release\storage\logs" mkdir "MyERP_Release\storage\logs"
if not exist "MyERP_Release\storage\cache" mkdir "MyERP_Release\storage\cache"
if not exist "MyERP_Release\storage\uploads" mkdir "MyERP_Release\storage\uploads"

copy /Y "MyERP.exe" "MyERP_Release\MyERP.exe" >nul
if not exist "MyERP_Release\.env" (
    if exist ".env" (
        copy /Y ".env" "MyERP_Release\.env" >nul
    ) else (
        copy /Y ".env.example" "MyERP_Release\.env" >nul
    )
)

xcopy /E /I /Y "app\views" "MyERP_Release\app\views" >nul
xcopy /E /I /Y "public" "MyERP_Release\public" >nul

echo [OK] All release assets packaged into MyERP_Release\
echo.

echo [4/4] Release Package Ready!
echo ---------------------------------------------------------
echo Location: %~dp0MyERP_Release\
echo.
echo Package Contents:
echo   - MyERP.exe   (Backend executable - Closed Source)
echo   - .env        (Environment configuration)
echo   - app\views\  (HTML Templates)
echo   - public\     (CSS, JS, Images)
echo   - storage\    (Logs, Cache, Uploads)
echo ---------------------------------------------------------
echo.
echo You can now ZIP the 'MyERP_Release' folder and send it to your client!
echo.
pause
