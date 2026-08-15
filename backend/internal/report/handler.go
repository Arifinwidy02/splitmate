package report

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	"github.com/google/uuid"

	"github.com/Arifinwidy02/splitmate-backend/internal/middleware"
	"github.com/Arifinwidy02/splitmate-backend/pkg/response"
)

const xlsxContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

var unsafeFilenameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Export streams the group report as an .xlsx attachment.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	groupID, ok := pathUUID(r, "groupId")
	if !ok {
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Invalid group id")
		return
	}

	report, err := h.service.BuildReport(r.Context(), userID, groupID)
	if err != nil {
		h.writeError(w, err)
		return
	}

	data, err := RenderXLSX(report)
	if err != nil {
		slog.Error("failed to render report workbook", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
		return
	}

	filename := safeFilename(report.GroupName) + "-report.xlsx"
	w.Header().Set("Content-Type", xlsxContentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"; filename*=UTF-8''"+url.PathEscape(filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		slog.Error("failed to write report workbook", "error", err)
	}
}

func (h *Handler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrGroupNotFound):
		response.WriteError(w, http.StatusNotFound, "GROUP_NOT_FOUND", "Group not found")
	default:
		slog.Error("report request failed", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
	}
}

func pathUUID(r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

// safeFilename keeps only filename-safe characters so the Content-Disposition
// header cannot be abused with the group name.
func safeFilename(name string) string {
	return unsafeFilenameChars.ReplaceAllString(name, "-")
}
