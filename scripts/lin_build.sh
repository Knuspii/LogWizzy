#!/bin/bash
set -e

echo "Updating Go modules..."
go get -u ../.
echo "Modules updated successfully."

echo "Formatting Go source files..."
gofmt -s -w ../.
echo "Formatting done."

echo "Building LogWizzy for Linux amd64..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../bin/logwizzy ../.
echo "Linux amd64 build succeeded."

echo "Building LogWizzy for Linux ARM64..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ../bin/logwizzy-arm64 ../.
echo "Linux ARM64 build succeeded."

echo "Tidying up Go modules..."
go mod tidy
echo "Modules tidied successfully."

echo "All builds finished successfully."
