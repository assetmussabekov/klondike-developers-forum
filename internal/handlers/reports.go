package handlers

import (
	"forum/internal/db"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
)

type ReportHandler struct {
	repo        *db.Repository
	log         *log.Logger
	projectRoot string
}

func NewReportHandler(repo *db.Repository, log *log.Logger, projectRoot string) *ReportHandler {
	return &ReportHandler{repo: repo, log: log, projectRoot: projectRoot}
}

// ReportForm displays the report submission form
func (h *ReportHandler) ReportForm(w http.ResponseWriter, r *http.Request) {
	isAuthenticated := r.Context().Value("userID") != nil
	if !isAuthenticated {
		http.Redirect(w, r, "/login?error=Требуется авторизация", http.StatusSeeOther)
		return
	}

	tmpl, err := template.ParseFiles(filepath.Join(h.projectRoot, "static", "report_form.html"))
	if err != nil {
		h.log.Printf("Ошибка загрузки шаблона: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}
	data := map[string]interface{}{
		"PostID":    r.URL.Query().Get("post_id"),
		"CommentID": r.URL.Query().Get("comment_id"),
		"Error":     r.URL.Query().Get("error"),
	}
	tmpl.Execute(w, data)
}

// SubmitReport handles report submission
func (h *ReportHandler) SubmitReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login?error=Требуется авторизация", http.StatusSeeOther)
		return
	}

	postIDStr := r.FormValue("post_id")
	commentIDStr := r.FormValue("comment_id")
	reason := r.FormValue("reason")
	if reason == "" {
		http.Redirect(w, r, r.Referer()+"?error=Укажите причину", http.StatusSeeOther)
		return
	}

	var postIDPtr, commentIDPtr *int
	if postIDStr != "" {
		pid, err := strconv.Atoi(postIDStr)
		if err == nil {
			postIDPtr = &pid
		}
	}
	if commentIDStr != "" {
		cid, err := strconv.Atoi(commentIDStr)
		if err == nil {
			commentIDPtr = &cid
		}
	}

	if postIDPtr == nil && commentIDPtr == nil {
		http.Redirect(w, r, "/?error=Ошибка жалобы", http.StatusSeeOther)
		return
	}

	if err := h.repo.CreateReport(userID, postIDPtr, commentIDPtr, reason); err != nil {
		h.log.Printf("Ошибка создания жалобы: %v", err)
		http.Redirect(w, r, "/?error=Ошибка создания жалобы", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/?success=Жалоба отправлена", http.StatusSeeOther)
}

// ListReports displays all reports (only for moderator and admin)
func (h *ReportHandler) ListReports(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}
	role, _ := r.Context().Value("role").(string)
	if role != "admin" && role != "moderator" {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}

	reports, err := h.repo.GetAllReports()
	if err != nil {
		h.log.Printf("Ошибка получения всех жалоб: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles(filepath.Join(h.projectRoot, "static", "reports.html"))
	if err != nil {
		h.log.Printf("Ошибка загрузки шаблона: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}
	data := map[string]interface{}{
		"Reports": reports,
	}
	tmpl.Execute(w, data)
}

// CloseReport handles report closure (only for moderator and admin)
func (h *ReportHandler) CloseReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}
	role, _ := r.Context().Value("role").(string)
	if role != "admin" && role != "moderator" {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}
	reportIDStr := r.URL.Query().Get("id")
	if reportIDStr == "" {
		http.Redirect(w, r, "/reports?error=Нет id жалобы", http.StatusSeeOther)
		return
	}
	reportID, err := strconv.Atoi(reportIDStr)
	if err != nil {
		http.Redirect(w, r, "/reports?error=Некорректный id", http.StatusSeeOther)
		return
	}
	if err := h.repo.CloseReport(reportID); err != nil {
		h.log.Printf("Ошибка закрытия жалобы: %v", err)
		http.Redirect(w, r, "/reports?error=Ошибка закрытия", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/reports?success=Жалоба закрыта", http.StatusSeeOther)
}
