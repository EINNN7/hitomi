package hitomi

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/EINNN7/hitomi/internal/util"
)

const MaxNodeSize = 464

type Search struct {
	options *Options

	logger       zerolog.Logger
	indexVersion map[string]string
}

func NewSearch(options *Options) *Search {
	return &Search{
		options:      options,
		indexVersion: map[string]string{},
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

func (s *Search) NodeByAddress(field string, address int) (*SearchNode, error) {
	s.options.Logger.Debug().Msgf("retrieve node at %s:%d", field, address)
	var url string
	switch field {
	case "galleries":
		if _, ok := s.indexVersion["galleriesindex"]; !ok {
			version, err := s.IndexVersion("galleriesindex")
			if err != nil {
				return nil, err
			}
			s.indexVersion["galleriesindex"] = version
		}
		url = fmt.Sprintf("https://ltn.hitomi.la/galleriesindex/galleries.%s.index", s.indexVersion["galleriesindex"])
	case "languages":
		if _, ok := s.indexVersion["languagesindex"]; !ok {
			version, err := s.IndexVersion("languagesindex")
			if err != nil {
				return nil, err
			}
			s.indexVersion["languagesindex"] = version
		}
		url = fmt.Sprintf("https://ltn.hitomi.la/languagesindex/languages.%s.index", s.indexVersion["languagesindex"])
	case "nozomiurl":
		if _, ok := s.indexVersion["nozomiurlindex"]; !ok {
			version, err := s.IndexVersion("nozomiurlindex")
			if err != nil {
				return nil, err
			}
			s.indexVersion["nozomiurlindex"] = version
		}
		url = fmt.Sprintf("https://ltn.hitomi.la/nozomiurlindex/nozomiurl.%s.index", s.indexVersion["nozomiurlindex"])
	default:
		if _, ok := s.indexVersion["tagindex"]; !ok {
			version, err := s.IndexVersion("tagindex")
			if err != nil {
				return nil, err
			}
			s.indexVersion["tagindex"] = version
		}
		url = fmt.Sprintf("https://ltn.hitomi.la/tagindex/%s.%s.index", field, s.indexVersion["tagindex"])
	}
	s.options.Logger.Debug().Msgf("calling %s", url)
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
	return DecodeNode(content)
}

type SearchNode struct {
	Key            [][]byte
	Data           [][2]int
	SubNodeAddress []int
}

func DecodeNode(data []byte) (*SearchNode, error) {
	node := new(SearchNode)
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

func (s *Search) SearchNode(field string, key []byte, node *SearchNode) ([2]int, error) {
	if node == nil {
		return [2]int{}, fmt.Errorf("node is nil")
	}

	s.options.Logger.Debug().Ints("nodes", node.SubNodeAddress).Msg("nodes")

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
	subNode, err := s.NodeByAddress(field, node.SubNodeAddress[next])
	if err != nil {
		return [2]int{}, fmt.Errorf("failed to retrive subNode %d", next)
	}
	return s.SearchNode(field, key, subNode)
}

func (s *Search) TagSuggestionData(field string, data [2]int) ([]string, error) {
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
	suggestionsLength := int32(binary.BigEndian.Uint32(content[0:4]))
	var suggestions = make([]string, suggestionsLength)
	for i := int32(0); i < suggestionsLength; i++ {
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
