#!/bin/zsh
#go run main.go -g 1 -S 1
#wait
#  echo "Runing S1N1"
#  ./test.sh
#wait
#
#go run main.go -g 2 -S 1
#wait
#  echo "Runing S2 N1"
#  ./test.sh
#wait
#
#go run main.go -g 2 -S 2
#wait
#  echo "Runing S2 N2"
#  ./test.sh
#wait
#
#go run main.go -g 4 -S 1
#wait
#  echo "Runing S1 N4"
#  ./test.sh
#wait
#go run main.go -g 4 -S 2
#wait
#  echo "Runing S2 N4"
#  ./test.sh
#wait
#
#go run main.go -g 4 -S 4
#wait
#  echo "Runing S4 N4"
#  ./test.sh
#wait

#go run main.go -g 8 -S 1
#wait
#  echo "Runing S1 N8"
#  ./test.sh
#wait
#
#go run main.go -g 8 -S 2
#wait
#  echo "Runing S2 N8"
#  ./test.sh
#wait

#go run main.go -g 8 -S 4
#wait
#  echo "Runing S4 N8"
#  ./test.sh
#wait

go run main.go -g 8 -S 8
wait
  echo "Runing S8 N8"
  ./test.sh
wait

go run main.go -g 16 -S 4
wait
  echo "Runing S4 N16"
  ./test.sh
wait

go run main.go -g 16 -S 8
wait
  echo "Runing S8 N16"
  ./test.sh
wait

go run main.go -g 16 -S 16
wait
  echo "Runing S16 N16"
  ./test.sh
wait

#go run main.go -g 32 -S 4
#wait
#  echo "Runing S4 N32"
#  ./test.sh
#wait
#
#go run main.go -g 32 -S 8
#wait
#  echo "Runing S8 N32"
#  ./test.sh
#wait
#
#go run main.go -g 32 -S 16
#wait
#  echo "Runing S16 N32"
#  ./test.sh
#wait
#
#go run main.go -g 32 -S 24
#wait
#  echo "Runing S24 N32"
#  ./test.sh
#wait
#
#go run main.go -g 32 -S 32
#wait
#  echo "Runing S32 N32"
#  ./test.sh
#wait
#
## 64 nodes
#go run main.go -g 64 -S 8
#wait
#  echo "Runing S8 N64"
#  ./test.sh
#wait
#
#go run main.go -g 64 -S 16
#wait
#  echo "Runing S16 N64"
#  ./test.sh
#wait
#
#go run main.go -g 64 -S 24
#wait
#  echo "Runing S24 N64"
#  ./test.sh
#wait
#
#go run main.go -g 64 -S 32
#wait
#  echo "Runing S32 N64"
#  ./test.sh
#wait
#
## 128 nodes
#go run main.go -g 128 -S 8
#wait
#  echo "Runing S8 N128"
#  ./test.sh
#wait
#
#go run main.go -g 128 -S 16
#wait
#  echo "Runing S16 N128"
#  ./test.sh
#wait
#
#go run main.go -g 128 -S 24
#wait
#  echo "Runing S24 N128"
#  ./test.sh
#wait
#
#go run main.go -g 128 -S 32
#wait
#  echo "Runing S32 N128"
#  ./test.sh
#wait

## 256 nodes
#go run main.go -g 256 -S 8
#wait
#  echo "Runing S8 N128"
#  ./test.sh
#wait
#
#go run main.go -g 256 -S 16
#wait
#  echo "Runing S16 N256"
#  ./test.sh
#wait
#
#go run main.go -g 256 -S 24
#wait
#  echo "Runing S24 N256"
#  ./test.sh
#wait
#
#go run main.go -g 256 -S 32
#wait
#  echo "Runing S32 N256"
#  ./test.sh
#wait

go run main.go -g 1
wait