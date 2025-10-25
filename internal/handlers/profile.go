package handlers

import (
	"forum/internal/db"
	"html/template"
	"log"
	"net/http"
)

type ProfileHandler struct {
	repo        *db.Repository
	log         *log.Logger
	projectRoot string
}

func NewProfileHandler(repo *db.Repository, log *log.Logger, projectRoot string) *ProfileHandler {
	return &ProfileHandler{repo: repo, log: log, projectRoot: projectRoot}
}

// Activity displays the user's activity page
func (h *ProfileHandler) Activity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		renderError(w, http.StatusMethodNotAllowed, "405 Method Not Allowed", "Метод не поддерживается", h.projectRoot)
		return
	}

	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login?error=Требуется авторизация", http.StatusSeeOther)
		return
	}

	posts, _ := h.repo.GetPostsByUser(userID)
	comments, _ := h.repo.GetCommentsByUser(userID)
	likes, _ := h.repo.GetLikesByUser(userID)

	tmpl, err := template.ParseFiles("static/profile.html")
	if err != nil {
		h.log.Printf("Ошибка загрузки шаблона: %v", err)
		renderError(w, http.StatusInternalServerError, "500 Internal Server Error", "Внутренняя ошибка сервера", h.projectRoot)
		return
	}
	data := map[string]interface{}{
		"Posts":    posts,
		"Comments": comments,
		"Likes":    likes,
	}
	tmpl.Execute(w, data)
}
