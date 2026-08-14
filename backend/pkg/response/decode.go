package response

import (
	"encoding/json"
	"net/http"
)

const maxBodyBytes = 1 << 20

func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Request body is not valid JSON")
		return err
	}

	return nil
}
