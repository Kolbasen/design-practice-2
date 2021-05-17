package main

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBalancer(t *testing.T) {
	// TODO: Реалізуйте юніт-тест для балансувальникка.
	server, err := getForwardServer([]Server{
		{
			URL:     "1",
			IsAlive: true,
		},
		{
			URL:     "2",
			IsAlive: true,
		},
		{
			URL:     "3",
			IsAlive: true,
		},
	}, &http.Request{
		URL: &url.URL{
			Path: "test path",
		},
	})

	assert.Equal(t, nil, err)
	assert.Equal(t, "2", server.URL)

	server, err = getForwardServer([]Server{
		{
			URL:     "1",
			IsAlive: true,
		},
		{
			URL:     "2",
			IsAlive: false,
		},
		{
			URL:     "3",
			IsAlive: true,
		},
	}, &http.Request{
		URL: &url.URL{
			Path: "test path",
		},
	})

	assert.Equal(t, nil, err)
	assert.Equal(t, "1", server.URL)

	server, err = getForwardServer([]Server{
		{
			URL:     "1",
			IsAlive: false,
		},
		{
			URL:     "2",
			IsAlive: false,
		},
		{
			URL:     "3",
			IsAlive: true,
		},
	}, &http.Request{
		URL: &url.URL{
			Path: "test path",
		},
	})

	assert.Equal(t, nil, err)
	assert.Equal(t, "3", server.URL)

	server, err = getForwardServer([]Server{
		{
			URL:     "1",
			IsAlive: false,
		},
		{
			URL:     "2",
			IsAlive: false,
		},
		{
			URL:     "3",
			IsAlive: false,
		},
	}, &http.Request{
		URL: &url.URL{
			Path: "test path",
		},
	})

	assert.Equal(t, "No healthy servers", err.Error())
}
