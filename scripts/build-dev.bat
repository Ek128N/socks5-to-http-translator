@echo off
setlocal

for %%I in ("%~dp0\..") do set ROOT=%%~fI
set OUT=%ROOT%\bin\proxy-dev.exe
set CONFIG_SRC=%ROOT%\config\config.yaml
set CONFIG_DST=%ROOT%\bin\config.yaml

echo === Building Dev Version ===

go build -tags=debug -o "%OUT%" -gcflags "all=-N -l" "%ROOT%"
if %ERRORLEVEL% neq 0 (
    echo Build failed.
    exit /b %ERRORLEVEL%
)

echo Copying config.yaml to output folder...
copy "%CONFIG_SRC%" "%CONFIG_DST%" >nul

echo Build complete: %OUT%

endlocal