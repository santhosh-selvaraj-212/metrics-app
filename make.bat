@echo off
set BUILD_DIR=bin
set TOOLS_DIR=tools

if "%1"=="" goto help
if "%1"=="all" goto all
if "%1"=="clean" goto clean
if "%1"=="build" goto build
if "%1"=="test" goto test
if "%1"=="run" goto run
if "%1"=="help" goto help
echo ERROR: unknown command: %1
goto help

:all
call :clean
call :test
call :build
call :run api
goto :eof

:build
if not exist %BUILD_DIR% mkdir %BUILD_DIR%
REM 
echo Building metrics-api.exe...
go build -ldflags="-s -w" -o %BUILD_DIR%\metrics-api.exe .\cmd\api
if %errorlevel% neq 0 (
    echo Error building metrics-api.exe
    exit /b %errorlevel%
)
echo Building metrics-ingest.exe...
go build -ldflags="-s -w" -o %BUILD_DIR%\metrics-ingest.exe .\cmd\ingest
if %errorlevel% neq 0 (
    echo Error building metrics-ingest.exe
    exit /b %errorlevel%
)
echo Build complete.
goto :eof

:clean
if exist %BUILD_DIR% rd /s /q %BUILD_DIR%
if exist %TOOLS_DIR% rd /s /q %TOOLS_DIR%
echo Cleaning Go modules...
go mod tidy
echo Clean complete.
goto :eof

:run
if "%2"=="" (
    echo ERROR: "run" command requires an argument: api or ingest
    goto help
)
call :build

pushd %BUILD_DIR%

if "%2"=="api" (
    echo Running metrics-api.exe from %CD%...
    metrics-api.exe
) else if "%2"=="ingest" (
    echo Running metrics-ingest.exe from %CD%...
    metrics-ingest.exe
) else (
    echo ERROR: Invalid run argument. Use "api" or "ingest".
    popd
    goto help
)

popd
goto :eof

:test
echo Running tests with race detector and coverage...
go test -race -cover -coverprofile=coverage.txt -covermode=atomic ./internal/...
if %errorlevel% neq 0 (
    echo Tests failed.
    exit /b %errorlevel%
)
echo Tests passed.
goto :eof

:help
echo Available commands:
echo   all      Run clean, test, then build and run the API server.
echo   build    Build the API server and ingestion script binaries.
echo   clean    Clean up build artifacts and Go modules.
echo   run [api|ingest] Run the built API server or ingestion script.
echo   test     Run tests with race detection and coverage.
echo   help     Display this help message.
goto :eof
