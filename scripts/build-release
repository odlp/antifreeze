#!/bin/bash

set -e

SANDBOX=$(mktemp -d)

printf "Building Linux...\n"
GOOS=linux go build -o $SANDBOX/antifreeze-linux github.com/odlp/antifreeze

printf "Building OSX...\n"
GOOS=darwin go build -o $SANDBOX/antifreeze-darwin github.com/odlp/antifreeze

printf "Building Windows...\n"
GOOS=windows go build -o $SANDBOX/antifreeze.exe github.com/odlp/antifreeze

printf "\nBuild summary:\n"
find $SANDBOX -type f -exec file {} \;

printf "\nSHA-1 digests for CF cli plugin repo:\n"
shasum $SANDBOX/antifreeze*

printf "\nBinaries are located here:\n$SANDBOX\n"
