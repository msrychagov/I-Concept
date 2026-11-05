// Package valuebank holds «Онтограф ценностей 1.1»: the graph of the
// traditional spiritual and moral values of decree № 809 by Ольга И., in the
// form its coding rules take (the file «nodes for the coder»): the five ideas,
// the values, the notions that unfold them, the antivalues and how both show
// in everyday life. The file is a word for word copy of Вики/Онтограф
// ценностей 1.1/онтограф_ценности1.1_nodes.json; the test compares them.
package valuebank

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
)

//go:embed ontograph-1.1.json
var file []byte

// The classes of the nodes. A text is marked with the coder classes only; the
// ideas are counted from their values and the threats are not in the file.
const (
	Idea         = "core"
	Value        = "value"
	Aspect       = "aspect"
	Antivalue    = "antivalue"
	Everyday     = "everyday"
	EverydayAnti = "everyday_anti"
)

// CoderClasses are the classes a mark may name.
var CoderClasses = []string{Value, Aspect, Antivalue, Everyday, EverydayAnti}

// Node is a node of the graph as the file for the coder has it.
type Node struct {
	ID    string `json:"id"`
	Class string `json:"class"`
	Label string `json:"label"`
	// Home is the idea of a value, a notion or an everyday value; nil for the
	// antivalues and their everyday forms.
	Home *string `json:"home"`
	// ParentValue is the value a notion unfolds (one, as the coding rules say).
	ParentValue string `json:"parent_value"`
	// ParentRef is the notion an everyday value is close to, or the antivalue
	// an everyday antivalue shows.
	ParentRef string `json:"parent_ref"`
	// OpposedTo is the value an antivalue overturns, and back.
	OpposedTo  string `json:"opposed_to"`
	Definition string `json:"definition"`
}

// Ontograph is the graph read from the file.
type Ontograph struct {
	Version string
	Nodes   []Node
	byID    map[string]int
}

// Load reads the embedded graph and checks what the counting stands on: every
// notion has a value, every everyday value a notion, every everyday antivalue
// an antivalue, every value an idea and every antivalue a value.
func Load() (*Ontograph, error) {
	var raw struct {
		Version string `json:"version"`
		Nodes   []Node `json:"nodes"`
	}
	if err := json.Unmarshal(file, &raw); err != nil {
		return nil, fmt.Errorf("valuebank: %w", err)
	}
	o := &Ontograph{Version: raw.Version, Nodes: raw.Nodes, byID: map[string]int{}}
	for i, node := range o.Nodes {
		if _, twice := o.byID[node.ID]; twice || node.ID == "" {
			return nil, fmt.Errorf("valuebank: node %q is not unique", node.ID)
		}
		o.byID[node.ID] = i
	}
	is := func(id, class string) bool {
		node, ok := o.Node(id)
		return ok && node.Class == class
	}
	for _, node := range o.Nodes {
		var bad bool
		switch node.Class {
		case Value:
			bad = node.Home == nil || !is(*node.Home, Idea) || !is(node.OpposedTo, Antivalue)
		case Aspect:
			bad = !is(node.ParentValue, Value)
		case Antivalue:
			bad = !is(node.OpposedTo, Value)
		case Everyday:
			bad = !is(node.ParentRef, Aspect)
		case EverydayAnti:
			bad = !is(node.ParentRef, Antivalue)
		case Idea:
		default:
			bad = true
		}
		if bad {
			return nil, fmt.Errorf("valuebank: node %q of class %q has no place in the counting", node.ID, node.Class)
		}
	}
	return o, nil
}

// Node finds a node by its id.
func (o *Ontograph) Node(id string) (Node, bool) {
	i, ok := o.byID[id]
	if !ok {
		return Node{}, false
	}
	return o.Nodes[i], true
}

// Of returns the nodes of a class in the order of the file.
func (o *Ontograph) Of(class string) []Node {
	list := []Node{}
	for _, node := range o.Nodes {
		if node.Class == class {
			list = append(list, node)
		}
	}
	return list
}

// Coded says whether a mark may name the node.
func (o *Ontograph) Coded(id string) bool {
	node, ok := o.Node(id)
	return ok && slices.Contains(CoderClasses, node.Class)
}

// Counted tells what a mark of the node counts for: the value or the antivalue
// and whether it is an everyday form, which the coding rules add apart from
// the main count. A notion counts for its value, an everyday value for the
// value of its notion, an everyday antivalue for its antivalue.
func (o *Ontograph) Counted(id string) (target string, everyday bool) {
	node, ok := o.Node(id)
	if !ok {
		return "", false
	}
	switch node.Class {
	case Value, Antivalue:
		return node.ID, false
	case Aspect:
		return node.ParentValue, false
	case Everyday:
		notion, _ := o.Node(node.ParentRef)
		return notion.ParentValue, true
	case EverydayAnti:
		return node.ParentRef, true
	}
	return "", false
}

// IdeaOf is the idea of a value, or of the value an antivalue overturns.
func (o *Ontograph) IdeaOf(id string) string {
	node, ok := o.Node(id)
	if ok && node.Class == Antivalue {
		node, ok = o.Node(node.OpposedTo)
	}
	if !ok || node.Home == nil {
		return ""
	}
	return *node.Home
}
