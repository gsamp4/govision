package file

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"mime/multipart"
	"os"
	"time"

	"govision/api/services/rabbitmq"
	storage "govision/api/services/storage"

	"github.com/oklog/ulid/v2"
)

const HOST_IMAGE_URL = "https://api.imgbb.com/1/upload"

type Service struct {
	publisher rabbitmq.JobPublisher
}

func NewService(p rabbitmq.JobPublisher) *Service {
	return &Service{publisher: p}
}

func (s *Service) ProcessUpload(ctx context.Context, fileHeader *multipart.FileHeader) (string, error) {
	log.Println("[RUNNING] - Validating file size.")
	if err := ValidateFileSize(fileHeader); err != nil {
		return "", fmt.Errorf("invalid file size: %w", err)
	}

	fileObject, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}
	defer fileObject.Close()

	log.Println("[RUNNING] - Validating file content...")
	if err := ValidateFileContent(fileObject); err != nil {
		return "", fmt.Errorf("invalid file content: %w", err)
	}

	log.Println("[RUNNING] - Processing file data...")
	buf := &bytes.Buffer{}
	if _, err = io.Copy(buf, fileObject); err != nil {
		return "", fmt.Errorf("error processing file image: %w", err)
	}

	log.Println("[RUNNING] - Sending image to storage service")
	imageURL, err := s.uploadToStorage(buf)
	if err != nil {
		return "", err
	}

	jobID := s.generateJobID()

	log.Println("[RUNNING] - Publishing job to queue...")
	if err := s.publisher.Publish(ctx, jobID, imageURL); err != nil {
		return "", fmt.Errorf("failed to enqueue job: %w", err)
	}

	log.Printf("[SUCCESS] - File image processed successfully!")
	return jobID, nil
}

func (s *Service) uploadToStorage(buf *bytes.Buffer) (string, error) {
	storageService := storage.StorageService[ImgBBResponse]{
		URL: HOST_IMAGE_URL,
	}

	apiKey := os.Getenv("STORAGE_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("storage API key not configured")
	}

	responseObj, err := storageService.GetImageUrl(buf, apiKey)
	if err != nil {
		return "", fmt.Errorf("storage service error: %w", err)
	}

	return responseObj.Data.URL, nil
}

func (s *Service) generateJobID() string {
	entropy := ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
}
