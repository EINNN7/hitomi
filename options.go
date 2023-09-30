package hitomi

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Options struct {
	Client               *http.Client
	Logger               zerolog.Logger
	UpdateScriptInterval time.Duration
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

func DefaultOptions() *Options {
	return &Options{
		Client:               &http.Client{},
		Logger:               log.Logger.With().Str("caller", "github.com/EINNN7/hitomi").Logger(),
		UpdateScriptInterval: -1,
	}
}
