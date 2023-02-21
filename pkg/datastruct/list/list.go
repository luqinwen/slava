package list

import "math"

// 双向链表的每一个节点
type ListNode struct {
	pre   *ListNode
	next  *ListNode
	value string
	// 【只有string类型】
}

// 双向链表本身
type List struct {
	head *ListNode
	tail *ListNode
	len  int
}

func NewListNode(value string) (listNode *ListNode) {
	listNode = &ListNode{
		value: value,
	}

	return
}

func NewList() (list *List) {
	list = &List{}

	return
}

func (node *ListNode) GetValue() (value string) {
	if node == nil {
		return
	}
	// value = node.value

	return node.value
}

// 返回链表长度
func (list *List) Len() (len int) {
	len = list.len

	return
}

func (list *List) RPush(value string) {
	node := NewListNode(value)
	if list.len == 0 {
		// 链表为空
		list.head = node
		list.tail = node
	} else {
		// 链表不为空，加入到末尾
		list.tail.next = node
		node.pre = list.tail
		list.tail = node
	}

	list.len++
}

func (list *List) LPush(value string) {
	node := NewListNode(value)
	if list.len == 0 {
		list.head = node
		list.tail = node
	} else {
		list.head.pre = node
		node.next = list.head
		list.head = node
	}

	list.len++
}

func (list *List) LPop() (node *ListNode) {
	if list.len == 0 {
		return
	}
	node = list.head
	if node.next == nil {
		list.head = nil
		list.tail = nil
	} else {
		list.head = node.next
	}

	list.len--
	return
}

// 根据index索引获取元素
// index为正数：从左往右0~n；index为负数：从右往左-1~n'
func (list *List) GetByIndex(index int) (node *ListNode) {
	if index >= 0 {
		node = list.head
		for i := 0; i < index && node != nil; i++ {
			node = node.next
		}
	} else {
		node = list.tail
		rightIndex := int(math.Abs(float64(index))) - 1
		for i := 0; i < rightIndex && node != nil; i++ {
			node = node.pre
		}
	}
	return
}

// 返回索引区间内的元素
// [start, stop]
func (list *List) Range(start, stop int) (nodes []*ListNode) {
	nodes = make([]*ListNode, 0)
	node := list.head
	for i := 0; i < list.len; i++ {
		if i >= start && i <= stop {
			nodes = append(nodes, node)
		}
		node = node.next
	}
	return
}
