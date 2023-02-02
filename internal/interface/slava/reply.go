package slava

// Reply is the interface of slava serialization protocol message
type Reply interface {
	ToBytes() []byte
}
