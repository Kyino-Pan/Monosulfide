import yaml
import os

# 配置参数
IMAGE_NAME = "monosulfide/monosulfide:test"
S_list = [4]  # 可根据需要修改
output_dir = os.path.abspath("./output")
csv_path = os.path.abspath("../2000000to2999999_BlockTransaction.csv")

services = {}
for S in S_list:
    node_count = S * 4
    for i in range(1, node_count + 1):
        name = f"node_S{S}_{i}"
        port = 20000 + (i - 1) * 10
        services[name] = {
            'image': IMAGE_NAME,
            'networks': ['ms-net'],
            'environment': [
                f'LOCALHOST={"0.0.0.0"}',
                f'DNSADDR=node_S{S}_1',
                f'LISTEN_PORT={port}'
            ],
            'volumes': [
                f'{csv_path}:/app/2000000to2999999_BlockTransaction.csv',
                f'{output_dir}:/app/output'
            ],
            'deploy': {'replicas': 1},
            'command': ["-S", str(S)]
        }

# 额外节点（如有）
services["node_S8_33"] = {
    'image': IMAGE_NAME,
    'networks': ['ms-net'],
    'environment': [
        'LOCALHOST=node_S8_33',
        'DNSADDR=node_S8_1',
        'LISTEN_PORT=20033'
    ],
    'volumes': [
        f'{csv_path}:/app/2000000to2999999_BlockTransaction.csv',
        f'{output_dir}:/app/output'
    ],
    'deploy': {'replicas': 1},
    'command': ["-p"]
}

compose = {
    'version': '3.8',
    'services': services,
    'networks': {
        'ms-net': {'external': True}
    }
}

with open('docker-compose.yml', 'w') as f:
    yaml.dump(compose, f, sort_keys=False)

print("docker-compose.yml 已生成。") 