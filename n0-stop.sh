#!/bin/zsh

# 运行 go run 命令，参数根据您提供的 .bat 文件
go run mainTester.go -k a -p 10

#go run mainTester.go -k a -p a
#go run mainTester.go -k a -p a
#go run main.go -n 2 -N 4 -s 0 -S 2 -m 3 &
#go run main.go -n 2 -N 4 -s 1 -S 2 -m 3 &
#go run main.go -n 3 -N 4 -s 0 -S 2 -m 3 &
#go run main.go -n 3 -N 4 -s 1 -S 2 -m 3 &
#go run main.go -c -N 4 -S 2 -m 3 &
#go run main.go -n 0 -N 4 -s 0 -S 2 -m 3 &
#go run main.go -n 0 -N 4 -s 1 -S 2 -m 3 &

# 等待所有后台任务完成
wait
