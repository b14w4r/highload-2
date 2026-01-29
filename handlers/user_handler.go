package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"go-microservice/models"
	"go-microservice/services"
	"go-microservice/utils"

	"github.com/gorilla/mux"
)

type UserHandler struct {
	svc      services.UserService
	audit    *utils.AuditLogger
	notifier *services.Notifier
	errs     *utils.ErrorReporter
}

func NewUserHandler(svc services.UserService, audit *utils.AuditLogger, notifier *services.Notifier, errs *utils.ErrorReporter) *UserHandler {
	// свяжем notifier с sink’ом ошибок
	notifier.BindErrorSink(errs.Channel())

	return &UserHandler{
		svc:      svc,
		audit:    audit,
		notifier: notifier,
		errs:     errs,
	}
}

func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users := h.svc.List()
	writeJSON(w, http.StatusOK, users)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	u, err := h.svc.Get(id)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		h.errs.Report(err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, u)
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var u models.User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	saved, err := h.svc.Create(u)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Важно: ID уже присвоен
	h.audit.Log("CREATE", saved.ID)
	h.notifier.Notify("CREATE", saved.ID)

	writeJSON(w, http.StatusCreated, saved)
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var u models.User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	updated, err := h.svc.Update(id, u)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.audit.Log("UPDATE", updated.ID)
	h.notifier.Notify("UPDATE", updated.ID)

	writeJSON(w, http.StatusOK, updated)
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.svc.Delete(id); err != nil {
		if errors.Is(err, services.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		h.errs.Report(err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.audit.Log("DELETE", id)
	h.notifier.Notify("DELETE", id)

	w.WriteHeader(http.StatusNoContent)
}

func parseID(r *http.Request) (int, error) {
	idStr := mux.Vars(r)["id"]
	return strconv.Atoi(idStr)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

