package set

import (
	"fmt"
	"reflect"
	"testing"
)

func TestAdd(t *testing.T) {
	testSet := Make()
	testSet.Add("a")
	testSet.Add("a")
	testSet.Add("b")
	testSet.Add("c")

	if testSet.Len() != 3 {
		t.Error("Add error")
	}
	fmt.Printf("expect [c b a] is %s\n", testSet.ToSlice())

	if reflect.DeepEqual(testSet.ToSlice(), []string{"c", "b", "a"}) != true {
		t.Errorf("Add %s wrong\n", "a,b,c")
	}
}

func TestRemove(t *testing.T) {
	testSet := Make()
	testSet.Add("a")
	testSet.Add("b")
	testSet.Add("c")

	fmt.Printf("expect [c b a] is %s\n", testSet.ToSlice())

	testSet.Remove("b")
	fmt.Printf("expect [c a] is %s\n", testSet.ToSlice())

	testSet.Remove("c")
	fmt.Printf("expect [a] is %s\n", testSet.ToSlice())

	if reflect.DeepEqual(testSet.ToSlice(), []string{"a"}) != true {
		t.Errorf("Remove %s wrong\n", "b,c")
	}
}

func TestHas(t *testing.T) {
	testSet := Make()
	testSet.Add("a")

	if !testSet.Has("a") {
		t.Errorf("%s is in the set but show not in\n", "a")
	}

	if testSet.Has("b") {
		t.Errorf("%s is not in the set but show in\n", "b")
	}
}

func TestLen(t *testing.T) {
	testSet := Make()
	testSet.Add("a")
	testSet.Add("b")
	testSet.Add("c")

	if testSet.Len() != 3 {
		t.Error("Add error")
	}
}

func TestToSlice(t *testing.T) {
	testSet := Make()
	testSet.Add("a")
	testSet.Add("b")
	testSet.Add("c")

	if reflect.DeepEqual(testSet.ToSlice(), []string{"c", "b", "a"}) != true {
		t.Errorf("Add %s wrong\n", "a,b,c")
	}
	fmt.Printf("expect [c b a] is %s\n", testSet.ToSlice())
}

func TestForeach(t *testing.T) {
	testSet1, testSet2 := Make(), Make()
	testSet1.Add("a")
	testSet1.Add("b")
	testSet1.Add("c")

	testSet1.ForEach(func(member string) bool {
		testSet2.Add(member)
		return true
	})

	if reflect.DeepEqual(testSet1, testSet2) != true {
		t.Error("Foreach error")
	}
	fmt.Printf("expect %s is %s\n", testSet1.ToSlice(), testSet2.ToSlice())
}

func TestIntersect(t *testing.T) {
	testSet1, testSet2 := Make(), Make()
	testSet1.Add("a")
	testSet1.Add("b")
	testSet1.Add("c")

	testSet2.Add("b")
	testSet2.Add("c")
	testSet2.Add("d")

	if reflect.DeepEqual(testSet1.Intersect(testSet2).ToSlice(), []string{"c", "b"}) != true {
		t.Error("Intersect error")
	}
	fmt.Printf("expect [c b] is %s\n", testSet1.Intersect(testSet2).ToSlice())
}

func TestUnion(t *testing.T) {
	testSet1, testSet2 := Make(), Make()
	testSet1.Add("a")
	testSet1.Add("b")
	testSet1.Add("c")

	testSet2.Add("b")
	testSet2.Add("c")
	testSet2.Add("d")

	if reflect.DeepEqual(testSet1.Union(testSet2).ToSlice(), []string{"d", "c", "b", "a"}) != true {
		t.Error("Union error")
	}
	fmt.Printf("expect [d c b a] is %s\n", testSet1.Union(testSet2).ToSlice())
}

func TestDiff(t *testing.T) {
	testSet1, testSet2 := Make(), Make()
	testSet1.Add("a")
	testSet1.Add("b")
	testSet1.Add("c")

	testSet2.Add("b")
	testSet2.Add("c")
	testSet2.Add("d")

	if reflect.DeepEqual(testSet1.Diff(testSet2).ToSlice(), []string{"a"}) != true {
		t.Error("Diff error")
	}
	fmt.Printf("expect [a] is %s\n", testSet1.Diff(testSet2).ToSlice())
}

func TestRandomMembers(t *testing.T) {
	testSet := Make()
	testSet.Add("a")
	testSet.Add("b")
	testSet.Add("c")
	rm := testSet.RandomMembers(3)

	if len(rm) != 3 {
		t.Error("Not enough members")
	}

	for _, v := range rm {
		if !testSet.Has(v) {
			t.Error("Member is not in the original set")
		}
	}
}

func TestRandomDistinctMembers(t *testing.T) {
	testSet1, testSet2 := Make(), Make()
	testSet1.Add("a")
	testSet1.Add("b")
	testSet1.Add("c")
	testSet1.Add("d")
	testSet1.Add("e")
	rdm := testSet1.RandomDistinctMembers(4)

	if len(rdm) != 4 {
		t.Error("Not enough members")
	}

	for _, v := range rdm {
		if !testSet1.Has(v) {
			t.Error("Member is not in the original set")
		}
		testSet2.Add(v)
	}

	if testSet2.Len() != 4 {
		t.Error("Exist duplicated member")
	}
}
