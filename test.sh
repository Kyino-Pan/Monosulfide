#!/bin/zsh
go run main.go -S 0 > ./output/out1.txt 2>&1 &
wait
