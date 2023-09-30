package hitomi

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/EINNN7/hitomi/internal/util"
)

const MaxNodeSize = 464

type Search struct {
	options *Options

	indexVersion map[string]string
	indexCache   map[string][]byte
}

func NewSearch(options *Options) *Search {
	return &Search{
		options:      options,
		indexVersion: map[string]string{},
		indexCache:   map[string][]byte{},
	}
}

func (s *Search) IndexVersion(name string) (string, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("https://ltn.hitomi.la/%s/version?_=%d", name, time.Now().UnixMilli()), nil)
	resp, err := s.options.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	version, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(version), nil
}

func (s *Search) TagSuggestion(query string) ([]string, error) {
	field := strings.Split(query, ":")
	if len(field) != 2 {
		return nil, fmt.Errorf("invalid query: %s", query)
	}
	firstNode, err := s.nodeByAddress(field[0], 0)
	if err != nil {
		return nil, err
	}
	dataOffset, err := s.searchNode(field[0], HashTerm(field[1]), firstNode)
	if err != nil {
		return nil, fmt.Errorf("cannot find search result: %w", err)
	}
	return s.tagSuggestionData(field[0], dataOffset)
}

func (s *Search) nodeByAddress(field string, address int) (*node, error) {
	var url string
	switch field {
	case "galleries":
		if _, ok := s.indexVersion["galleriesindex"]; !ok {
			s.options.Logger.Debug().Msg("galleriesindex version not found, fetch fresh one")
			version, err := s.IndexVersion("galleriesindex")
			if err != nil {
				return nil, err
			}
			s.indexVersion["galleriesindex"] = version
		}
		url = fmt.Sprintf("https://ltn.hitomi.la/galleriesindex/galleries.%s.index", s.indexVersion["galleriesindex"])
	case "languages":
		if _, ok := s.indexVersion["languagesindex"]; !ok {
			s.options.Logger.Debug().Msg("languagesindex version not found, fetch fresh one")
			version, err := s.IndexVersion("languagesindex")
			if err != nil {
				return nil, err
			}
			s.indexVersion["languagesindex"] = version
		}
		url = fmt.Sprintf("https://ltn.hitomi.la/languagesindex/languages.%s.index", s.indexVersion["languagesindex"])
	case "nozomiurl":
		if _, ok := s.indexVersion["nozomiurlindex"]; !ok {
			s.options.Logger.Debug().Msg("nozomiurlindex version not found, fetch fresh one")
			version, err := s.IndexVersion("nozomiurlindex")
			if err != nil {
				return nil, err
			}
			s.indexVersion["nozomiurlindex"] = version
		}
		url = fmt.Sprintf("https://ltn.hitomi.la/nozomiurlindex/nozomiurl.%s.index", s.indexVersion["nozomiurlindex"])
	default:
		if _, ok := s.indexVersion["tagindex"]; !ok {
			s.options.Logger.Debug().Msg("tagindex version not found, fetch fresh one")
			version, err := s.IndexVersion("tagindex")
			if err != nil {
				return nil, err
			}
			s.indexVersion["tagindex"] = version
		}
		url = fmt.Sprintf("https://ltn.hitomi.la/tagindex/%s.%s.index", field, s.indexVersion["tagindex"])
	}
	if s.options.CacheWholeIndex {
		if v, ok := s.indexCache[url]; ok {
			return decodeNode(v[address : address+MaxNodeSize-1])
		}
		s.options.Logger.Debug().Msgf("indexCache for %s not found, fetch fresh one", url)
		req, _ := http.NewRequest("GET", url, nil)
		resp, err := s.options.Client.Do(req)
		if err != nil {
			return nil, err
		}
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("failed to get node: %d", resp.StatusCode)
		}
		content, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		s.indexCache[url] = content
		return decodeNode(content[address : address+MaxNodeSize-1])
	} else {
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", address, address+MaxNodeSize-1))
		resp, err := s.options.Client.Do(req)
		if err != nil {
			return nil, err
		}
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("failed to get node: %d", resp.StatusCode)
		}
		content, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return decodeNode(content)
	}
}

type node struct {
	Key            [][]byte
	Data           [][2]int
	SubNodeAddress []int
}

func decodeNode(data []byte) (*node, error) {
	node := new(node)
	node.Key = [][]byte{}
	node.Data = [][2]int{}
	node.SubNodeAddress = []int{}

	var pos int32 = 4
	keyLength := int32(binary.BigEndian.Uint32(data[0:4]))

	for i := int32(0); i < keyLength; i++ {
		keySize := int32(binary.BigEndian.Uint32(data[pos : pos+4]))
		if keySize == 0 || keySize > 32 {
			return nil, fmt.Errorf("invalid key size: %d", keySize)
		}
		pos += 4
		node.Key = append(node.Key, data[pos:pos+keySize])
		pos += keySize
	}

	dataLength := int32(binary.BigEndian.Uint32(data[pos : pos+4]))
	pos += 4

	for i := int32(0); i < dataLength; i++ {
		offset := int64(binary.BigEndian.Uint64(data[pos : pos+8]))
		pos += 8

		length := int32(binary.BigEndian.Uint32(data[pos : pos+4]))
		pos += 4

		node.Data = append(node.Data, [2]int{int(offset), int(length)})
	}

	for i := 0; i < 16+1; i++ {
		subNodeAddress := binary.BigEndian.Uint64(data[pos : pos+8])
		pos += 8
		node.SubNodeAddress = append(node.SubNodeAddress, int(subNodeAddress))
	}

	return node, nil
}

func (s *Search) searchNode(field string, key []byte, node *node) ([2]int, error) {
	if node == nil {
		return [2]int{}, fmt.Errorf("node is nil")
	}
	var found bool
	var next int
	if found, next = util.SliceContains(node.Key, key); found {
		return node.Data[next], nil
	} else {
		if util.IsLeaf(node.SubNodeAddress) {
			return [2]int{}, fmt.Errorf("latest leaf node")
		}
	}
	if node.SubNodeAddress[next] == 0 {
		return [2]int{}, fmt.Errorf("non-root node address 0")
	}
	subNode, err := s.nodeByAddress(field, node.SubNodeAddress[next])
	if err != nil {
		return [2]int{}, fmt.Errorf("failed to retrive subNode %d", next)
	}
	return s.searchNode(field, key, subNode)
}

func (s *Search) tagSuggestionData(field string, data [2]int) ([]string, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("https://ltn.hitomi.la/tagindex/%s.%s.data", field, s.indexVersion["tagindex"]), nil)
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", data[0], data[0]+data[1]))
	resp, err := s.options.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var position = 4
	suggestionLength := int32(binary.BigEndian.Uint32(content[0:4]))
	var suggestions = make([]string, suggestionLength)
	for i := int32(0); i < suggestionLength; i++ {
		headerLength := int32(binary.BigEndian.Uint32(content[position : position+4]))
		position += 4
		header := string(content[position : position+int(headerLength)])
		position += int(headerLength)
		tagLength := int32(binary.BigEndian.Uint32(content[position : position+4]))
		position += 4
		tag := string(content[position : position+int(tagLength)])
		position += int(tagLength) + 4
		suggestions[i] = header + ":" + strings.ReplaceAll(tag, " ", "_")
	}
	return suggestions, nil
}

func HashTerm(query string) []byte {
	bytes := sha256.Sum256([]byte(query))
	return bytes[0:4]
}
