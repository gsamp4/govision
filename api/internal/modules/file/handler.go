package file

import (
	"log"
	"net/http"

	"govision/api/services/rabbitmq"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	service *Service
}

func NewHandler(p rabbitmq.JobPublisher) *Handler {
	return &Handler{service: NewService(p)}
}

func (h *Handler) UploadFileImage(c echo.Context) error {
	log.Println("[STARTING] - calling route /image/upload...")

	var request UploadRequest
	if err := c.Bind(&request); err != nil {
		log.Printf("[ERROR] - Invalid payload: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "Invalid payload",
		})
	}

	log.Println("[RUNNING] - Getting file.")
	file, err := c.FormFile("file")
	if err != nil {
		log.Printf("[ERROR] - error getting file data: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "Error getting file data",
		})
	}

	ctx := c.Request().Context()
	jobID, err := h.service.ProcessUpload(ctx, file)
	if err != nil {
		log.Printf("[ERROR] - %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusAccepted, map[string]string{
		"job_id": jobID,
		"status": "queued",
	})
}
