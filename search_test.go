package hitomi

import (
	"testing"
)

var search *Search

func TestSearch_SearchNode(t *testing.T) {
	node, err := search.NodeByAddress("global", 0)
	if err != nil {
		t.Fatal(err)
	}
	data, err := search.SearchNode("global", HashTerm("big"), node)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(data)
	tags, err := search.TagSuggestionData("global", data)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(tags)
}
