package string

import (
	"reflect"
	"slava/internal/protocol"
	"slava/internal/utils"
	db "slava/pkg/database"
	"testing"
)

func TestExecGet(t *testing.T) {
	testDB := db.MakeDB()
	testData := make([][]byte, 2)
	testData[0] = []byte("Hello")
	testData[1] = []byte("World")
	execSet(testDB, testData)

	testSearch := make([][]byte, 1)
	testSearch[0] = testData[0]

	if !utils.BytesEquals(execGet(testDB, testSearch).ToBytes(), protocol.MakeBulkReply(testData[1]).ToBytes()) {
		t.Errorf("Get value incorrect")
	}
}

func TestExecGetEX(t *testing.T) {
	testDB := db.MakeDB()
	testData := make([][]byte, 2)
	testData[0] = []byte("Hello")
	testData[1] = []byte("World")
	execSet(testDB, testData)

	testSearch := make([][]byte, 3)
	testSearch[0] = []byte("Hello")
	testSearch[1] = []byte("EX")
	testSearch[2] = []byte("60")

	if !utils.BytesEquals(execGet(testDB, testSearch).ToBytes(), protocol.MakeBulkReply(testData[1]).ToBytes()) {
		t.Errorf("Get value incorrect")
	}

	if testDB.IsExpired(string(testSearch[0])) {
		t.Errorf("don't set the expire time")
	}
}

func TestExecSetNX(t *testing.T) {
	testDB := db.MakeDB()
	testData1 := make([][]byte, 2)
	testData1[0] = []byte("Hello")
	testData1[1] = []byte("World")
	execSet(testDB, testData1)

	testData2 := make([][]byte, 2)
	testData2[0] = []byte("Test")
	testData2[1] = []byte("Data")

	if !utils.BytesEquals(execSetNX(testDB, testData1).ToBytes(), protocol.MakeIntReply(0).ToBytes()) {
		t.Errorf("Existed data but doesn't return 0")
	}

	if !utils.BytesEquals(execSetNX(testDB, testData2).ToBytes(), protocol.MakeIntReply(1).ToBytes()) {
		t.Errorf("No existed data but doesn't return 1")
	}
}

func TestExecSetEX(t *testing.T) {
	testDB := db.MakeDB()
	testData := make([][]byte, 3)
	testData[0] = []byte("Hello")
	testData[1] = []byte("60")
	testData[2] = []byte("World")

	execSetEX(testDB, testData)

	testSearch := make([][]byte, 1)
	testSearch[0] = testData[0]

	if !utils.BytesEquals(execGet(testDB, testSearch).ToBytes(), protocol.MakeBulkReply(testData[2]).ToBytes()) {
		t.Errorf("Get value incorrect")
	}

	if testDB.IsExpired(string(testSearch[0])) {
		t.Errorf("don't set the expire time")
	}
}

func TestExecPSetEX(t *testing.T) {
	testDB := db.MakeDB()
	testData := make([][]byte, 3)
	testData[0] = []byte("Hello")
	testData[1] = []byte("6000")
	testData[2] = []byte("World")

	execPSetEX(testDB, testData)

	testSearch := make([][]byte, 1)
	testSearch[0] = testData[0]

	if !utils.BytesEquals(execGet(testDB, testSearch).ToBytes(), protocol.MakeBulkReply(testData[2]).ToBytes()) {
		t.Errorf("Get value incorrect")
	}

	if testDB.IsExpired(string(testSearch[0])) {
		t.Errorf("don't set the expire time")
	}
}

func TestPrepareMSet(t *testing.T) {
	testData := make([][]byte, 4)
	testData[0] = []byte("Hello")
	testData[1] = []byte("World")
	testData[2] = []byte("Test")
	testData[3] = []byte("Data")

	key, value := prepareMSet(testData)

	if !reflect.DeepEqual(key, []string{"Hello", "Test"}) || value != nil {
		t.Errorf("Prepare doesn't succeed")
	}
}

func TestExecMSet(t *testing.T) {
	testDB := db.MakeDB()
	testData := make([][]byte, 4)
	testData[0] = []byte("Hello")
	testData[1] = []byte("World")
	testData[2] = []byte("Test")
	testData[3] = []byte("Data")

	execMSet(testDB, testData)

	testSearch1 := make([][]byte, 1)
	testSearch1[0] = testData[0]
	testSearch2 := make([][]byte, 1)
	testSearch2[0] = testData[2]

	if !utils.BytesEquals(execGet(testDB, testSearch1).ToBytes(), protocol.MakeBulkReply(testData[1]).ToBytes()) {
		t.Errorf("ExecMSet wrong")
	}

	if !utils.BytesEquals(execGet(testDB, testSearch2).ToBytes(), protocol.MakeBulkReply(testData[3]).ToBytes()) {
		t.Errorf("ExecMSet wrong")
	}
}

func TestExecMGet(t *testing.T) {
	testDB := db.MakeDB()
	testData := make([][]byte, 4)
	testData[0] = []byte("Hello")
	testData[1] = []byte("World")
	testData[2] = []byte("Test")
	testData[3] = []byte("Data")

	execMSet(testDB, testData)

	testSearch := make([][]byte, 2)
	testSearch[0] = testData[0]
	testSearch[1] = testData[2]
	expectAns := make([][]byte, 2)
	expectAns[0] = testData[1]
	expectAns[1] = testData[3]

	if !utils.BytesEquals(execMGet(testDB, testSearch).ToBytes(), protocol.MakeMultiBulkReply(expectAns).ToBytes()) {
		t.Errorf("ExecMGet wrong")
	}
}

func TestExecMSetNX(t *testing.T) {
	testDB := db.MakeDB()
	testData := make([][]byte, 2)
	testData[0] = []byte("Hello")
	testData[1] = []byte("World")
	execSet(testDB, testData)

	testSearch1 := make([][]byte, 4)
	testSearch1[0] = testData[0]
	testSearch1[1] = testData[1]
	testSearch1[2] = []byte("Test")
	testSearch1[3] = []byte("Data")

	testSearch2 := make([][]byte, 4)
	testSearch2[0] = []byte("Test")
	testSearch2[1] = []byte("Data")
	testSearch2[2] = []byte("MSet")
	testSearch2[3] = []byte("NX")

	if !utils.BytesEquals(execMSetNX(testDB, testSearch1).ToBytes(), protocol.MakeIntReply(0).ToBytes()) {
		t.Errorf("Existed data but doesn't return 0")
	}

	if !utils.BytesEquals(execMSetNX(testDB, testSearch2).ToBytes(), protocol.MakeIntReply(1).ToBytes()) {
		t.Errorf("No existed data but doesn't return 1")
	}
}

func TestExecGetSet(t *testing.T) {
	testDB := db.MakeDB()
	testData := make([][]byte, 2)
	testData[0] = []byte("Hello")
	testData[1] = []byte("World")
	execSet(testDB, testData)

	testSearch := make([][]byte, 2)
	testSearch[0] = testData[0]
	testSearch[1] = []byte("Golang")

	if !utils.BytesEquals(execGetSet(testDB, testSearch).ToBytes(), protocol.MakeBulkReply(testData[1]).ToBytes()) {
		t.Errorf("Return old value incorrect")
	}

	if !utils.BytesEquals(execGet(testDB, testSearch).ToBytes(), protocol.MakeBulkReply(testSearch[1]).ToBytes()) {
		t.Errorf("New value incorrect")
	}
}

func TestExecGetDel(t *testing.T) {
	testDB := db.MakeDB()
	testData := make([][]byte, 2)
	testData[0] = []byte("Hello")
	testData[1] = []byte("World")
	execSet(testDB, testData)

	testSearch := make([][]byte, 1)
	testSearch[0] = testData[0]

	if !utils.BytesEquals(execGetDel(testDB, testSearch).ToBytes(), protocol.MakeBulkReply(testData[1]).ToBytes()) {
		t.Errorf("Get value incorrect")
	}

	if !utils.BytesEquals(execGetDel(testDB, testSearch).ToBytes(), protocol.MakeNullBulkReply().ToBytes()) {
		t.Errorf("Key not deleted")
	}
}

func TestExecIncr(t *testing.T) {
	testDB := db.MakeDB()
	testData := make([][]byte, 4)
	testData[0] = []byte("K1")
	testData[1] = []byte("1")
	testData[2] = []byte("K2")
	testData[3] = []byte("10.5")
	execMSet(testDB, testData)

	testSearch1 := make([][]byte, 1)
	testSearch1[0] = testData[0]
	testSearch2 := make([][]byte, 1)
	testSearch2[0] = testData[2]
	testSearch3 := make([][]byte, 1)
	testSearch3[0] = []byte("K3")

	if !utils.BytesEquals(execIncr(testDB, testSearch1).ToBytes(), protocol.MakeIntReply(2).ToBytes()) {
		t.Errorf("Value is a number but don't add 1")
	}
	if !utils.BytesEquals(execIncr(testDB, testSearch2).ToBytes(),
		protocol.MakeErrReply("ERR value is not an integer or out of range").ToBytes()) {
		t.Errorf("Value is not an interger but don't get correct error message")
	}
	if !utils.BytesEquals(execIncr(testDB, testSearch3).ToBytes(), protocol.MakeIntReply(1).ToBytes()) {
		t.Errorf("Key doesn't exist but the added value is not 1")
	}
}

func TestExecIncrBy(t *testing.T) {
	testDB := db.MakeDB()
	testData := make([][]byte, 4)
	testData[0] = []byte("K1")
	testData[1] = []byte("1")
	testData[2] = []byte("K2")
	testData[3] = []byte("10.5")
	execMSet(testDB, testData)

	testSearch1 := make([][]byte, 2)
	testSearch1[0] = testData[0]
	testSearch1[1] = []byte("10")
	testSearch2 := make([][]byte, 2)
	testSearch2[0] = testData[2]
	testSearch2[1] = []byte("20.5")
	testSearch3 := make([][]byte, 2)
	testSearch3[0] = []byte("K3")
	testSearch3[1] = []byte("30")

	if !utils.BytesEquals(execIncrBy(testDB, testSearch1).ToBytes(), protocol.MakeIntReply(11).ToBytes()) {
		t.Errorf("Value is a number but don't add the given number correctly")
	}
	if !utils.BytesEquals(execIncrBy(testDB, testSearch2).ToBytes(),
		protocol.MakeErrReply("ERR value is not an integer or out of range").ToBytes()) {
		t.Errorf("Value is not an interger but don't get correct error message")
	}
	if !utils.BytesEquals(execIncrBy(testDB, testSearch3).ToBytes(), protocol.MakeIntReply(30).ToBytes()) {
		t.Errorf("Key doesn't exist but the added value is not the given value")
	}
}

func TestExecIncrByFloat(t *testing.T) {
	testDB := db.MakeDB()
	testData := make([][]byte, 4)
	testData[0] = []byte("K1")
	testData[1] = []byte("1")
	testData[2] = []byte("K2")
	testData[3] = []byte("10.5")
	execMSet(testDB, testData)

	testSearch1 := make([][]byte, 2)
	testSearch1[0] = testData[0]
	testSearch1[1] = []byte("10.5")
	testSearch2 := make([][]byte, 2)
	testSearch2[0] = testData[2]
	testSearch2[1] = []byte("A")
	testSearch3 := make([][]byte, 2)
	testSearch3[0] = []byte("K3")
	testSearch3[1] = []byte("30.5")

	if !utils.BytesEquals(execIncrByFloat(testDB, testSearch1).ToBytes(),
		protocol.MakeBulkReply([]byte("11.5")).ToBytes()) {
		t.Errorf("Value is a number but don't add the given number correctly")
	}
	if !utils.BytesEquals(execIncrByFloat(testDB, testSearch2).ToBytes(),
		protocol.MakeErrReply("ERR value is not a valid float").ToBytes()) {
		t.Errorf("Value is not a float but don't get correct error message")
	}
	if !utils.BytesEquals(execIncrByFloat(testDB, testSearch3).ToBytes(),
		protocol.MakeBulkReply(testSearch3[1]).ToBytes()) {
		t.Errorf("Key doesn't exist but the added value is not the given value")
	}
}

func TestExecDecr(t *testing.T) {
	testDB := db.MakeDB()
	testData := make([][]byte, 4)
	testData[0] = []byte("K1")
	testData[1] = []byte("1")
	testData[2] = []byte("K2")
	testData[3] = []byte("10.5")
	execMSet(testDB, testData)

	testSearch1 := make([][]byte, 1)
	testSearch1[0] = testData[0]
	testSearch2 := make([][]byte, 1)
	testSearch2[0] = testData[2]
	testSearch3 := make([][]byte, 1)
	testSearch3[0] = []byte("K3")

	if !utils.BytesEquals(execDecr(testDB, testSearch1).ToBytes(), protocol.MakeIntReply(0).ToBytes()) {
		t.Errorf("Value is a number but don't minus 1")
	}
	if !utils.BytesEquals(execDecr(testDB, testSearch2).ToBytes(),
		protocol.MakeErrReply("ERR value is not an integer or out of range").ToBytes()) {
		t.Errorf("Value is not an interger but don't get correct error message")
	}
	if !utils.BytesEquals(execDecr(testDB, testSearch3).ToBytes(), protocol.MakeIntReply(-1).ToBytes()) {
		t.Errorf("Key doesn't exist but the added value is not -1")
	}
}

func TestExecDecrBy(t *testing.T) {
	testDB := db.MakeDB()
	testData := make([][]byte, 4)
	testData[0] = []byte("K1")
	testData[1] = []byte("1")
	testData[2] = []byte("K2")
	testData[3] = []byte("10.5")
	execMSet(testDB, testData)

	testSearch1 := make([][]byte, 2)
	testSearch1[0] = testData[0]
	testSearch1[1] = []byte("10")
	testSearch2 := make([][]byte, 2)
	testSearch2[0] = testData[2]
	testSearch2[1] = []byte("20.5")
	testSearch3 := make([][]byte, 2)
	testSearch3[0] = []byte("K3")
	testSearch3[1] = []byte("30")

	if !utils.BytesEquals(execDecrBy(testDB, testSearch1).ToBytes(), protocol.MakeIntReply(-9).ToBytes()) {
		t.Errorf("Value is a number but don't minus the given number correctly")
	}
	if !utils.BytesEquals(execDecrBy(testDB, testSearch2).ToBytes(),
		protocol.MakeErrReply("ERR value is not an integer or out of range").ToBytes()) {
		t.Errorf("Value is not an interger but don't get correct error message")
	}
	if !utils.BytesEquals(execDecrBy(testDB, testSearch3).ToBytes(), protocol.MakeIntReply(-30).ToBytes()) {
		t.Errorf("Key doesn't exist but the added value is not the opposite of the given value")
	}
}

func TestExecStrLen(t *testing.T) {
	testDB := db.MakeDB()
	testData := make([][]byte, 2)
	testData[0] = []byte("Hello")
	testData[1] = []byte("World")
	execSet(testDB, testData)

	testSearch1 := make([][]byte, 1)
	testSearch1[0] = testData[0]
	testSearch2 := make([][]byte, 1)
	testSearch2[0] = []byte("Test")

	if !utils.BytesEquals(execStrLen(testDB, testSearch1).ToBytes(), protocol.MakeIntReply(5).ToBytes()) {
		t.Errorf("Value exists but length is wrong")
	}
	if !utils.BytesEquals(execStrLen(testDB, testSearch2).ToBytes(), protocol.MakeIntReply(0).ToBytes()) {
		t.Errorf("Value doesn't exist but length is not 0")
	}
}

func TestExecAppend(t *testing.T) {
	testDB := db.MakeDB()
	testData := make([][]byte, 2)
	testData[0] = []byte("Hello")
	testData[1] = []byte("World")
	execSet(testDB, testData)

	testSearch := make([][]byte, 2)
	testSearch[0] = testData[0]
	testSearch[1] = []byte("Golang")

	if !utils.BytesEquals(execAppend(testDB, testSearch).ToBytes(), protocol.MakeIntReply(11).ToBytes()) {
		t.Errorf("Appended length wrong")
	}
	if !utils.BytesEquals(execGet(testDB, testSearch).ToBytes(), protocol.MakeBulkReply([]byte("WorldGolang")).ToBytes()) {
		t.Errorf("Appended value wrong")
	}
}

func TestExecSetRange(t *testing.T) {
	testDB := db.MakeDB()
	testData := make([][]byte, 2)
	testData[0] = []byte("Hello")
	testData[1] = []byte("World")
	execSet(testDB, testData)

	testSearch := make([][]byte, 3)
	testSearch[0] = testData[0]
	testSearch[1] = []byte("5")
	testSearch[2] = []byte("Golang")

	if !utils.BytesEquals(execSetRange(testDB, testSearch).ToBytes(), protocol.MakeIntReply(11).ToBytes()) {
		t.Errorf("Length wrong")
	}
	if !utils.BytesEquals(execGet(testDB, testSearch).ToBytes(), protocol.MakeBulkReply([]byte("WorldGolang")).ToBytes()) {
		t.Errorf("Value wrong")
	}
}

func TestExEcGetRange(t *testing.T) {
	testDB := db.MakeDB()
	testData := make([][]byte, 2)
	testData[0] = []byte("Hello")
	testData[1] = []byte("World")
	execSet(testDB, testData)

	testSearch1 := make([][]byte, 3)
	testSearch1[0] = testData[0]
	testSearch1[1] = []byte("1")
	testSearch1[2] = []byte("3")
	testSearch2 := make([][]byte, 3)
	testSearch2[0] = testData[0]
	testSearch2[1] = []byte("1")
	testSearch2[2] = []byte("3.5")
	testSearch3 := make([][]byte, 3)
	testSearch3[0] = []byte("Test")
	testSearch3[1] = []byte("1")
	testSearch3[2] = []byte("3")

	if !utils.BytesEquals(execGetRange(testDB, testSearch1).ToBytes(), protocol.MakeBulkReply([]byte("orl")).ToBytes()) {
		t.Errorf("Get substring is not correct")
	}
	if !utils.BytesEquals(execGetRange(testDB, testSearch2).ToBytes(),
		protocol.MakeErrReply("ERR value is not an integer or out of range").ToBytes()) {
		t.Errorf("Error value not detect")
	}
	if !utils.BytesEquals(execGetRange(testDB, testSearch3).ToBytes(), protocol.MakeNullBulkReply().ToBytes()) {
		t.Errorf("Key doesn't exist but find value")
	}
}

func TestExecSetBit(t *testing.T) {

}
