#!/bin/zsh
go run main.go -S 2 > ./output/out1.txt 2>&1 &
go run main.go -S 2 > ./output/out2.txt 2>&1 &
go run main.go -S 2 > ./output/out3.txt 2>&1 &
go run main.go -S 2 > ./output/out4.txt 2>&1 &
wait
