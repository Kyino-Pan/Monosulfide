package storage

import (
	"blockEmulator/config"
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type CsvWriter struct {
	buffer   [][]string
	fileName string
	mu       sync.Mutex
	cond     *sync.Cond
	done     chan bool
	Row      int
}

func NewCsvWriter(row int, fileName string) *CsvWriter {
	ret := &CsvWriter{
		buffer:   make([][]string, 0),
		fileName: fileName,
		mu:       sync.Mutex{},
		done:     make(chan bool),
		Row:      row,
	}
	ret.cond = sync.NewCond(&ret.mu)
	return ret
}

var CommLogger *CsvWriter
var StateLogger *CsvWriter

func Init(cnt int) {
	CommLogger = NewCsvWriter(cnt, "out_"+strconv.Itoa(cnt)+".csv")
	StateLogger = NewCsvWriter(-1, "state.csv")
}

func (cw *CsvWriter) ResetRow(cnt int) {
	cw.Row = cnt
}

func (cw *CsvWriter) Writef(format string, args ...interface{}) {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	// 获取当前时间戳
	timestamp := time.Now().Format("15:04:05.000000")

	// 使用fmt.Sprintf进行格式化
	formattedStr := fmt.Sprintf(format, args...)

	// 创建一行数据，初始化为空字符串
	record := make([]string, cw.Row+2) // i+1列需要i+2个元素，因为第一列是时间戳
	record[0] = timestamp              // 第一列是时间戳
	record[cw.Row+1] = formattedStr    // 第i+1列是格式化后的字符串
	// 将记录追加到缓冲区
	cw.buffer = append(cw.buffer, record)
	// 唤醒等待中的csvWriter线程
	cw.cond.Signal()
}

func (cw *CsvWriter) Run() *CsvWriter {
	for {
		cw.mu.Lock()
		// 如果缓冲区没有数据，等待
		for len(cw.buffer) == 0 {
			cw.cond.Wait()
		}
		// 取出缓冲区中的数据
		data := cw.buffer
		cw.buffer = nil // 清空缓冲区
		cw.mu.Unlock()
		file, err := os.OpenFile(config.OutputPath+cw.fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		// 文件锁定
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_EX)

		writer := csv.NewWriter(file)
		{
			// 将每一行数据写入文件
			for _, record := range data {
				_ = writer.Write(record)
			}
			writer.Flush()
		}
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		file.Close()
	}
}

func Merge() error {
	i := 0
	outputFile := config.OutputPath + "Fide-Result" + time.Now().Format("15:04:05") + ".csv"
	// 创建或打开最终输出文件
	f, err := os.OpenFile(outputFile, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %v", err)
	}
	defer f.Close()
	// 开始循环，直到没有更多的文件
	for {
		inputFile := fmt.Sprintf("%sFide-Result-%d.csv", config.OutputPath, i)
		_, err := os.Stat(inputFile)

		// 如果文件不存在，则退出循环
		if os.IsNotExist(err) {
			break
		} else if err != nil {
			return fmt.Errorf("error checking file existence: %v", err)
		}

		// 打开输入文件
		file, err := os.Open(inputFile)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %v", inputFile, err)
		}

		// 读取 CSV 内容
		reader := csv.NewReader(bufio.NewReader(file))
		records, err := reader.ReadAll()
		if err != nil {
			return fmt.Errorf("failed to read CSV file %s: %v", inputFile, err)
		}

		// 删除第一列并提取第二列数据
		var secondColumn []string
		for _, record := range records {
			if len(record) > 1 {
				secondColumn = append(secondColumn, record[1])
			}
		}
		// 打开输出文件，准备写入数据
		outFile, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open output file %s: %v", outputFile, err)
		}

		// 写入第二列数据作为一行
		writer := csv.NewWriter(outFile)
		writer.Write(secondColumn)
		writer.Flush()

		outFile.Close()
		file.Close()
		// 删除已处理的 CSV 文件
		if err := os.Remove(inputFile); err != nil {
			return fmt.Errorf("failed to delete file %s: %v", inputFile, err)
		}
		// 增加i，继续循环
		i++
	}
	return nil
}

func MergeCsv() {
	var allRecords [][]string
	_ = Merge()
	// 搜索并读取所有的 out_x.csv 文件
	for i := 0; ; i++ {
		fileName := fmt.Sprintf(config.OutputPath+"out_%d.csv", i)
		if _, err := os.Stat(fileName); os.IsNotExist(err) {
			break // 如果文件不存在，停止搜索
		}
		// 打开文件
		file, err := os.Open(fileName)
		if err != nil {
			fmt.Printf("无法打开文件 %s: %v\n", fileName, err)
			continue
		}

		reader := csv.NewReader(file)
		records, err := reader.ReadAll()
		file.Close()

		if err != nil {
			fmt.Printf("读取文件 %s 时出错: %v\n", fileName, err)
			continue
		}

		// 将读取的记录添加到allRecords中
		allRecords = append(allRecords, records...)
	}

	// 如果没有记录，直接返回
	if len(allRecords) == 0 {
		fmt.Println("没有找到任何 CSV 文件或文件中没有数据")
		return
	}

	// 按照时间戳（第一列）升序排序
	sort.Slice(allRecords, func(i, j int) bool {
		// 将时间戳字符串解析为时间类型
		timeI, _ := time.Parse("15:04:05.000000", allRecords[i][0])
		timeJ, _ := time.Parse("15:04:05.000000", allRecords[j][0])
		return timeI.Before(timeJ)
	})
	// 创建合并后的 CSV 文件
	// 创建带有时间戳的合并后的 CSV 文件名
	currentTime := time.Now().Format("20060102_150405") // 格式化为 YYYYMMDD_HHMMSS
	mergedFileName := fmt.Sprintf(config.OutputPath+"merged_%s.csv", currentTime)

	mergedFile, err := os.Create(mergedFileName)
	if err != nil {
		fmt.Printf("无法创建合并后的文件 %s: %v\n", mergedFileName, err)
		return
	}
	defer mergedFile.Close()
	writer := csv.NewWriter(mergedFile)
	defer writer.Flush()

	// 将所有记录写入合并后的文件
	for _, record := range allRecords {
		err := writer.Write(record)
		if err != nil {
			fmt.Printf("写入记录时出错: %v\n", err)
			return
		}
	}
	fmt.Printf("合并完成，结果保存在 %s 文件中\n", mergedFileName)

	// 合并完成后删除所有 out_x.csv 文件
	for i := 0; ; i++ {
		fileName := fmt.Sprintf(config.OutputPath+"out_%d.csv", i)
		if _, err := os.Stat(fileName); os.IsNotExist(err) {
			break // 如果文件不存在，停止删除
		}
		err := os.Remove(fileName)
		if err != nil {
			fmt.Printf("删除文件 %s 时出错: %v\n", fileName, err)
			continue
		}
		fmt.Printf("已删除文件 %s\n", fileName)
	}
}
