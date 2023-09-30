package hitomi

import (
	"encoding/json"
	"net/http"
	"testing"
)

var client *Client

func pp(v any) string {
	m, _ := json.MarshalIndent(v, "", "    ")
	return string(m)
}

func TestMain(t *testing.M) {
	client = NewClient(DefaultOptions())
	t.Run()
}

func TestClient_UpdateScript(t *testing.T) {
	if err := client.UpdateScript(); err != nil {
		t.Fatal(err)
	}
}

func TestClient_Gallery(t *testing.T) {
	gallery, err := client.Gallery("1142761")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(pp(gallery))
}

func TestClient_File(t *testing.T) {
	file, err := client.File(client.FileURL("bd950fbb6310a70d790082d194a282c3585a3a87b19ed4df7f8320ad965829c4"), "1142761")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(http.DetectContentType(file))
}
