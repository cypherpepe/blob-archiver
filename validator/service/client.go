package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/attestantio/go-eth2-client/api"
	"github.com/base-org/blob-archiver/common/storage"
)

type Format string

const (
	// FormatJson instructs the client to request the response in JSON format
	FormatJson Format = "application/json"
	// FormatSSZ instructs the client to request the response in SSZ format
	FormatSSZ Format = "application/octet-stream"
)

// BlobSidecarClient is a minimal client for fetching sidecars from the blob service.
type BlobSidecarClient interface {
	// FetchSidecars fetches the sidecars for a given slot from the blob sidecar API.
	// It returns the HTTP status code and the sidecars.
	FetchSidecars(id string, format Format) (int, storage.BlobSidecars, error)
}

type httpBlobSidecarClient struct {
	url    string
	client *http.Client
}

// NewBlobSidecarClient creates a new BlobSidecarClient that fetches sidecars from the given URL.
func NewBlobSidecarClient(url string) BlobSidecarClient {
	return &httpBlobSidecarClient{
		url:    url,
		client: &http.Client{},
	}
}

// FetchSidecars fetches the sidecars for a given slot from the blob sidecar API.
func (c *httpBlobSidecarClient) FetchSidecars(id string, format Format) (int, storage.BlobSidecars, error) {
	url := fmt.Sprintf("%s/eth/v1/beacon/blob_sidecars/%s", c.url, id)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return http.StatusInternalServerError, storage.BlobSidecars{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", string(format))

	response, err := c.client.Do(req)
	if err != nil {
		return http.StatusInternalServerError, storage.BlobSidecars{}, fmt.Errorf("failed to fetch sidecars: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return response.StatusCode, storage.BlobSidecars{}, nil
	}

	var sidecars storage.BlobSidecars
	if format == FormatJson {
		if err := decodeJSON(response.Body, &sidecars); err != nil {
			return response.StatusCode, storage.BlobSidecars{}, err
		}
	} else {
		if err := decodeSSZ(response.Body, &sidecars); err != nil {
			return response.StatusCode, storage.BlobSidecars{}, err
		}
	}

	return response.StatusCode, sidecars, nil
}

// decodeJSON decodes a JSON response body into a BlobSidecars struct.
func decodeJSON(body io.Reader, sidecars *storage.BlobSidecars) error {
	if err := json.NewDecoder(body).Decode(sidecars); err != nil {
		return fmt.Errorf("failed to decode json response: %w", err)
	}
	return nil
}

// decodeSSZ decodes an SSZ response body into a BlobSidecars struct.
func decodeSSZ(body io.Reader, sidecars *storage.BlobSidecars) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var s api.BlobSidecars
	if err := s.UnmarshalSSZ(data); err != nil {
		return fmt.Errorf("failed to decode ssz response: %w", err)
	}

	sidecars.Data = s.Sidecars
	return nil
}
