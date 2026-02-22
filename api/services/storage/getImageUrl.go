package storage

import (
	"bytes"
	"encoding/json"
	utils "govision/api/pkg/utils"
)

type StorageService[T any] struct {
	URL           string
	ResponseModel T
}

func (s *StorageService[T]) GetImageUrl(body *bytes.Buffer, apiKey string) (*T, error) {
	err, respBytes := utils.SendRequest(s.URL, body, apiKey)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(respBytes, &s.ResponseModel); err != nil {
		return nil, err
	}

	return &s.ResponseModel, nil
}
