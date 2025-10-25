package handlers

import (
	"forum/internal/db"
	"html/template"
	"log"
	"net/http"
)

type CategoryHandler struct {
	repo        *db.Repository
	log         *log.Logger
	projectRoot string
}

func NewCategoryHandler(repo *db.Repository, log *log.Logger, projectRoot string) *CategoryHandler {
	return &CategoryHandler{repo: repo, log: log, projectRoot: projectRoot}
}

// ListCategories отображает список всех категорий
func (h *CategoryHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		renderError(w, http.StatusMethodNotAllowed, "405 Method Not Allowed", "Метод не поддерживается", h.projectRoot)
		return
	}
	categories, err := h.repo.GetAllCategories()
	if err != nil {
		h.log.Printf("Ошибка загрузки категорий: %v", err)
		renderError(w, http.StatusInternalServerError, "500 Internal Server Error", "Внутренняя ошибка сервера", h.projectRoot)
		return
	}

	tmpl, err := template.ParseFiles("static/categories.html")
	if err != nil {
		h.log.Printf("Ошибка загрузки шаблона: %v", err)
		renderError(w, http.StatusInternalServerError, "500 Internal Server Error", "Внутренняя ошибка сервера", h.projectRoot)
		return
	}

	// Ручное определение пользователя по cookie
	cookie, err := r.Cookie("session_id")
	isAuthenticated := false
	username := ""
	if err == nil {
		session, err := h.repo.GetSession(cookie.Value)
		if err == nil {
			user, err := h.repo.GetUserByID(session.UserID)
			if err == nil {
				isAuthenticated = true
				username = user.Username
			}
		}
	}

	data := map[string]interface{}{
		"Categories":      categories,
		"Error":           r.URL.Query().Get("error"),
		"Success":         r.URL.Query().Get("success"),
		"IsAuthenticated": isAuthenticated,
		"Username":        username,
	}
	role, _ := r.Context().Value("role").(string)
	if role == "admin" {
		data["IsAdmin"] = true
	} else {
		data["IsAdmin"] = false
	}
	if err := tmpl.Execute(w, data); err != nil {
		h.log.Printf("Ошибка рендеринга шаблона: %v", err)
		renderError(w, http.StatusInternalServerError, "500 Internal Server Error", "Внутренняя ошибка сервера", h.projectRoot)
	}
}

// CreateCategory обрабатывает создание новой категории (только для администратора)
func (h *CategoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		renderError(w, http.StatusMethodNotAllowed, "405 Method Not Allowed", "Метод не поддерживается", h.projectRoot)
		return
	}
	// Проверка роли пользователя (должен быть админ)
	role, _ := r.Context().Value("role").(string)
	if role != "admin" {
		renderError(w, http.StatusForbidden, "403 Forbidden", "Доступ запрещён", h.projectRoot)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		http.Redirect(w, r, "/categories?error=Введите название категории", http.StatusSeeOther)
		return
	}
	if err := h.repo.CreateCategory(name); err != nil {
		h.log.Printf("Ошибка создания категории: %v", err)
		http.Redirect(w, r, "/categories?error=Ошибка создания категории", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/categories?success=Категория создана", http.StatusSeeOther)
	return
}
