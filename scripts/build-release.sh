#!/bin/bash

set -e

SANDBOX=$(mktemp -d)

echo "Building Linux..."
GOOS=linux go build -o $SANDBOX/antifreeze-linux github.com/odlp/antifreeze

echo "Building OSX..."
GOOS=darwin go build -o $SANDBOX/antifreeze-darwin github.com/odlp/antifreeze

echo "Building Windows..."
GOOS=windows go build -o $SANDBOX/antifreeze.exe github.com/odlp/antifreeze

echo

find $SANDBOX -type f -exec file {} \;

echo
echo "Binaries are in: $SANDBOX"
