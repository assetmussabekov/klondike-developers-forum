package handlers

import (
	"forum/internal/db"
	"forum/internal/models"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"html/template"
	"path/filepath"

	"unicode/utf8"

	"github.com/google/uuid"
)

var loginAttempts = make(map[string][]time.Time) // username -> slice of failed attempt times
const maxLoginAttempts = 5
const loginBlockDuration = 10 * time.Minute

type AuthHandler struct {
	repo        *db.Repository
	log         *log.Logger
	projectRoot string
}

func NewAuthHandler(repo *db.Repository, log *log.Logger, projectRoot string) *AuthHandler {
	return &AuthHandler{repo: repo, log: log, projectRoot: projectRoot}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	// Проверка: если пользователь уже залогинен, редирект на главную
	cookie, err := r.Cookie("session_id")
	if err == nil {
		_, err := h.repo.GetSession(cookie.Value)
		if err == nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}
	if r.Method == http.MethodGet {
		tmpl, err := template.ParseFiles(filepath.Join(h.projectRoot, "static", "register.html"))
		if err != nil {
			h.log.Printf("Ошибка загрузки шаблона: %v", err)
			http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
			return
		}
		data := map[string]interface{}{
			"Error":   r.URL.Query().Get("error"),
			"Success": r.URL.Query().Get("success"),
		}
		tmpl.Execute(w, data)
		return
	}
	if r.Method == http.MethodPost {
		email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
		username := strings.ReplaceAll(strings.TrimSpace(strings.ToLower(r.FormValue("username"))), " ", "")
		password := strings.TrimSpace(r.FormValue("password"))
		log.Println("checked")
		// Проверка на пустой email/username
		if email == "" || username == "" {
			http.Redirect(w, r, "/register?error=Email и username не могут быть пустыми", http.StatusSeeOther)
			return
		}
		// Проверка длины
		if runeLen := utf8.RuneCountInString(username); runeLen < 3 || runeLen > 30 {
			http.Redirect(w, r, "/register?error=Имя пользователя должно быть от 3 до 30 символов", http.StatusSeeOther)
			return
		}
		// Проверка длины пароля по количеству рун
		if runeLen := utf8.RuneCountInString(password); runeLen < 6 || runeLen > 50 {
			http.Redirect(w, r, "/register?error=Пароль должен быть от 6 до 50 символов", http.StatusSeeOther)
			return
		}
		// Проверка email с помощью регулярного выражения
		emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
		matched := false
		if re := regexp.MustCompile(emailRegex); re.MatchString(email) {
			matched = true
		}
		if !matched {
			http.Redirect(w, r, "/register?error=Некорректный формат email", http.StatusSeeOther)
			return
		}
		// Проверка уникальности email/username
		isTaken, err := h.repo.IsEmailOrUsernameTaken(email, username)
		if err != nil {
			h.log.Printf("Ошибка проверки уникальности: %v", err)
			http.Redirect(w, r, "/register?error=Ошибка регистрации", http.StatusSeeOther)
			return
		}
		if isTaken {
			http.Redirect(w, r, "/register?error=Email или username уже заняты", http.StatusSeeOther)
			return
		}

		user := &models.User{
			Email:    email,
			Username: username,
		}
		log.Println("not created")
		if err := h.repo.CreateUser(user, password); err != nil {
			h.log.Printf("Ошибка добавления пользователя: %v", err)
			http.Redirect(w, r, "/register?error=Ошибка регистрации", http.StatusSeeOther)
			return
		}
		log.Println("created")
		h.log.Printf("Пользователь зарегистрирован: %s", username)
		http.Redirect(w, r, "/login?success=Регистрация успешна", http.StatusSeeOther)
		return
	}
	http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	// Проверка: если пользователь уже залогинен, редирект на главную
	cookie, err := r.Cookie("session_id")
	if err == nil {
		_, err := h.repo.GetSession(cookie.Value)
		if err == nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}
	if r.Method == http.MethodGet {
		tmpl, err := template.ParseFiles(filepath.Join(h.projectRoot, "static", "login.html"))
		if err != nil {
			h.log.Printf("Ошибка загрузки шаблона: %v", err)
			http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
			return
		}
		data := map[string]interface{}{
			"Error":   r.URL.Query().Get("error"),
			"Success": r.URL.Query().Get("success"),
		}
		tmpl.Execute(w, data)
		return
	}
	if r.Method == http.MethodPost {
		username := strings.ReplaceAll(strings.TrimSpace(strings.ToLower(r.FormValue("username"))), " ", "")
		password := r.FormValue("password")

		// Проверка блокировки
		now := time.Now()
		attempts := loginAttempts[username]
		// Оставляем только попытки за последние 10 минут
		var recent []time.Time
		for _, t := range attempts {
			if now.Sub(t) < loginBlockDuration {
				recent = append(recent, t)
			}
		}
		if len(recent) >= maxLoginAttempts {
			http.Redirect(w, r, "/login?error=Слишком много неудачных попыток. Попробуйте через 10 минут.", http.StatusSeeOther)
			return
		}
		loginAttempts[username] = recent

		user, err := h.repo.GetUserByUsername(username)
		if err != nil {
			h.log.Printf("Ошибка входа для пользователя %s: %v", username, err)
			loginAttempts[username] = append(loginAttempts[username], now)
			http.Redirect(w, r, "/login?error=Неверный логин или пароль", http.StatusSeeOther)
			return
		}

		if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
			h.log.Printf("Ошибка входа: неверный пароль для пользователя %s", username)
			loginAttempts[username] = append(loginAttempts[username], now)
			http.Redirect(w, r, "/login?error=Неверный логин или пароль", http.StatusSeeOther)
			return
		}

		// Успешный вход — сбрасываем попытки
		delete(loginAttempts, username)

		sessionID := uuid.New().String()
		expiresAt := time.Now().Add(24 * time.Hour)
		session := &models.Session{
			SessionID: sessionID,
			UserID:    user.ID,
			Expires:   expiresAt,
		}

		if err := h.repo.CreateSession(session); err != nil {
			h.log.Printf("Ошибка создания сессии: %v", err)
			http.Redirect(w, r, "/login?error=Ошибка входа", http.StatusSeeOther)
			return
		}

		// Удаляем все другие сессии этого пользователя
		if err := h.repo.DeleteUserSessions(user.ID, sessionID); err != nil {
			h.log.Printf("Ошибка удаления старых сессий: %v", err)
		}

		http.SetCookie(w, &http.Cookie{
			Name:    "session_id",
			Value:   sessionID,
			Path:    "/",
			Expires: expiresAt,
		})

		h.log.Printf("Пользователь вошел: %s", username)
		http.Redirect(w, r, "/?success=Вход выполнен", http.StatusSeeOther)
		return
	}
	http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	sessionID, err := r.Cookie("session_id")
	if err == nil {
		if err := h.repo.DeleteSession(sessionID.Value); err != nil {
			h.log.Printf("Ошибка удаления сессии: %v", err)
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "session_id",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	h.log.Printf("Пользователь вышел из системы")
	http.Redirect(w, r, "/?success=Вы успешно вышли", http.StatusSeeOther)
}
