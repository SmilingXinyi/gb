package main

import (
	"encoding/json"
	"time"

	"github.com/SmilingXinyi/gb/oss"
)

type jsonObjectMeta struct {
	Key          string            `json:"key"`
	Size         int64             `json:"size"`
	ContentType  string            `json:"content_type"`
	ETag         string            `json:"etag"`
	LastModified string            `json:"last_modified"`
	StorageClass string            `json:"storage_class"`
	Metadata     map[string]string `json:"metadata"`
}

type jsonObjectItem struct {
	Key          string `json:"key"`
	Size         int64  `json:"size"`
	ETag         string `json:"etag"`
	LastModified string `json:"last_modified"`
	StorageClass string `json:"storage_class"`
}

type jsonListResult struct {
	Objects        []jsonObjectItem `json:"objects"`
	CommonPrefixes []string         `json:"common_prefixes"`
	NextToken      string           `json:"next_token"`
	IsTruncated    bool             `json:"is_truncated"`
}

func marshalObjectMeta(m *oss.ObjectMeta) (string, error) {
	meta := m.Metadata
	if meta == nil {
		meta = map[string]string{}
	}
	j := jsonObjectMeta{
		Key:          m.Key,
		Size:         m.Size,
		ContentType:  m.ContentType,
		ETag:         m.ETag,
		LastModified: m.LastModified.UTC().Format(time.RFC3339),
		StorageClass: m.StorageClass,
		Metadata:     meta,
	}
	b, err := json.Marshal(j)
	return string(b), err
}

func marshalListResult(r *oss.ListResult) (string, error) {
	items := make([]jsonObjectItem, len(r.Objects))
	for i, obj := range r.Objects {
		items[i] = jsonObjectItem{
			Key:          obj.Key,
			Size:         obj.Size,
			ETag:         obj.ETag,
			LastModified: obj.LastModified.UTC().Format(time.RFC3339),
			StorageClass: obj.StorageClass,
		}
	}
	prefixes := r.CommonPrefixes
	if prefixes == nil {
		prefixes = []string{}
	}
	j := jsonListResult{
		Objects:        items,
		CommonPrefixes: prefixes,
		NextToken:      r.NextToken,
		IsTruncated:    r.IsTruncated,
	}
	b, err := json.Marshal(j)
	return string(b), err
}
