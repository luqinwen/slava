package sortedset

import (
	"reflect"
	"testing"
)

func TestAdd(t *testing.T) {
	testSet := Make()
	elements := []*Element{&Element{"a", 10}, &Element{"b", 20}, &Element{"c", 30}}
	for _, v := range elements {
		testSet.Add(v.Member, v.Score)
	}

	for i := 0; i < int(testSet.Len()); i++ {
		member, ok := testSet.Get(elements[i].Member)
		if !ok || !reflect.DeepEqual(member, elements[i]) {
			t.Error("Get incorrect element")
		}
	}
}

func TestLen(t *testing.T) {
	testSet := Make()
	elements := []*Element{&Element{"a", 10}, &Element{"b", 20}, &Element{"c", 30}}

	testSet.Add("a", 100)
	for _, v := range elements {
		testSet.Add(v.Member, v.Score)
	}

	if testSet.Len() != 3 {
		t.Error(":Length error")
	}
}

func TestGet(t *testing.T) {
	testSet := Make()
	testSet.Add("a", 100)

	member1, ok1 := testSet.Get("a")
	member2, ok2 := testSet.Get("b")

	if !ok1 || !reflect.DeepEqual(member1, &Element{"a", 100}) {
		t.Error(":Element is in the set but cannot get")
	}
	if ok2 || member2 != nil {
		t.Error("Element is not in the set but can get")
	}
}

func TestRemove(t *testing.T) {
	testSet := Make()
	elements := []*Element{&Element{"a", 10}, &Element{"b", 20}, &Element{"c", 30}}
	for _, v := range elements {
		testSet.Add(v.Member, v.Score)
	}

	testSet.Remove("b")

	element, ok := testSet.Get("b")

	if ok || element != nil {
		t.Error("Remove wrong")
	}
}

func TestGetRank(t *testing.T) {
	testSet := Make()
	elements := []*Element{&Element{"a", 10}, &Element{"b", 20}, &Element{"c", 30}}

	for _, v := range elements {
		testSet.Add(v.Member, v.Score)
	}

	for i := 0; i < int(testSet.Len()); i++ {
		if int(testSet.GetRank(elements[i].Member, false)) != i {
			t.Errorf("Get the wrong rank")
		}
	}
}

func TestForEach(t *testing.T) {
	testSet1 := Make()
	var scores []float64
	elements := []*Element{&Element{"a", 10}, &Element{"b", 20}, &Element{"c", 30}}

	for _, v := range elements {
		testSet1.Add(v.Member, v.Score)
	}

	testSet1.ForEach(0, 3, false, func(element *Element) bool {
		scores = append(scores, element.Score)
		return true
	})

	if !reflect.DeepEqual(scores, []float64{10, 20, 30}) {
		t.Error("Foreach error")
	}
}

func TestRange(t *testing.T) {
	testSet1 := Make()
	elements := []*Element{&Element{"a", 10}, &Element{"b", 20}, &Element{"c", 30}}

	for _, v := range elements {
		testSet1.Add(v.Member, v.Score)
	}

	testSet2 := testSet1.Range(1, 3, false)

	if reflect.DeepEqual([]*Element{&Element{"b", 20}, &Element{"c", 30}}, testSet2) != true {
		t.Error("Range error")
	}
}

func TestCount(t *testing.T) {
	testSet1 := Make()
	elements := []*Element{&Element{"a", 10}, &Element{"b", 20}, &Element{"c", 30}}

	for _, v := range elements {
		testSet1.Add(v.Member, v.Score)
	}

	if testSet1.Count(&ScoreBorder{0, 20, false}, &ScoreBorder{0, 30, false}) != 2 {
		t.Error("Count error")
	}
}

func TestForEachByScore(t *testing.T) {
	testSet1 := Make()
	var scores []float64
	elements := []*Element{&Element{"a", 10}, &Element{"b", 20}, &Element{"c", 30}}

	for _, v := range elements {
		testSet1.Add(v.Member, v.Score)
	}

	testSet1.ForEachByScore(
		&ScoreBorder{0, 20, false}, &ScoreBorder{0, 30, false},
		0, 3, false, func(element *Element) bool {
			scores = append(scores, element.Score)
			return true
		})

	if reflect.DeepEqual(scores, []float64{20, 30}) != true {
		t.Error("ForEachByScore error")
	}
}

func TestRangeByScore(t *testing.T) {
	testSet1 := Make()
	elements := []*Element{&Element{"a", 10}, &Element{"b", 20}, &Element{"c", 30}}

	for _, v := range elements {
		testSet1.Add(v.Member, v.Score)
	}

	testSet2 := testSet1.RangeByScore(
		&ScoreBorder{0, 20, false}, &ScoreBorder{0, 30, false},
		0, 3, false)

	if reflect.DeepEqual([]*Element{&Element{"b", 20}, &Element{"c", 30}}, testSet2) != true {
		t.Error("RangeByScore error")
	}
}

func TestRemoveByScore(t *testing.T) {
	testSet := Make()
	elements := []*Element{&Element{"a", 10}, &Element{"b", 20}, &Element{"c", 30}}
	for _, v := range elements {
		testSet.Add(v.Member, v.Score)
	}

	testSet.RemoveByScore(&ScoreBorder{0, 20, false}, &ScoreBorder{0, 25, false})
	element, ok := testSet.Get("b")

	if ok || element != nil {
		t.Error("RemoveByScore wrong")
	}
}

func TestPopMin(t *testing.T) {
	testSet1 := Make()
	elements := []*Element{&Element{"a", 10}, &Element{"b", 20}, &Element{"c", 30}}
	for _, v := range elements {
		testSet1.Add(v.Member, v.Score)
	}

	testSet2 := testSet1.PopMin(2)

	if reflect.DeepEqual([]*Element{&Element{"a", 10}, &Element{"b", 20}}, testSet2) != true {
		t.Error("PopMin wrong")
	}
}

func TestRemoveByRank(t *testing.T) {
	testSet := Make()
	elements := []*Element{&Element{"a", 10}, &Element{"b", 20}, &Element{"c", 30}}
	for _, v := range elements {
		testSet.Add(v.Member, v.Score)
	}

	if testSet.RemoveByRank(0, 2) != 2 {
		t.Error("RemoveByRank wrong")
	}
}
