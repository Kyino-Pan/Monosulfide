#!/bin/bash

set -e

IMAGE_NAME="monosulfide/monosulfide:test"

# 重新编译Go程序并构建最新镜像
echo "[INFO] 编译最新Go程序..."
GOOS=linux GOARCH=arm64 go build -o app main.go

# 删除旧镜像（如存在）
if docker images | grep -q "monosulfide/monosulfide"; then
  echo "[INFO] 删除旧的 $IMAGE_NAME 镜像..."
  docker rmi -f $IMAGE_NAME || true
fi

echo "[INFO] 构建最新docker镜像..."
docker build --no-cache -t $IMAGE_NAME .

docker network create monosulfide-net || true

mkdir -p log

# 依次对 S=1,2,4,8,16
for S in 16; do
  node_count=$((S * 4))
  echo "[INFO] 启动 S=$S, 共 $node_count 个节点..."
  node_names=()
  for i in $(seq 1 $node_count); do
    name="node_S${S}_$i"
    port=$((20000 + (i - 1) * 10))
    # mkdir -p output_$name
    docker run -d --rm \
      --network monosulfide-net \
      --name $name \
      -e LOCALHOST=$name \
      -e DNSADDR="node_S${S}_1" \
      -e LISTEN_PORT=$port \
      -v $(pwd)/../2000000to2999999_BlockTransaction.csv:/app/2000000to2999999_BlockTransaction.csv \
      -v $(pwd)/output:/app/output \
      $IMAGE_NAME -S $S
    # 日志后台收集
    (docker logs -f $name >> ./log/$name.log 2>&1 &)
    node_names+=("$name")
  done
  echo "所有节点已启动。用 'docker ps' 查看。日志在 ./log 目录下。"
  # 等待本组所有容器退出
  sleep 15
  while [ $(docker ps --filter "ancestor=monosulfide/monosulfide:test" --format '{{.ID}}' | wc -l) -gt 0 ]; do
    echo "还有容器在运行，等待中..."
    sleep 5
  done 
done
docker run -d --rm \
  --network monosulfide-net \
  --name node_S8_33 \
  -e LOCALHOST=node_S8_33 \
  -e DNSADDR=node_S8_1 \
  -e LISTEN_PORT=20033 \
  -v $(pwd)/../2000000to2999999_BlockTransaction.csv:/app/2000000to2999999_BlockTransaction.csv \
  -v $(pwd)/output:/app/output \
  $IMAGE_NAME -p