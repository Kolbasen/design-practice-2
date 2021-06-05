package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
)

const baseAddress = "http://balancer:8090"

var client = http.Client{
	Timeout: 3 * time.Second,
}

func TestBalancer(t *testing.T) {
	// TODO: Реалізуйте інтеграційний тест для балансувальникка.
	
	// The same server should response for the same endpoint
	// endpoint "some-data"
	resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data?key=kfcteam", baseAddress));
	if err != nil {
		assert.Equal(t, nil, err)
		return;
	}
	serverSelected := resp.Header.Get("lb-from");
	for i := 0; i < 9; i++ {
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data?key=kfcteam", baseAddress));
		if err != nil {
			assert.Equal(t, nil, err)
			return;
		}
		assert.Equal(t, serverSelected, resp.Header.Get("lb-from"))
	}

	// endpoint "another-data"
	resp, err = client.Get(fmt.Sprintf("%s/api/v1/another-data?key=kfcteam", baseAddress));
	if err != nil {
		assert.Equal(t, nil, err)
		return;
	}
	serverSelected = resp.Header.Get("lb-from");
	for i := 0; i < 9; i++ {
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/another-data?key=kfcteam", baseAddress));
		if err != nil {
			assert.Equal(t, nil, err)
			return;
		}
		assert.Equal(t, serverSelected, resp.Header.Get("lb-from"))
	}

	// endpoint "last-data"
	resp, err = client.Get(fmt.Sprintf("%s/api/v1/last-data", baseAddress));
	if err != nil {
		assert.Equal(t, nil, err)
		return;
	}
	serverSelected = resp.Header.Get("lb-from");
	for i := 0; i < 9; i++ {
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/last-data?key=kfcteam", baseAddress));
		if err != nil {
			assert.Equal(t, nil, err)
			return;
		}
		assert.Equal(t, serverSelected, resp.Header.Get("lb-from"))
	}
}

func BenchmarkBalancer(b *testing.B) {
	for n := 0; n < 100; n++ {
		_, err := client.Get(fmt.Sprintf("%s/api/v1/some-data?key=kfcteam", baseAddress))
		if err != nil {
			assert.Equal(b, nil, err)
			return;
		}

		_, err = client.Get(fmt.Sprintf("%s/api/v1/another-data?key=kfcteam", baseAddress))
		if err != nil {
			assert.Equal(b, nil, err)
			return;
		}

		_, err = client.Get(fmt.Sprintf("%s/api/v1/last-data?key=kfcteam", baseAddress))
		if err != nil {
			assert.Equal(b, nil, err)
			return;
		}
	}
}
