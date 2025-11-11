@echo off

echo Updating Go modules...
go get -u ..\.
if errorlevel 1 (
    echo Failed to update modules
    exit /b 1
)
echo Modules updated successfully.

echo Formatting Go source files...
gofmt -s -w ..\.
echo Formatting done.

echo Building LogWizard for Windows amd64...
set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64
go build -o ..\bin\logwizard.exe ..\.
if errorlevel 1 (
    echo Windows build failed
    exit /b 1
)
echo Windows build succeeded.

echo Building LogWizard for Linux amd64...
set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64
go build -o ..\bin\logwizard ..\.
if errorlevel 1 (
    echo Linux build failed
    exit /b 1
)
echo Linux amd64 build succeeded.

echo Building LogWizard for Linux ARM64...
set CGO_ENABLED=0
set GOOS=linux
set GOARCH=arm64
go build -o ..\bin\logwizard-arm64 ..\.
if errorlevel 1 (
    echo Linux ARM64 build failed
    exit /b 1
)
echo Linux ARM64 build succeeded.

echo Tidying up Go modules...
go mod tidy
echo Modules tidied successfully.

echo All builds finished successfully.
timeout /t 3
exit