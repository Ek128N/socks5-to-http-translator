@echo off
:: Check for admin rights
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo.
    echo This script requires administrator privileges.
    echo Please right-click and "Run as Administrator".
    pause
    exit /b 1
)

:: Get full path to executable
setlocal
for %%I in ("%~dp0\..") do set ROOT=%%~fI
set EXE=%ROOT%\bin\proxy.exe

:: If not built, build it
if not exist "%EXE%" (
    echo Building production executable...
    call "%ROOT%\scripts\build-prod.bat"
    if not exist "%EXE%" (
        echo Build failed. Exiting.
        exit /b 1
    )
)

:menu
echo.
echo 1. Run in Console
echo 2. Install as Service
echo 3. Uninstall Service
echo 4. Exit
set /p choice=Choose [1-4]:

if "%choice%"=="1" (
    "%EXE%" console
    goto menu
) else if "%choice%"=="2" (
    sc create SOCKSHTTPBridge binPath= "%EXE%" start= auto
    echo Installed service.
    goto menu
) else if "%choice%"=="3" (
    sc delete SOCKSHTTPBridge
    echo Removed service.
    goto menu
) else if "%choice%"=="4" (
    exit
) else (
    echo Invalid choice.
    goto menu
)
