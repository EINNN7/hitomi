package hitomi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/EINNN7/hitomi/internal/script"
)

// Client is a hitomi client
type Client struct {
	options *Options

	script            *script.Script
	lastScriptUpdated time.Time
}

// NewClient creates a new hitomi client.
func NewClient(options *Options) *Client {
	return &Client{
		options: options,
	}
}

// UpdateScript updates script from https://ltn.hitomi.la/gg.js
// This is required to calculated file url.
func (c *Client) UpdateScript() error {
	req, err := http.NewRequest("GET", "https://ltn.hitomi.la/gg.js", nil)
	if err != nil {
		return err
	}
	resp, err := c.options.Client.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to get gg.js: %d", resp.StatusCode)
	}
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	c.script = script.ParseScript(string(content))
	c.lastScriptUpdated = time.Now()

	c.options.Logger.Debug().Str("base_path", c.script.BasePath).Msgf("Script updated")
	return nil
}

// Gallery returns normalized gallery information.
func (c *Client) Gallery(id string) (*Gallery, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://ltn.hitomi.la/galleries/%s.js", id), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.options.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to get gallery: %d", resp.StatusCode)
	}
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	galleryScript := new(galleryScript)
	if err := json.Unmarshal([]byte(strings.Replace(string(content), "var galleryinfo = ", "", 1)), galleryScript); err != nil {
		return nil, err
	}
	return galleryScript.Normalize(), nil
}

// File returns file bytes
func (c *Client) File(url, galleryId string) ([]byte, error) {
	req := c.FileRequest(url, galleryId)
	resp, err := c.options.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to get file: %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// FileURL returns calculated url for file
// returned file url is not permanent, usually it lasts 30~ minutes after gg.js updated
func (c *Client) FileURL(hash string) string {
	if c.options.UpdateScriptInterval != -1 && time.Since(c.lastScriptUpdated) > c.options.UpdateScriptInterval {
		if err := c.UpdateScript(); err != nil {
			c.options.Logger.Warn().Err(err).Msg("failed to update script")
		}
	}
	return fmt.Sprintf("https://%s.hitomi.la/webp/%s.webp", c.script.SubdomainFromURL(fmt.Sprintf("https://a.hitomi.la/webp/%s", c.script.FullPathFromHash(hash)), "a"), c.script.FullPathFromHash(hash))
}

// FileRequest returns *http.Request for file
// useful when displaying download progress.
func (c *Client) FileRequest(url, galleryId string) *http.Request {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Set("Referer", fmt.Sprintf("https://hitomi.la/reader/%s.html", galleryId))
	return req
}

// Gallery represents gallery information.
type Gallery struct {
	Language          string
	Id                string
	Blocked           bool
	Related           []string
	LanguageUrl       string
	LanguageLocalName string
	Title             string
	Date              string
	Type              string
	GalleryUrl        string
	JapaneseTitle     *string
	Languages         []struct {
		LocalName string
		GalleryId string
		Name      string
		Url       string
	}
	Tags []struct {
		Tag string
		Url string
	}
	Artists []struct {
		Artist string
		Url    string
	}
	Files []struct {
		HasJXL  bool
		HasAVIF bool
		HasWEBP bool
		Width   int
		Height  int
		Name    string
		Hash    string
		Single  bool
	}
	Characters []struct {
		Character string
		Url       string
	}
	Parodies []struct {
		Parody string
		Url    string
	}
	Groups []struct {
		Group string
		Url   string
	}
}

type galleryScript struct {
	Related      []int         `json:"related"`
	SceneIndexes []interface{} `json:"scene_indexes"`
	Languages    []struct {
		LanguageLocalname string `json:"language_localname"`
		Galleryid         string `json:"galleryid"`
		Name              string `json:"name"`
		Url               string `json:"url"`
	} `json:"languages"`
	Tags []struct {
		Male   interface{} `json:"male"`
		Female interface{} `json:"female"`
		Tag    string      `json:"tag"`
		Url    string      `json:"url"`
	} `json:"tags"`
	Videofilename interface{} `json:"videofilename"`
	JapaneseTitle *string     `json:"japanese_title"`
	Artists       []struct {
		Artist string `json:"artist"`
		Url    string `json:"url"`
	} `json:"artists"`
	LanguageUrl       string `json:"language_url"`
	LanguageLocalname string `json:"language_localname"`
	Title             string `json:"title"`
	Files             []struct {
		Hasjxl  int    `json:"hasjxl"`
		Hasavif int    `json:"hasavif"`
		Width   int    `json:"width"`
		Haswebp int    `json:"haswebp"`
		Height  int    `json:"height"`
		Name    string `json:"name"`
		Hash    string `json:"hash"`
		Single  int    `json:"single,omitempty"`
	} `json:"files"`
	Date       string      `json:"date"`
	Video      interface{} `json:"video"`
	Type       string      `json:"type"`
	Characters []struct {
		Character string `json:"character"`
		Url       string `json:"url"`
	} `json:"characters"`
	Parodys []struct {
		Parody string `json:"parody"`
		Url    string `json:"url"`
	} `json:"parodys"`
	Galleryurl string `json:"galleryurl"`
	Groups     []struct {
		Group string `json:"group"`
		Url   string `json:"url"`
	} `json:"groups"`
	Language string           `json:"language"`
	Id       *json.RawMessage `json:"id"`
	Blocked  int              `json:"blocked"`
}

func (g *galleryScript) GetId() string {
	if g.Id == nil {
		return ""
	}
	if strings.HasPrefix(string(*g.Id), `"`) {
		return strings.Trim(string(*g.Id), `"`)
	}
	return string(*g.Id)
}

func (g *galleryScript) Normalize() *Gallery {
	gallery := &Gallery{}
	for _, related := range g.Related {
		gallery.Related = append(gallery.Related, strconv.Itoa(related))
	}
	for _, language := range g.Languages {
		gallery.Languages = append(gallery.Languages, struct {
			LocalName string
			GalleryId string
			Name      string
			Url       string
		}{
			LocalName: language.LanguageLocalname,
			GalleryId: language.Galleryid,
			Name:      language.Name,
			Url:       language.Url,
		})
	}
	// Convert tags
	for _, tag := range g.Tags {
		var tagType string
		if v, ok := tag.Male.(string); ok && v == "1" {
			tagType = "male:"
		} else if v, ok := tag.Female.(string); ok && v == "1" {
			tagType = "female:"
		} else {
			tagType = "tag:"
		}
		gallery.Tags = append(gallery.Tags, struct {
			Tag string
			Url string
		}{
			Tag: tagType + strings.ReplaceAll(tag.Tag, " ", "_"),
			Url: tag.Url,
		})
	}
	gallery.JapaneseTitle = g.JapaneseTitle
	gallery.Artists = []struct {
		Artist string
		Url    string
	}(g.Artists)
	gallery.LanguageUrl = g.LanguageUrl
	gallery.LanguageLocalName = g.LanguageLocalname
	gallery.Title = g.Title

	// Convert files
	for _, file := range g.Files {
		gallery.Files = append(gallery.Files, struct {
			HasJXL  bool
			HasAVIF bool
			HasWEBP bool
			Width   int
			Height  int
			Name    string
			Hash    string
			Single  bool
		}{
			HasJXL:  file.Hasjxl == 1,
			HasAVIF: file.Hasavif == 1,
			HasWEBP: file.Haswebp == 1,
			Width:   file.Width,
			Height:  file.Height,
			Name:    file.Name,
			Hash:    file.Hash,
			Single:  file.Single == 1,
		})
	}

	gallery.Date = g.Date
	gallery.Type = g.Type
	gallery.Characters = []struct {
		Character string
		Url       string
	}(g.Characters)
	gallery.Parodies = []struct {
		Parody string
		Url    string
	}(g.Parodys)
	gallery.GalleryUrl = g.Galleryurl
	gallery.Groups = []struct {
		Group string
		Url   string
	}(g.Groups)
	gallery.Language = g.Language
	gallery.Id = g.GetId()
	gallery.Blocked = g.Blocked == 1

	return gallery
}
