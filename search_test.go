package hitomi

import (
	"testing"
)

var search *Search

func TestSearch_SearchNode(t *testing.T) {
	node, err := search.nodeByAddress("female", 0)
	if err != nil {
		t.Fatal(err)
	}
	data, err := search.searchNode("female", HashTerm("big"), node)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(data)
	tags, err := search.tagSuggestionData("female", data)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(tags)
}

func TestSearch_TagSuggestion(t *testing.T) {
	result, err := search.TagSuggestion("tag:")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(result)
}

func TestSearch_TagSuggestion_CacheWholeIndex(t *testing.T) {
	csc := NewSearch(DefaultOptions().WithCacheWholeIndex(true))
	result, err := csc.TagSuggestion("female:big")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(result)
}

func BenchmarkSearch_TagSuggestion_CacheWholeIndex(b *testing.B) {
	b.StopTimer()
	csc := NewSearch(DefaultOptions().WithCacheWholeIndex(true))
	_, _ = csc.TagSuggestion("female:")
	b.Log("warmup done")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, err := csc.TagSuggestion("female:big")
		if err != nil {
			b.Fatal(err)
		}
	}
}
