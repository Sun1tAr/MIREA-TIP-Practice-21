package http

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/sun1tar/MIREA-TIP-Practice-21/tech-ip-sem2/shared/middleware"
	"github.com/sun1tar/MIREA-TIP-Practice-21/tech-ip-sem2/tasks/internal/client/authclient"
	"github.com/sun1tar/MIREA-TIP-Practice-21/tech-ip-sem2/tasks/internal/models"
	"github.com/sun1tar/MIREA-TIP-Practice-21/tech-ip-sem2/tasks/internal/service"
)

type TaskHandler struct {
	taskService *service.TaskService
	authClient  *authclient.Client
	logger      *logrus.Logger
}

func NewTaskHandler(ts *service.TaskService, ac *authclient.Client, logger *logrus.Logger) *TaskHandler {
	return &TaskHandler{
		taskService: ts,
		authClient:  ac,
		logger:      logger,
	}
}

func (h *TaskHandler) verifyToken(w http.ResponseWriter, r *http.Request) bool {
	requestID := middleware.GetRequestID(r.Context())
	logEntry := h.logger.WithFields(logrus.Fields{
		"component":  "http_handler",
		"request_id": requestID,
	})

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		logEntry.Warn("missing authorization header")
		http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
		return false
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		logEntry.WithField("auth_header", authHeader).Warn("invalid authorization header format")
		http.Error(w, `{"error":"invalid authorization header format"}`, http.StatusUnauthorized)
		return false
	}
	token := parts[1]

	valid, _, err := h.authClient.VerifyToken(r.Context(), token)
	if err != nil {
		logEntry.WithError(err).Error("authentication service unavailable")
		http.Error(w, `{"error":"authentication service unavailable"}`, http.StatusServiceUnavailable)
		return false
	}
	if !valid {
		logEntry.WithField("token_present", token != "").Warn("invalid token")
		http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
		return false
	}

	logEntry.Debug("token verified successfully")
	return true
}

type createTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	DueDate     string `json:"due_date"`
}

type updateTaskRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	DueDate     *string `json:"due_date,omitempty"`
	Done        *bool   `json:"done,omitempty"`
}

type taskResponse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	DueDate     string `json:"due_date,omitempty"`
	Done        bool   `json:"done"`
}

func toTaskResponse(t *models.Task) taskResponse {
	return taskResponse{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		DueDate:     t.DueDate,
		Done:        t.Done,
	}
}

func toTaskResponses(tasks []*models.Task) []taskResponse {
	result := make([]taskResponse, len(tasks))
	for i, t := range tasks {
		result[i] = toTaskResponse(t)
	}
	return result
}

func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r.Context())
	logEntry := h.logger.WithFields(logrus.Fields{
		"component":  "http_handler",
		"handler":    "CreateTask",
		"request_id": requestID,
	})

	if !h.verifyToken(w, r) {
		return
	}

	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logEntry.WithError(err).Warn("invalid request body")
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		logEntry.Warn("title is required")
		http.Error(w, `{"error":"title is required"}`, http.StatusBadRequest)
		return
	}

	task, err := h.taskService.Create(r.Context(), req.Title, req.Description, req.DueDate)
	if err != nil {
		logEntry.WithError(err).Error("failed to create task")
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	logEntry.WithField("task_id", task.ID).Info("task created successfully")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(toTaskResponse(task))
}

func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r.Context())
	logEntry := h.logger.WithFields(logrus.Fields{
		"component":  "http_handler",
		"handler":    "ListTasks",
		"request_id": requestID,
	})

	if !h.verifyToken(w, r) {
		return
	}

	tasks, err := h.taskService.List(r.Context())
	if err != nil {
		logEntry.WithError(err).Error("failed to list tasks")
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	logEntry.WithField("count", len(tasks)).Debug("tasks listed")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toTaskResponses(tasks))
}

func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r.Context())
	logEntry := h.logger.WithFields(logrus.Fields{
		"component":  "http_handler",
		"handler":    "GetTask",
		"request_id": requestID,
	})

	if !h.verifyToken(w, r) {
		return
	}

	id := r.PathValue("id")
	task, err := h.taskService.GetByID(r.Context(), id)
	if err == sql.ErrNoRows {
		logEntry.WithField("task_id", id).Warn("task not found")
		http.Error(w, `{"error":"task not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		logEntry.WithError(err).Error("failed to get task")
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	logEntry.WithField("task_id", id).Debug("task retrieved")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toTaskResponse(task))
}

func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r.Context())
	logEntry := h.logger.WithFields(logrus.Fields{
		"component":  "http_handler",
		"handler":    "UpdateTask",
		"request_id": requestID,
	})

	if !h.verifyToken(w, r) {
		return
	}

	id := r.PathValue("id")
	var req updateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logEntry.WithError(err).Warn("invalid request body")
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	task, err := h.taskService.Update(r.Context(), id, req.Title, req.Description, req.DueDate, req.Done)
	if err == sql.ErrNoRows {
		logEntry.WithField("task_id", id).Warn("task not found for update")
		http.Error(w, `{"error":"task not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		logEntry.WithError(err).Error("failed to update task")
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	logEntry.WithField("task_id", id).Info("task updated successfully")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toTaskResponse(task))
}

func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r.Context())
	logEntry := h.logger.WithFields(logrus.Fields{
		"component":  "http_handler",
		"handler":    "DeleteTask",
		"request_id": requestID,
	})

	if !h.verifyToken(w, r) {
		return
	}

	id := r.PathValue("id")
	err := h.taskService.Delete(r.Context(), id)
	if err == sql.ErrNoRows {
		logEntry.WithField("task_id", id).Warn("task not found for deletion")
		http.Error(w, `{"error":"task not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		logEntry.WithError(err).Error("failed to delete task")
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	logEntry.WithField("task_id", id).Info("task deleted successfully")
	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskHandler) SearchTasks(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r.Context())
	logEntry := h.logger.WithFields(logrus.Fields{
		"component":  "http_handler",
		"handler":    "SearchTasks",
		"request_id": requestID,
	})

	if !h.verifyToken(w, r) {
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, `{"error":"search query parameter 'q' is required"}`, http.StatusBadRequest)
		return
	}

	// Параметр unsafe для демонстрации SQL-инъекции
	unsafe := r.URL.Query().Get("unsafe") == "true"

	logEntry.WithFields(logrus.Fields{
		"query":  query,
		"unsafe": unsafe,
	}).Info("searching tasks")

	tasks, err := h.taskService.SearchByTitle(r.Context(), query, unsafe)
	if err != nil {
		logEntry.WithError(err).Error("failed to search tasks")
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toTaskResponses(tasks))
}
