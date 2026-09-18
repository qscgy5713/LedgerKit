package ledger

import (
	"sort"
	"strings"

	"ledgerkit/internal/journal"
)

// Node is one segment of the ":"-separated account hierarchy.
type Node struct {
	Name       string
	FullName   string
	Children   map[string]*Node
	childOrder []string
	Own        journal.Amounts // balance posted directly to this exact account
	Total      journal.Amounts // Own + all descendants
}

func newNode(name, full string) *Node {
	return &Node{
		Name:     name,
		FullName: full,
		Children: map[string]*Node{},
		Own:      journal.NewAmounts(),
		Total:    journal.NewAmounts(),
	}
}

// SortedChildren returns this node's children ordered by account name.
func (n *Node) SortedChildren() []*Node {
	names := append([]string(nil), n.childOrder...)
	sort.Strings(names)
	out := make([]*Node, len(names))
	for i, name := range names {
		out[i] = n.Children[name]
	}
	return out
}

// BuildTree aggregates matching postings into a hierarchical balance tree.
func BuildTree(txns []journal.Transaction, f Filter) *Node {
	root := newNode("", "")
	for _, t := range txns {
		if !inRange(t.Date, f) {
			continue
		}
		for _, p := range t.Postings {
			if p.Amount == nil || !matchAccount(p.Account, f.AccountPattern) {
				continue
			}
			addToTree(root, p.Account, *p.Amount)
		}
	}
	computeTotals(root)
	return root
}

func addToTree(root *Node, account string, amt journal.Amount) {
	parts := strings.Split(account, ":")
	node := root
	full := ""
	for i, part := range parts {
		if i == 0 {
			full = part
		} else {
			full = full + ":" + part
		}
		child, ok := node.Children[part]
		if !ok {
			child = newNode(part, full)
			node.Children[part] = child
			node.childOrder = append(node.childOrder, part)
		}
		node = child
	}
	node.Own = node.Own.Add(amt)
}

// computeTotals sums each node's own balance with every descendant's,
// depth-first. Child iteration order doesn't affect the sum, so it walks
// n.Children directly rather than paying for SortedChildren's sort (that
// ordering only matters for display, in cmd_balance.go's printNode).
func computeTotals(n *Node) journal.Amounts {
	total := n.Own
	for _, child := range n.Children {
		total = total.Merge(computeTotals(child))
	}
	n.Total = total
	return total
}
