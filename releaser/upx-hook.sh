#!/bin/bash

TARGET=$1
BINARY=$2

if [[ $TARGET == windows_* ]];
then
    echo "Not a UNIX Target, skipping...";
    exit 0
else
    upx "$BINARY";
fi