package stream

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// StreamID 表示一个Stream的ID
type StreamID struct {
	Ms  uint64 // 消息的时间戳，单位为毫秒
	Seq uint64 // 消息序列号，用于在同一毫秒内排序
}

// Stream 表示一个 Stream
type Stream struct {
	rax    radixTree                       // 存储 Stream 中所有消息的 radix 树，key 为消息的 ID，value 为 ListPack 格式的消息内容
	length uint64                          // Stream 中消息的总数
	lastID StreamID                        // Stream 中最后一条消息的 ID
	groups map[string]*StreamConsumerGroup // Stream 的消费者组信息
}

// StreamConsumer 表示一个消费者
type StreamConsumer struct {
	seenTime time.Time               // 消费者最后一次看到该消费组的消息的时间戳
	name     string                  // 消费者名字
	pending  map[StreamID]StreamNACK // 存储消费者还未确认的消息，key 为消息 ID，value 为消息的 StreamNACK
}

// StreamConsumerGroup 表示一个消费者组
type StreamConsumerGroup struct {
	lastID    StreamID                   // 消费组最后一个消息的 ID
	pel       map[StreamID]StreamNACK    // 存储消费组下游所有消费者当前还未处理的消息
	consumers map[string]*StreamConsumer // 存储消费组下游所有消费者，key 为消费者名字，value 为 StreamConsumer 结构体指针
}

// StreamNACK 表示一个尚未确认的消息
type StreamNACK struct {
	deliveryTime    time.Time       // 消息被投递给消费者的时间戳
	deliveryCount   uint64          // 消息被投递给消费者的次数，每次投递加 1
	currentConsumer *StreamConsumer // 当前持有该消息的消费者
}

// NewStreamID 生成新的 StreamID
func NewStreamID(ms, seq uint64) *StreamID {
	return &StreamID{Ms: ms, Seq: seq}
}

// ParseStreamID 解析字符串表示的 StreamID
func ParseStreamID(str string) (*StreamID, error) {
	parts := strings.Split(str, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid stream ID")
	}

	ms, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid stream ID")
	}

	seq, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid stream ID")
	}

	return &StreamID{Ms: ms, Seq: seq}, nil
}

func decodeMsgFields(lp *ListPack) (map[string]string, error) {
	fields := make(map[string]string)
	var err error

	// 解析ListPack中的字段名和值
	err = lp.Iterate(func(index int, value []byte) bool {
		if index%2 == 0 {
			// 字段名
			key := string(value)
			fields[key] = ""
		} else {
			// 字段值
			key := getLastKey(fields)
			fields[key] = string(value)
		}
		return true
	})

	if err != nil {
		return nil, err
	}

	return fields, nil
}

// 获取最后一个key
func getLastKey(m map[string]string) string {
	var lastKey string
	for k := range m {
		lastKey = k
	}
	return lastKey
}

func XADD(stream *Stream, id *StreamID, fields map[string]string) (*StreamID, error) {
	// 生成新的 StreamID
	if id == nil {
		if stream.length == 0 {
			id = &StreamID{Ms: uint64(time.Now().UnixNano() / 1000000), Seq: 0}
		} else {
			id = &StreamID{Ms: stream.lastID.Ms, Seq: stream.lastID.Seq + 1}
		}
	} else if id.Seq < stream.lastID.Seq {
		return nil, fmt.Errorf("invalid sequence number")
	} else if id.Seq == stream.lastID.Seq && id.Ms <= stream.lastID.Ms {
		return nil, fmt.Errorf("invalid timestamp")
	}

	// 将数据转换成 ListPack 格式
	lp := NewListPack()
	lp.Add([]byte("id"))
	lp.Add([]byte(fmt.Sprintf("%d-%d", id.Ms, id.Seq)))
	for k, v := range fields {
		lp.Add([]byte(k))
		lp.Add([]byte(v))
	}

	// 将数据插入到 Stream 中
	stream.rax.Insert(fmt.Sprintf("%d-%d", id.Ms, id.Seq), lp)

	// 更新 Stream 的信息
	stream.length++
	stream.lastID = *id

	// 返回新插入数据的 ID
	return id, nil
}

// XDEL 从 Stream 中删除指定的消息
func XDEL(stream *Stream, ids ...*StreamID) (uint64, error) {
	var deleted uint64
	for _, id := range ids {
		if !stream.rax.root.delete(fmt.Sprintf("%d-%d", id.Ms, id.Seq)) {
			continue
		}
		deleted++
		stream.length--
		if stream.length == 0 {
			stream.lastID = StreamID{}
		}
	}
	return deleted, nil
}

// XLEN 返回 Stream 中消息的数量
func XLEN(stream *Stream) uint64 {
	return stream.length
}

func XRANGE(stream *Stream, startID *StreamID, endID *StreamID, count int) ([]*StreamID, []map[string]string, error) {
	// 根据 startID 和 endID 查找所有符合条件的消息
	var start, end string
	if startID == nil {
		start = "-"
	} else {
		start = fmt.Sprintf("%d-%d", startID.Ms, startID.Seq)
	}
	if endID == nil {
		end = "+"
	} else {
		end = fmt.Sprintf("%d-%d", endID.Ms, endID.Seq)
	}
	ids := make([]*StreamID, 0, count)
	fieldsList := make([]map[string]string, 0, count)
	stream.rax.Walk(func(key string, value *ListPack) bool {
		if key >= start && key <= end {
			fields, err := decodeMsgFields(value)
			if err != nil {
				return false
			}
			id, _ := ParseStreamID(key)
			ids = append(ids, id)
			fieldsList = append(fieldsList, fields)
			if count > 0 && len(ids) == count {
				return false
			}
		}
		return true
	})

	return ids, fieldsList, nil
}

// XREVRANGE 返回在指定范围内的所有消息的 StreamID 和对应的字段
func XREVRANGE(stream *Stream, startID *StreamID, endID *StreamID, count int) ([]*StreamID, []map[string]string, error) {
	// 根据 startID 和 endID 查找所有符合条件的消息
	var start, end string
	if startID == nil {
		end = "+"
	} else {
		end = fmt.Sprintf("%d-%d", startID.Ms, startID.Seq)
	}
	if endID == nil {
		start = "-"
	} else {
		start = fmt.Sprintf("%d-%d", endID.Ms, endID.Seq)
	}

	// 初始化返回结果的 slice
	ids := make([]*StreamID, 0, count)
	fieldsList := make([]map[string]string, 0, count)

	// 遍历 stream 中的所有消息
	stream.rax.Walk(func(key string, value *ListPack) bool {
		// 如果消息的 StreamID 在指定的范围内，则解码该消息并将 StreamID 和字段添加到返回结果中
		if key >= start && key <= end {
			fields, err := decodeMsgFields(value)
			if err != nil {
				return false
			}
			id, _ := ParseStreamID(key)
			ids = append(ids, id)
			fieldsList = append(fieldsList, fields)
			// 如果 count 大于 0，并且已经返回的消息数量等于 count，则结束遍历
			if count > 0 && len(ids) == count {
				return false
			}
		}
		return true
	})

	// 将结果倒序排列，即得到 XREVRANGE 的结果
	for i := 0; i < len(ids)/2; i++ {
		ids[i], ids[len(ids)-1-i] = ids[len(ids)-1-i], ids[i]
		fieldsList[i], fieldsList[len(fieldsList)-1-i] = fieldsList[len(fieldsList)-1-i], fieldsList[i]
	}

	return ids, fieldsList, nil
}

// XTRIM 截断 Stream，只保留指定范围内的消息
func XTRIM(stream *Stream, maxLen uint64, approximate bool) (uint64, error) {
	if stream.length <= maxLen {
		// Stream 的长度小于等于 maxLen，无需截断
		return 0, nil
	}

	// 计算要截断的消息数量
	var trimCount uint64
	if approximate {
		// 如果 approximate 为 true，则使用 stream 的长度减去 maxLen 作为要截断的消息数量
		trimCount = stream.length - maxLen
	} else {
		// 如果 approximate 为 false，则需要精确计算要截断的消息数量
		ids, _, err := XRANGE(stream, &StreamID{}, &StreamID{}, int(maxLen))
		if err != nil {
			return 0, err
		}
		if len(ids) < int(maxLen) {
			// 如果 stream 中的消息数量小于等于 maxLen，则无需截断
			return 0, nil
		}
		trimCount = stream.length - uint64(len(ids))
	}

	// 删除需要截断的消息
	ids, _, err := XRANGE(stream, nil, nil, int(trimCount))
	if err != nil {
		return 0, err
	}
	deleted, err := XDEL(stream, ids...)
	if err != nil {
		return 0, err
	}
	return deleted, nil
}

// XREAD 从 Stream 中读取消息
func XREAD(stream *Stream, id *StreamID, count int, block time.Duration) ([]*StreamID, []map[string]string, error) {
	// 如果 id 为 nil，则读取最新的消息
	if id == nil {
		lastID, err := ParseStreamID("-1")
		if err != nil {
			return nil, nil, err
		}
		id = lastID
	}

	// 如果 block 不为 0，则使用阻塞模式读取消息
	var timeout <-chan time.Time
	if block > 0 {
		timeout = time.After(block)
	}

	for {
		// 查找符合条件的消息
		ids, fieldsList, err := XRANGE(stream, id, nil, count)
		if err != nil {
			return nil, nil, err
		}
		if len(ids) > 0 {
			// 如果找到了消息，则返回
			return ids, fieldsList, nil
		}

		// 如果 block 不为 0，且没有找到消息，则等待指定时间再重试
		if block > 0 {
			select {
			case <-timeout:
				return nil, nil, nil
			default:
				time.Sleep(50 * time.Millisecond)
			}
		} else {
			return nil, nil, nil
		}
	}
}

// XGROUP CREATE 创建一个新的消费者组
func XGROUP_CREATE(stream *Stream, groupName, startID string) error {
	if _, ok := stream.groups[groupName]; ok {
		return fmt.Errorf("consumer group already exists")
	}

	// 解析 startID
	var start *StreamID
	if startID != "$" {
		var err error
		start, err = ParseStreamID(startID)
		if err != nil {
			return fmt.Errorf("invalid start ID")
		}
		if _, _, err := XRANGE(stream, start, start, 1); err != nil {
			return fmt.Errorf("invalid start ID")
		}
	} else {
		start = &stream.lastID
	}

	stream.groups[groupName] = &StreamConsumerGroup{
		lastID:    *start,
		pel:       make(map[StreamID]StreamNACK),
		consumers: make(map[string]*StreamConsumer),
	}

	return nil
}

// XACK 从消费者组的挂起列表中删除消息并将其标记为已确认
func XACK(stream *Stream, group string, ids ...*StreamID) (uint64, error) {
	// 查找消费者组
	g, ok := stream.groups[group]
	if !ok {
		return 0, fmt.Errorf("consumer group not found")
	}
	var ackCount uint64

	// 更新消费组中待处理消息的列表
	for _, id := range ids {
		// 在消费组中找到该消息对应的 PEL 节点
		nack, ok := g.pel[*id]
		if !ok {
			return 0, fmt.Errorf("PEL node not found")
		}

		// 更新 PEL 节点中该消费者的状态
		nack.deliveryCount--
		ackCount++

		// 如果该消息已经被所有消费者确认，则将该消息从 PEL 中删除
		if nack.deliveryCount == 0 {
			delete(g.pel, *id)
		}
	}

	return ackCount, nil
}

func XGROUPSETID(stream *Stream, groupName, consumerName, id string) error {
	// 检查 Stream 是否存在
	if stream == nil {
		return fmt.Errorf("stream not exists")
	}

	// 查找指定的消费组
	group, exists := stream.groups[groupName]
	if !exists {
		return fmt.Errorf("group not exists")
	}

	// 更新消费者ID
	if _, exists := group.consumers[consumerName]; exists {
		group.consumers[consumerName].seenTime = time.Now()
		group.consumers[consumerName].name = id
	} else {
		return fmt.Errorf("consumer not exists")
	}

	return nil
}

// XGROUP DELCONSUMER 从消费者组中删除一个消费者，该消费者将不再接收组内的消息
func XGROUP_DELCONSUMER(stream *Stream, groupName string, consumerName string) (bool, error) {
	if _, ok := stream.groups[groupName]; !ok {
		return false, fmt.Errorf("group %s does not exist", groupName)
	}

	if _, ok := stream.groups[groupName].consumers[consumerName]; !ok {
		return false, fmt.Errorf("consumer %s does not exist in group %s", consumerName, groupName)
	}

	// 删除消费者信息
	delete(stream.groups[groupName].consumers, consumerName)

	// 删除消费者持有但未确认的消息
	stream.groups[groupName].consumers[consumerName].pending = nil

	return true, nil
}

func XGROUPDESTROY(stream *Stream, groupName string) (bool, error) {
	_, ok := stream.groups[groupName]
	if !ok {
		return false, nil
	}
	delete(stream.groups, groupName)
	return true, nil
}

// XPENDING 返回消费组中未处理的消息信息
func XPENDING(stream *Stream, group string, startID, endID *StreamID, count int, consumer string) ([]*StreamNACK, error) {
	// 获取消费组信息
	groupInfo, exists := stream.groups[group]
	if !exists {
		return nil, fmt.Errorf("consumer group does not exist")
	}

	entries := make([]*StreamNACK, 0, count)

	for k, v := range groupInfo.pel {
		if (consumer == "" || v.currentConsumer.name == consumer) && k.Ms >= startID.Ms && k.Ms <= startID.Ms {
			entries = append(entries, &v)
		}
	}

	return entries, nil
}
