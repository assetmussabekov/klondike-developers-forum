package handlers

import (
	"forum/internal/db"
	"forum/internal/models"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"unicode/utf8"
)

type CommentHandler struct {
	repo        *db.Repository
	log         *log.Logger
	projectRoot string
}

func NewCommentHandler(repo *db.Repository, log *log.Logger, projectRoot string) *CommentHandler {
	return &CommentHandler{repo: repo, log: log, projectRoot: projectRoot}
}

func (h *CommentHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login?error=Требуется авторизация", http.StatusSeeOther)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	postID, err := strconv.Atoi(r.FormValue("post_id"))
	if err != nil || postID <= 0 {
		http.Redirect(w, r, "/posts?error=Неверный ID поста", http.StatusSeeOther)
		return
	}

	content := strings.TrimSpace(r.FormValue("content"))
	if content == "" {
		http.Redirect(w, r, "/post?id="+strconv.Itoa(postID)+"&error=Комментарий не может быть пустым", http.StatusSeeOther)
		return
	}

	contentLen := utf8.RuneCountInString(content)
	if contentLen < 2 || contentLen > 1000 {
		http.Redirect(w, r, "/post?id="+strconv.Itoa(postID)+"&error=Комментарий должен быть от 2 до 1000 символов", http.StatusSeeOther)
		return
	}

	// Check post existence
	exists, err := h.repo.PostExists(postID)
	if err != nil {
		h.log.Printf("Ошибка проверки поста: %v", err)
		http.Redirect(w, r, "/posts?error=Ошибка проверки поста", http.StatusSeeOther)
		return
	}
	if !exists {
		http.Redirect(w, r, "/posts?error=Пост не существует", http.StatusSeeOther)
		return
	}

	comment := &models.Comment{
		PostID:  postID,
		UserID:  userID,
		Content: content,
	}

	if err := h.repo.CreateComment(comment); err != nil {
		h.log.Printf("Ошибка создания комментария: %v", err)
		http.Redirect(w, r, "/post?id="+strconv.Itoa(postID)+"&error=Ошибка создания комментария", http.StatusSeeOther)
		return
	}

	// Notify post author
	post, err := h.repo.GetPostByID(postID)
	if err == nil && post.UserID != userID {
		fromUserID := userID
		h.repo.CreateNotification(post.UserID, "comment", &fromUserID, &postID, nil)
	}

	h.log.Printf("Комментарий добавлен к посту %d пользователем %d", postID, userID)
	http.Redirect(w, r, "/post?id="+strconv.Itoa(postID)+"&success=Комментарий успешно добавлен", http.StatusSeeOther)
}

// DeleteComment deletes a comment (only author, moderator, or admin)
func (h *CommentHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login?error=Требуется авторизация", http.StatusSeeOther)
		return
	}
	role, _ := r.Context().Value("role").(string)

	commentID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || commentID <= 0 {
		http.Redirect(w, r, "/posts?error=Неверный ID комментария", http.StatusSeeOther)
		return
	}

	comment, err := h.repo.GetCommentByID(commentID)
	if err != nil {
		http.Redirect(w, r, "/posts?error=Комментарий не найден", http.StatusSeeOther)
		return
	}
	if comment.UserID != userID && role != "admin" && role != "moderator" {
		http.Error(w, "Нет прав на удаление", http.StatusForbidden)
		return
	}
	if err := h.repo.DeleteComment(commentID); err != nil {
		h.log.Printf("Ошибка удаления комментария: %v", err)
		http.Redirect(w, r, "/post?id="+strconv.Itoa(comment.PostID)+"&error=Ошибка удаления комментария", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/post?id="+strconv.Itoa(comment.PostID)+"&success=Комментарий удалён", http.StatusSeeOther)
}

// EditComment edits a comment (only author, moderator, or admin)
func (h *CommentHandler) EditComment(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login?error=Требуется авторизация", http.StatusSeeOther)
		return
	}
	role, _ := r.Context().Value("role").(string)

	commentID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || commentID <= 0 {
		http.Redirect(w, r, "/posts?error=Неверный ID комментария", http.StatusSeeOther)
		return
	}

	comment, err := h.repo.GetCommentByID(commentID)
	if err != nil {
		http.Redirect(w, r, "/posts?error=Комментарий не найден", http.StatusSeeOther)
		return
	}
	if comment.UserID != userID && role != "admin" && role != "moderator" {
		http.Error(w, "Нет прав на редактирование", http.StatusForbidden)
		return
	}

	if r.Method == http.MethodPost {
		newContent := strings.TrimSpace(r.FormValue("content"))
		if newContent == "" {
			http.Redirect(w, r, "/edit-comment?id="+strconv.Itoa(commentID)+"&error=Текст не может быть пустым", http.StatusSeeOther)
			return
		}
		contentLen := utf8.RuneCountInString(newContent)
		if contentLen < 2 || contentLen > 1000 {
			http.Redirect(w, r, "/edit-comment?id="+strconv.Itoa(commentID)+"&error=Комментарий должен быть от 2 до 1000 символов", http.StatusSeeOther)
			return
		}
		if err := h.repo.UpdateComment(commentID, newContent); err != nil {
			h.log.Printf("Ошибка обновления комментария: %v", err)
			http.Redirect(w, r, "/edit-comment?id="+strconv.Itoa(commentID)+"&error=Ошибка обновления", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/post?id="+strconv.Itoa(comment.PostID)+"&success=Комментарий обновлён", http.StatusSeeOther)
		return
	}

	tmpl, err := template.ParseFiles(filepath.Join(h.projectRoot, "static", "edit_comment.html"))
	if err != nil {
		h.log.Printf("Ошибка загрузки шаблона: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}
	data := map[string]interface{}{
		"Comment": comment,
		"Error":   r.URL.Query().Get("error"),
	}
	tmpl.Execute(w, data)
}
