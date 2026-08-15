package group

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Arifinwidy02/splitmate-backend/internal/middleware"
	"github.com/Arifinwidy02/splitmate-backend/pkg/apperror"
	"github.com/Arifinwidy02/splitmate-backend/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type createGroupRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Currency    string  `json:"currency"`
}

type updateGroupRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Currency    *string `json:"currency"`
}

type inviteRequest struct {
	Email string `json:"email"`
}

type groupResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	Currency    string    `json:"currency"`
	Role        string    `json:"role"`
	MemberCount int       `json:"memberCount"`
	HasLogo     bool      `json:"hasLogo"`
	CreatedAt   time.Time `json:"createdAt"`
}

type groupData struct {
	Group *groupResponse `json:"group"`
}

type groupsData struct {
	Groups []*groupResponse `json:"groups"`
}

type memberResponse struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joinedAt"`
}

type membersData struct {
	Members []*memberResponse `json:"members"`
}

type invitationResponse struct {
	ID        uuid.UUID `json:"id"`
	GroupID   uuid.UUID `json:"groupId"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expiresAt"`
	Token     string    `json:"token,omitempty"`
}

type invitationData struct {
	Invitation *invitationResponse `json:"invitation"`
}

type emptyData struct{}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	var req createGroupRequest
	logo, err := decodeGroupRequest(w, r, &req)
	if err != nil {
		return
	}

	g, err := h.service.CreateGroup(r.Context(), userID, req.Name, deref(req.Description), req.Currency, logo)
	if err != nil {
		h.writeGroupError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, envelope{Data: groupData{Group: toGroupResponse(g)}})
}

func decodeGroupRequest(w http.ResponseWriter, r *http.Request, dst *createGroupRequest) (*Logo, error) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := response.DecodeJSON(w, r, dst); err != nil {
			return nil, err
		}
		return nil, nil
	}

	logo, err := decodeMultipartGroupRequest(r, dst)
	if err != nil {
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return nil, err
	}

	return logo, nil
}

func decodeMultipartGroupRequest(r *http.Request, dst *createGroupRequest) (*Logo, error) {
	if err := r.ParseMultipartForm(logoFieldLimit); err != nil {
		return nil, errors.New("Invalid form data")
	}

	form := r.MultipartForm
	get := func(key string) string {
		if values := form.Value[key]; len(values) > 0 {
			return values[0]
		}
		return ""
	}
	dst.Name = get("name")
	dst.Currency = get("currency")
	if desc := get("description"); desc != "" {
		dst.Description = &desc
	}

	file, header, err := r.FormFile("logo")
	if errors.Is(err, http.ErrMissingFile) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.New("Invalid logo upload")
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxLogoBytes+1))
	if err != nil {
		return nil, errors.New("Invalid logo upload")
	}

	return &Logo{
		Image:       data,
		ContentType: header.Header.Get("Content-Type"),
	}, nil
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	groups, err := h.service.ListGroups(r.Context(), userID)
	if err != nil {
		h.writeGroupError(w, err)
		return
	}

	resp := make([]*groupResponse, 0, len(groups))
	for _, g := range groups {
		resp = append(resp, toGroupResponse(g))
	}

	response.WriteJSON(w, http.StatusOK, envelope{Data: groupsData{Groups: resp}})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	groupID, ok := pathUUID(r, "groupId")
	if !ok {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid group id")
		return
	}

	g, err := h.service.GetGroup(r.Context(), userID, groupID)
	if err != nil {
		h.writeGroupError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, envelope{Data: groupData{Group: toGroupResponse(g)}})
}

func (h *Handler) GetLogo(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	groupID, ok := pathUUID(r, "groupId")
	if !ok {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid group id")
		return
	}

	logo, err := h.service.GetLogo(r.Context(), userID, groupID)
	if err != nil {
		h.writeGroupError(w, err)
		return
	}

	w.Header().Set("Content-Type", logo.ContentType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(logo.Image); err != nil {
		slog.Error("write group logo", "error", err)
	}
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	groupID, ok := pathUUID(r, "groupId")
	if !ok {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid group id")
		return
	}

	var req updateGroupRequest
	if err := response.DecodeJSON(w, r, &req); err != nil {
		return
	}

	g, err := h.service.UpdateGroup(r.Context(), userID, groupID, req.Name, req.Description, req.Currency)
	if err != nil {
		h.writeGroupError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, envelope{Data: groupData{Group: toGroupResponse(g)}})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	groupID, ok := pathUUID(r, "groupId")
	if !ok {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid group id")
		return
	}

	if err := h.service.DeleteGroup(r.Context(), userID, groupID); err != nil {
		h.writeGroupError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, envelope{Data: emptyData{}})
}

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	groupID, ok := pathUUID(r, "groupId")
	if !ok {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid group id")
		return
	}

	members, err := h.service.ListMembers(r.Context(), userID, groupID)
	if err != nil {
		h.writeGroupError(w, err)
		return
	}

	resp := make([]*memberResponse, 0, len(members))
	for _, m := range members {
		resp = append(resp, &memberResponse{
			ID:       m.UserID,
			Name:     m.Name,
			Email:    m.Email,
			Role:     m.Role,
			JoinedAt: m.JoinedAt,
		})
	}

	response.WriteJSON(w, http.StatusOK, envelope{Data: membersData{Members: resp}})
}

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	groupID, ok := pathUUID(r, "groupId")
	if !ok {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid group id")
		return
	}

	memberID, ok := pathUUID(r, "userId")
	if !ok {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid member id")
		return
	}

	if err := h.service.RemoveMember(r.Context(), userID, groupID, memberID); err != nil {
		h.writeGroupError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, envelope{Data: emptyData{}})
}

func (h *Handler) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	groupID, ok := pathUUID(r, "groupId")
	if !ok {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid group id")
		return
	}

	var req inviteRequest
	if err := response.DecodeJSON(w, r, &req); err != nil {
		return
	}

	inv, token, err := h.service.CreateInvitation(r.Context(), userID, groupID, req.Email)
	if err != nil {
		h.writeGroupError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, envelope{Data: invitationData{Invitation: &invitationResponse{
		ID:        inv.ID,
		GroupID:   inv.GroupID,
		Email:     inv.Email,
		Status:    inv.Status,
		ExpiresAt: inv.ExpiresAt,
		Token:     token,
	}}})
}

func (h *Handler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	token := r.PathValue("token")

	g, err := h.service.AcceptInvitation(r.Context(), userID, token)
	if err != nil {
		h.writeGroupError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, envelope{Data: groupData{Group: toGroupResponse(g)}})
}

func (h *Handler) writeGroupError(w http.ResponseWriter, err error) {
	var valErr *apperror.Validation
	switch {
	case errors.As(err, &valErr):
		response.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", valErr.Message)
	case errors.Is(err, ErrGroupNotFound):
		response.WriteError(w, http.StatusNotFound, "GROUP_NOT_FOUND", "Group not found")
	case errors.Is(err, ErrNoLogo):
		response.WriteError(w, http.StatusNotFound, "LOGO_NOT_FOUND", "Group has no logo")
	case errors.Is(err, ErrMemberNotFound):
		response.WriteError(w, http.StatusNotFound, "MEMBER_NOT_FOUND", "Member not found")
	case errors.Is(err, ErrInvitationNotFound):
		response.WriteError(w, http.StatusNotFound, "INVITATION_NOT_FOUND", "Invitation not found")
	case errors.Is(err, ErrInvitationExpired):
		response.WriteError(w, http.StatusGone, "INVITATION_EXPIRED", "Invitation has expired")
	case errors.Is(err, ErrForbidden):
		response.WriteError(w, http.StatusForbidden, "FORBIDDEN", "You do not have permission to do this")
	case errors.Is(err, ErrInvitationForbidden):
		response.WriteError(w, http.StatusForbidden, "INVITATION_FORBIDDEN", "This invitation is for a different email")
	case errors.Is(err, ErrMemberExists):
		response.WriteError(w, http.StatusConflict, "MEMBER_EXISTS", "This user is already a member")
	case errors.Is(err, ErrInvitationExists):
		response.WriteError(w, http.StatusConflict, "INVITATION_EXISTS", "A pending invitation already exists for this email")
	case errors.Is(err, ErrInvitationUsed):
		response.WriteError(w, http.StatusConflict, "INVITATION_USED", "Invitation has already been used")
	default:
		slog.Error("group request failed", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong")
	}
}

func toGroupResponse(g *Group) *groupResponse {
	return &groupResponse{
		ID:          g.ID,
		Name:        g.Name,
		Description: g.Description,
		Currency:    g.Currency,
		Role:        g.Role,
		MemberCount: g.MemberCount,
		HasLogo:     g.HasLogo,
		CreatedAt:   g.CreatedAt,
	}
}

func pathUUID(r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

type envelope struct {
	Data any `json:"data"`
}
