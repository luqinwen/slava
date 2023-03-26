package stream

import (
	"bytes"
	"encoding/binary"
	"errors"
)

const (
	LP_MAX_INT_ENCODING_LEN = 9          // ListPack 中整数类型的最大编码长度
	LP_EOF                  = byte(0xFF) // ListPack 中结束符的字节表示
)

var (
	ErrInvalidListPack = errors.New("invalid listpack format") // 解析 ListPack 时出现格式错误
)

type ListPack struct {
	data []byte // 存储 ListPack 的字节数组
}

func NewListPack() *ListPack {
	lp := &ListPack{
		data: []byte{LP_EOF}, // 初始化时添加一个结束符
	}
	return lp
}

func (lp *ListPack) Add(element []byte) {
	entry := lp.encodeEntry(element)                     // 将元素编码为 ListPack 格式
	lp.data = append(lp.data[:len(lp.data)-1], entry...) // 将编码后的元素添加到 ListPack 中
	lp.data = append(lp.data, LP_EOF)                    // 在 ListPack 结尾添加一个结束符
}

func (lp *ListPack) encodeEntry(element []byte) []byte {
	var buf bytes.Buffer // 用于存储编码后的元素

	elementSize := len(element) // 计算元素的大小
	if elementSize < 64 {
		buf.WriteByte(byte(elementSize)) // 如果元素大小小于 64，使用 1 字节表示元素大小
	} else if elementSize < 4096 {
		buf.WriteByte(byte((elementSize>>8)&0xFF) | 0x40) // 如果元素大小小于 4096，使用 2 字节表示元素大小
		buf.WriteByte(byte(elementSize & 0xFF))
	} else {
		buf.WriteByte(0x80 | byte(LP_MAX_INT_ENCODING_LEN-1)) // 如果元素大小大于等于 4096，使用多字节表示元素大小
		binary.Write(&buf, binary.LittleEndian, uint64(elementSize))
	}

	buf.Write(element) // 将元素的值写入缓冲区
	return buf.Bytes() // 返回编码后的元素
}

func (lp *ListPack) Iterate(callback func(index int, value []byte) bool) error {
	var index int
	reader := bytes.NewReader(lp.data) // 创建一个读取 ListPack 的 Reader

	for {
		entry, err := lp.readEntry(reader) // 读取 ListPack 中的一个元素
		if err == ErrInvalidListPack {     // 如果解析错误，返回错误
			return err
		}
		if entry == nil { // 如果遇到 ListPack 结尾，结束循环
			break
		}

		if !callback(index, entry) { // 对每个元素执行指定的回调函数
			break
		}
		index++
	}
	return nil
}

func (lp *ListPack) readEntry(reader *bytes.Reader) ([]byte, error) {
	encoding, err := reader.ReadByte() // 读取 ListPack 中元素的编码
	if err != nil {
		return nil, nil
	}
	if encoding == LP_EOF { // 如果遇到 ListPack 结尾，返回 nil
		return nil, nil
	}

	var elementSize int
	if encoding < 64 { // 如果编码表示元素大小的字节小于 64，直接解析元素大小
		elementSize = int(encoding)
	} else if encoding < 0x80 { // 如果编码表示元素大小的字节在 64 和 128 之间，使用 2 字节解析元素大小
		elementSize = int((encoding&0x3F)<<8) | int(encoding)
	} else { // 如果编码表示元素大小的字节大于等于 128，使用多字节解析元素大小
		elementSize = int(binary.LittleEndian.Uint64(lp.readBytes(reader, LP_MAX_INT_ENCODING_LEN-1)))
	}

	element := lp.readBytes(reader, elementSize) // 读取元素的值
	return element, nil
}

func (lp *ListPack) readBytes(reader *bytes.Reader, n int) []byte {
	buf := make([]byte, n)
	_, _ = reader.Read(buf) // 读取指定长度的字节
	return buf
}

func (lp *ListPack) Bytes() []byte {
	return lp.data
}
