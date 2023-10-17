#!/bin/sh

pid=$(pgrep -f "go run main.go")
test -n "$pid" && kill -9 $pid
go run main.go -v -d=./av &
