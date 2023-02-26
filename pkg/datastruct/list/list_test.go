/*
 * @Author: libowei
 * @Date: 2023-02-20 23:02:08
 * @LastEditTime: 2023-02-25 14:52:38
 * @FilePath: \slava\pkg\datastruct\list\list_test.go
 */
package list

import (
	"fmt"
	"reflect"
	"testing"
)

func printNode(nodes []*ListNode) []string {
	values := make([]string, 0)
	for _, node := range nodes {
		values = append(values, node.value)
	}
	return values
}

func TestRpush(t *testing.T) {
	list := NewList()
	list.RPush("a")
	list.RPush("b")
	list.RPush("c")

	if list.len != 3 {
		t.Error("RPush error")
	}
	fmt.Printf("want [a b c]  is %s\n", printNode(list.Range(0, 2)))

	if reflect.DeepEqual(printNode(list.Range(0, 2)), []string{"a", "b", "c"}) != true {
		t.Errorf("RPush %s wrong\n", "a,b,c")
	}
}

func TestLpush(t *testing.T) {
	list := NewList()
	list.LPush("a")
	list.LPush("b")
	list.LPush("c")

	if list.len != 3 {
		t.Error("LPush error")
	}
	fmt.Printf("want [c b a]  is %s\n", printNode(list.Range(0, 2)))

	if reflect.DeepEqual(printNode(list.Range(0, 2)), []string{"c", "b", "a"}) != true {
		t.Errorf("LPush %s wrong\n", "c,b,a")
	}
}

func TestRpop(t *testing.T) {
	list := NewList()
	list.RPush("a")
	list.RPush("b")
	list.RPush("c")

	fmt.Printf("want [a b c]  is %s\n", printNode(list.Range(0, 2)))

	list.RPop()
	fmt.Printf("want [a b]  is %s\n", printNode(list.Range(0, 1)))

	list.RPop()
	fmt.Printf("want [a]  is %s\n", printNode(list.Range(0, 1)))

	if reflect.DeepEqual(printNode(list.Range(0, 0)), []string{"a"}) != true {
		t.Errorf("Rpop %s wrong\n", "b,c")
	}
}

func TestLpop(t *testing.T) {
	list := NewList()
	list.RPush("a")
	list.RPush("b")
	list.RPush("c")

	fmt.Printf("want [a b c]  is %s\n", printNode(list.Range(0, 2)))

	list.LPop()
	fmt.Printf("want [b c]  is %s\n", printNode(list.Range(0, 1)))

	list.LPop()
	fmt.Printf("want [c]  is %s\n", printNode(list.Range(0, 1)))

	if reflect.DeepEqual(printNode(list.Range(0, 0)), []string{"c"}) != true {
		t.Errorf("Lpop %s wrong\n", "a,b")
	}
}

func TestRange(t *testing.T) {
	list := NewList()
	fmt.Println("Range", printNode(list.Range(0, -1)))
	fmt.Println("Range", printNode(list.Range(0, 2)))

	list.RPush("a")
	list.RPush("b")
	list.RPush("c")
	list.RPush("d")
	list.RPush("e")

	fmt.Println("Range", printNode(list.Range(0, -1)))
	fmt.Println("Range", printNode(list.Range(0, 2)))
	fmt.Println("Range", printNode(list.Range(0, 4)))
	fmt.Println("Range", printNode(list.Range(2, 3)))

	list.LPush("a")
	list.LPush("b")
	list.LPush("c")
	list.LPush("d")
	list.LPush("e")

	fmt.Println("Range", printNode(list.Range(0, 10)))
}
