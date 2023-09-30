package hitomi

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Options struct {
	Client *http.Client
	Logger zerolog.Logger

	// Client-specific options

	// UpdateScriptInterval is an option to update the script every interval.
	// if it is set to -1, it will never update the script.
	UpdateScriptInterval time.Duration

	// Search-specific options

	// CacheWholeIndex is an option to download the whole index and cache it.
	// it can be extremely slow the first time (especially for gallery index) and consume much memory space,
	// but it will be a lot faster when you search.
	CacheWholeIndex bool
}

func (o *Options) WithClient(c *http.Client) *Options {
	o.Client = c
	return o
}

func (o *Options) WithLogger(l zerolog.Logger) *Options {
	o.Logger = l
	return o
}

func (o *Options) WithUpdateScriptInterval(t time.Duration) *Options {
	o.UpdateScriptInterval = t
	return o
}

func (o *Options) WithCacheWholeIndex(b bool) *Options {
	o.CacheWholeIndex = b
	return o
}

func DefaultOptions() *Options {
	return &Options{
		Client:               &http.Client{},
		Logger:               log.Logger.With().Str("caller", "github.com/EINNN7/hitomi").Logger(),
		UpdateScriptInterval: -1,

		CacheWholeIndex: false,
	}
}
