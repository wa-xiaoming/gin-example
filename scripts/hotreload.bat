@echo off
chcp 65001
go install github.com/codegangsta/gin@latest
gin -p 9998 -a 9999 -l 127.0.0.1 -b hotreload run ./main.go -env fat
