package handlers

import (
	"forum/internal/db"
	"html/template"
	"log"
	"net/http"
)

type NotificationsHandler struct {
	repo        *db.Repository
	log         *log.Logger
	projectRoot string
}

func NewNotificationsHandler(repo *db.Repository, log *log.Logger, projectRoot string) *NotificationsHandler {
	return &NotificationsHandler{repo: repo, log: log, projectRoot: projectRoot}
}

// ListNotifications отображает уведомления пользователя
func (h *NotificationsHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		renderError(w, http.StatusMethodNotAllowed, "405 Method Not Allowed", "Метод не поддерживается", h.projectRoot)
		return
	}

	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login?error=Требуется авторизация", http.StatusSeeOther)
		return
	}

	notifs, err := h.repo.GetNotificationsByUser(userID)
	if err != nil {
		h.log.Printf("Ошибка загрузки уведомлений: %v", err)
		renderError(w, http.StatusInternalServerError, "500 Internal Server Error", "Внутренняя ошибка сервера", h.projectRoot)
		return
	}

	tmpl, err := template.ParseFiles("static/notifications.html")
	if err != nil {
		h.log.Printf("Ошибка загрузки шаблона: %v", err)
		renderError(w, http.StatusInternalServerError, "500 Internal Server Error", h.projectRoot, "Внутренняя ошибка сервера")
		return
	}

	data := map[string]interface{}{
		"Notifications": notifs,
	}
	tmpl.Execute(w, data)
}
