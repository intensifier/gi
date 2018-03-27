package compiler

import (
	"fmt"
	"io"

	"github.com/glycerine/gi/pkg/types"
)

//  depend.go
//
//  Implement Depth-First-Search (DFS)
//  on the graph of depedencies
//  between types. A pre-order
//  traversal will print
//  leaf types before the compound
//  types that need them defined.

var dependTestMode = false

func isBasicTyp(n *dfsNode) bool {
	_, ok := n.typ.(*types.Basic)
	return ok
}

type dfsNode struct {
	id            int
	name          string
	typ           types.Type
	stale         bool
	made          bool
	children      []*dfsNode
	dedupChildren map[*dfsNode]bool
	visited       bool

	createCode []byte
}

func (me *dfsNode) bloom(w io.Writer) {
	_, err := w.Write(me.createCode)
	panicOn(err)
}

// a func on nodes to force instantiation of
// any types this node depends on, i.e. those
// types (not values) that were described but
// lazily instantated. Calls me.typ.bloom
// on our subtree in depth-first order.
//
func (me *dfsNode) makeRequiredTypes(w io.Writer) {
	if me.made {
		return
	}
	me.made = true
	if isBasicTyp(me) {
		return // basic types are always leaf nodes, no children.
	}

	for _, ch := range me.children {
		ch.makeRequiredTypes(w)
	}
	me.bloom(w)
}

func (s *dfsState) newDfsNode(name string, typ types.Type, createCode []byte) *dfsNode {
	if typ == nil {
		panic("typ cannot be nil in newDfsNode")
	}

	nd, ok := s.dfsDedup[typ]
	if ok {
		return nd
	}

	node := &dfsNode{
		id:            s.dfsNextID,
		name:          name,
		typ:           typ,
		stale:         true,
		dedupChildren: make(map[*dfsNode]bool),
		createCode:    createCode,
	}
	s.dfsNextID++
	s.dfsDedup[typ] = node
	s.dfsNodes = append(s.dfsNodes, node)

	return node
}

// par should be a node; e.g. typ.dfsNode
func (s *dfsState) addChild(par, ch *dfsNode) {

	if par == nil {
		panic("par cannot be nil in addChild")
	}
	if ch == nil {
		panic("ch cannot be nil in addChild")
	}

	// we can skip all basic types,
	// as they are already defined.
	if !dependTestMode {
		if isBasicTyp(ch) {
			return
		}
		if isBasicTyp(par) {
			panic(fmt.Sprintf("addChild error: parent was basic type. "+
				"cannot add child to basic typ %v", par))
		}
	}

	// dedup happens in newDfsNode now.
	/*
		prior, present := s.dfsDedup[ch.typ]
		if present {
			// child was previously generated, so
			// we don't need to worry about this
			// dependency
			vv("dedup child, previously seen: %s was seen as %s", ch.name, prior.name)
			return
		}
	*/
	if par.dedupChildren[ch] {
		vv("avoid adding same child twice to a parent.")
		return
	}

	par.children = append(par.children, ch)
	par.dedupChildren[ch] = true
	s.stale = true
}

func (s *dfsState) markGraphUnVisited() {
	s.dfsOrder = []*dfsNode{}
	for _, n := range s.dfsNodes {
		n.visited = false
	}
	s.stale = false
}

func (me *dfsState) reset() {
	// empty the graph
	me.dfsOrder = []*dfsNode{}
	me.dfsNodes = []*dfsNode{}              // node stored in value.
	me.dfsDedup = map[types.Type]*dfsNode{} // payloadTyp key -> node value.
	me.dfsNextID = 0
	me.stale = false
}

func (s *dfsState) dfsHelper(node *dfsNode) {
	if node == nil {
		return
	}
	if node.visited {
		return
	}
	node.visited = true

	for _, ch := range node.children {
		s.dfsHelper(ch)
	}

	vv("post-order visit sees node %v : %v", node.id, node.name)
	s.dfsOrder = append(s.dfsOrder, node)

}

func (s *dfsState) showDFSOrder() {
	if s.stale {
		s.doDFS()
	}
	for i, n := range s.dfsOrder {
		vv("dfs order %v is %v : %v", i, n.id, n.name)
	}
}

func (s *dfsState) doDFS() {
	s.markGraphUnVisited()
	for _, n := range s.dfsNodes {
		s.dfsHelper(n)
	}
	s.stale = false
}

func (s *dfsState) hasTypes() bool {
	return s.dfsNextID != 0
}

type dfsState struct {
	dfsNodes  []*dfsNode
	dfsOrder  []*dfsNode
	dfsDedup  map[types.Type]*dfsNode
	dfsNextID int
	stale     bool
}

func NewDFSState() *dfsState {
	return &dfsState{
		dfsNodes: []*dfsNode{},
		dfsOrder: []*dfsNode{},
		dfsDedup: make(map[types.Type]*dfsNode),
	}
}
