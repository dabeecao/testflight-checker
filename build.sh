#!/bin/bash

# Build for current architecture
go build -o testflight-checker main.go

echo "Build complete! You can run the bot with ./testflight-checker"
