package stream

type radixTree struct {
	root *radixNode
}

type radixNode struct {
	key      string
	value    *ListPack
	isLeaf   bool
	children map[string]*radixNode
}

func newRadixNode() *radixNode {
	return &radixNode{
		children: make(map[string]*radixNode),
	}
}

func newRadixTree() *radixTree {
	return &radixTree{
		root: newRadixNode(),
	}
}

func (t *radixTree) Insert(key string, value *ListPack) {
	node := t.root
	var i int
	for ; i < len(key); i++ {
		child, ok := node.children[string(key[i])]
		if !ok {
			break
		}
		node = child
	}

	for ; i < len(key); i++ {
		child := newRadixNode()
		child.key = key[i:]
		node.children[string(key[i])] = child
		node = child
	}

	node.isLeaf = true
	node.value = value
}

func (t *radixTree) Search(key string) (*ListPack, bool) {
	node := t.root
	var i int
	for ; i < len(key); i++ {
		child, ok := node.children[string(key[i])]
		if !ok {
			break
		}
		node = child
	}

	if i == len(key) && node.isLeaf {
		return node.value, true
	}
	return nil, false
}

func (t *radixTree) Delete(key string) {
	t.root.delete(key)
}

func (n *radixNode) delete(key string) bool {
	if len(key) == 0 {
		if !n.isLeaf {
			return false
		}
		n.isLeaf = false
		n.value = nil
		return len(n.children) == 0
	}
	child, ok := n.children[string(key[0])]
	if !ok {
		return false
	}

	if child.delete(key[1:]) {
		delete(n.children, string(key[0]))
		return len(n.children) == 0 && !n.isLeaf
	}
	return false
}

// Walk calls the function f with the key and value of each item in the tree
func (t *radixTree) Walk(f func(key string, value *ListPack) bool) {
	t.root.walk(f)
}

func (n *radixNode) walk(f func(key string, value *ListPack) bool) bool {
	if n.isLeaf {
		if !f(n.key, n.value) {
			return false
		}
	}
	for _, child := range n.children {
		if !child.walk(f) {
			return false
		}
	}
	return true
}
