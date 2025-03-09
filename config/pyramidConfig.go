package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"time"
)

var (
	shardAmount     = 6
	PyrConf         = InitShardConfig()
	ActivatePyramid = false
	Record          = make([]time.Duration, 12)
)

type PyramidConfig struct {
	ShardAmount       int
	ShardDistribution [][]bool  // sd[i][j]=true means shard i has chain j
	shardRoute        [][][]int // sr[i][j][0] contains the first step to get to shard j
	Enable            bool
	DefaultAddr       []string
}

// 深度优先搜索检查 shard 是否连通
func dfs(shard int, visited []bool, sd [][]bool) {
	visited[shard] = true
	for neighbor := 0; neighbor < len(sd[shard]); neighbor++ {
		// 如果 shard i->neighbor 或者 neighbor->i 存在边且 neighbor 未访问过
		if (sd[shard][neighbor] || sd[neighbor][shard]) && !visited[neighbor] {
			dfs(neighbor, visited, sd)
		}
	}
}

// 检查是否所有 shards 连通
func (conf *PyramidConfig) checkConnectivity() error {
	visited := make([]bool, conf.ShardAmount)
	dfs(0, visited, conf.ShardDistribution) // 从第一个 shard 开始深度优先遍历
	// 检查是否所有的 shard 都被访问过
	for i, v := range visited {
		if !v {
			return fmt.Errorf("shard %d is not connected to all other shards", i)
		}
	}
	return nil
}

// route 构建每对 shard 间的路径
func (conf *PyramidConfig) route() {
	conf.shardRoute = make([][][]int, conf.ShardAmount)
	for i := 0; i < conf.ShardAmount; i++ {
		conf.shardRoute[i] = make([][]int, conf.ShardAmount)
	}
	for i := 0; i < conf.ShardAmount; i++ {
		for j := 0; j < conf.ShardAmount; j++ {
			// 可以优化成j从i+1开始，但是懒
			linked := false
			if i != j {
				// 检查是否有直接连接
				if !(conf.ShardDistribution[i][j] || conf.ShardDistribution[j][i]) {
					// 不直接相连
					for k := 0; k < conf.ShardAmount; k++ {
						if k != i && k != j {
							if conf.ShardDistribution[k][i] && conf.ShardDistribution[k][j] {
								linked = true
								break
							}
						}
					}
				} else {
					linked = true
				}
				//且没有共同父节点
				if linked == false {
					conf.shardRoute[i][j] = conf.findRoute(i, j)
				}
			}
		}
	}
}

func (conf *PyramidConfig) findRoute(i int, j int) []int {
	// 用于记录每个节点是否已访问
	vis := make([]bool, conf.ShardAmount)
	// 队列存储当前正在处理的节点
	q := append(make([]int, 0), i)
	// prev 数组记录每个节点的前一个节点，以便构建路径
	prev := make([]int, conf.ShardAmount)
	for k := range prev {
		prev[k] = -1 // -1 表示没有前驱节点
	}
	vis[i] = true // 标记起始节点已访问

	// 开始 BFS
	for len(q) != 0 {
		curr := q[0]
		q = q[1:]

		// 检查是否到达目标节点
		if curr == j {
			// 通过 prev 数组构建路径
			var path []int
			for at := j; at != -1; at = prev[at] {
				path = append([]int{at}, path...)
			}
			for idx := 1; idx < len(path)-1; idx++ {
				sid := path[idx]
				pre := path[idx-1]
				nxt := path[idx+1]
				if conf.ShardDistribution[sid][pre] && conf.ShardDistribution[sid][nxt] {
					path = append(path[:idx], path[idx+1:]...)
				}
			}
			return path
		}
		// 遍历当前节点的邻居
		for neighbor := 0; neighbor < conf.ShardAmount; neighbor++ {
			// shardDistribution 视为无向图，因此需要检查双向边
			if (conf.ShardDistribution[curr][neighbor] || conf.ShardDistribution[neighbor][curr]) && !vis[neighbor] {
				q = append(q, neighbor)
				vis[neighbor] = true
				prev[neighbor] = curr // 记录前驱节点
			}
		}
	}

	// 如果找不到路径，返回空切片
	return []int{}
}

func (conf *PyramidConfig) InRoute(i int, s int, r int) bool {
	rt := conf.shardRoute[s][r]
	if rt == nil {
		return false
	}
	for _, p := range rt {
		if conf.ShardDistribution[i][p] == true {
			return true
		}
	}
	return false
}

func (conf *PyramidConfig) GetRoute(s, r int) []int {
	return conf.shardRoute[s][r]
}

func InitShardConfig() *PyramidConfig {
	conf := new(PyramidConfig)
	conf.ShardAmount = shardAmount
	conf.ShardDistribution = make([][]bool, shardAmount)
	dis := conf.ShardDistribution
	for i := 0; i < shardAmount; i++ {
		dis[i] = make([]bool, shardAmount)
		dis[i][i] = true
	}
	dis[3][0] = true
	dis[3][1] = true
	dis[3][2] = true
	dis[4][1] = true
	dis[4][2] = true
	dis[4][5] = true
	//
	//for i := 0; i < shardAmount; i++ {
	//	dis[shardAmount-1][i] = true
	//}
	err := conf.checkConnectivity()
	if err != nil {
		log.Panic(err)
	}
	conf.route()
	conf.Enable = ActivatePyramid
	conf.DefaultAddr = make([]string, shardAmount)
	for i := 0; i < shardAmount; i++ {
		conf.DefaultAddr[i] = GenerateShardAddress(i)
	}
	//}
	//dis[i][j] = true -> shard i obtains shard j's information
	return conf
}

// GenerateShardAddress generates a default address for a given shardID
func GenerateShardAddress(shardID int) string {
	if shardID < 0 || shardID >= shardAmount {
		panic("Invalid shardID")
	}
	// 使用 shardID 创建一个唯一的地址 (例如: 基于 hash)
	shardPrefix := fmt.Sprintf("shard-%d", shardID)
	hash := sha256.Sum256([]byte(shardPrefix))
	address := hex.EncodeToString(hash[:])

	// 计算 shardID 对应的最后 8 个字符的 16 进制表示
	last16Addr := fmt.Sprintf("%016x", shardID)

	// 将生成的地址替换最后 8 个字符为 shardID 的 16 进制表示
	finalAddress := "0x" + address[:len(address)-8] + last16Addr

	return finalAddress
}
