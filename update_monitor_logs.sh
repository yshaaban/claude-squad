#!/bin/bash
# Script to update all log.InfoLog to log.FileOnlyInfoLog in monitor.go

sed -i.bak 's/log\.InfoLog/log.FileOnlyInfoLog/g' /Users/ysh/src/claude-squad/web/monitor.go

echo "Updated all InfoLog references to FileOnlyInfoLog"