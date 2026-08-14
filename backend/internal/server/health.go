package server

import (
	"net/http"

	"github.com/Arifinwidy02/splitmate-backend/pkg/response"
)

type healthResponse struct {
	Status string `json:"status"`
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	response.WriteJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}
