package registration

import (
	"encoding/json"
	"net/http"

	"github.com/norest-mail/server/internal/response"
)

type EnhancedHandler struct {
	service *EnhancedService
}

func NewEnhancedHandler(service *EnhancedService) *EnhancedHandler {
	return &EnhancedHandler{service: service}
}

// Register handles user registration with automatic domain type detection.
func (h *EnhancedHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	authRes, flowRes, err := h.service.RegisterWithDomainDetection(r.Context(), req.Email, req.Password)
	if err != nil {
		// If registration failed due to auto-provisioning, return an appropriate error
		if err == ErrAddressNotAvailable {
			response.Error(w, http.StatusConflict, "address not available - please choose a different username")
			return
		}
		if err == ErrDomainNotAvailable {
			response.Error(w, http.StatusBadRequest, "domain not available for registration")
			return
		}
		// Handle other errors
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"id":           authRes.ID,
		"email":        authRes.Email,
		"status":       authRes.Status,
		"access_token": authRes.AccessToken,
		"refresh_token": authRes.RefreshToken,
		"registration_flow": flowRes,
	})
}