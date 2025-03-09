package message

import (
	"bytes"
	"log"
	"strings"
)

type Content []byte

func escapeTab(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\t", "\\\t")
	return s
}

/*
	[tab]->[esc][tab]
	[esc]->[esc][esc]
*/

func NewStrContent(msgs ...string) *Content {
	escapedMsgs := make([]string, len(msgs))
	for i, msg := range msgs {
		escapedMsgs[i] = escapeTab(msg)
	}
	ret := []byte(strings.Join(escapedMsgs, "\t"))
	return (*Content)(&ret)
}

func NewByteContent(msgs ...*[]byte) *Content {
	escapedMsgs := make([][]byte, len(msgs))
	for i, msg := range msgs {
		escapedMsgs[i] = []byte(escapeTab(string(*msg)))
	}
	ret := bytes.Join(escapedMsgs, []byte("\t"))
	return (*Content)(&ret)
}

func (content *Content) AppendByteContent(msgs ...*[]byte) *Content {
	if len(msgs) == 0 {
		return content
	}
	escapedMsgs := make([][]byte, len(msgs))
	for i, msg := range msgs {
		escapedMsgs[i] = []byte(escapeTab(string(*msg)))
	}
	*content = append(*content, byte('\t'))
	*content = append(*content, bytes.Join(escapedMsgs, []byte("\t"))...)
	return content
}

func (content *Content) AppendStrContent(msgs ...string) *Content {
	if len(msgs) == 0 {
		return content
	}
	escapedMsgs := make([]string, len(msgs))
	for i, msg := range msgs {
		escapedMsgs[i] = escapeTab(msg)
	}
	*content = append(*content, byte('\t'))
	*content = append(*content, []byte(strings.Join(escapedMsgs, "\t"))...)
	return content
}

func (content *Content) ParseContent() [][]byte {
	// 将Content转换回[]byte，然后按\t分割
	// 将分割得到的字节切片数组转换为字符串切片
	msgs := make([][]byte, 0)
	msgs = append(msgs, make([]byte, 0))
	escMod := false
	for i := 0; i < len(*content); i++ {
		b := (*content)[i]
		last := &msgs[len(msgs)-1]
		if b == '\\' {
			if escMod == true {
				*last = append(*last, b)
				escMod = false
			} else {
				escMod = true
			}
		} else if b == '\t' {
			if escMod == true {
				*last = append(*last, b)
				escMod = false
			} else {
				msgs = append(msgs, make([]byte, 0))
			}
		} else {
			*last = append(*last, b)
		}
	}
	return msgs
}

func (content *Content) Sign(name string) *Content {
	content.AppendStrContent(name)
	return content
}

func (content *Content) CheckSign() string { // read and remove the sign from content
	// 首先检查 content 是否为空
	if content == nil || len(*content) == 0 {
		return "ERROR"
	}

	// 寻找最后一个 \t 的位置
	contents := content.ParseContent()
	lastTab := contents[len(contents)-1]
	// 提取签名并更新 content
	signature := string(lastTab)
	if len(signature) > 1000 {
		log.Printf("ERROR")
		log.Printf("%v", signature)
	}
	temp := NewByteContent(&contents[0])
	for i := 1; i < len(contents)-1; i++ {
		temp.AppendByteContent(&contents[i])
	}
	*content = *temp
	return signature
}
