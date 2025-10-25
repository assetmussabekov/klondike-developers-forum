package middleware

import (
	"context"
	"forum/internal/db"
	"log"
	"net/http"
)

func AuthMiddleware(repo *db.Repository, logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_id")
			if err != nil {
				http.Redirect(w, r, "/login?error=Требуется авторизация", http.StatusSeeOther)
				return
			}

			session, err := repo.GetSession(cookie.Value)
			if err != nil {
				http.Redirect(w, r, "/login?error=Сессия недействительна", http.StatusSeeOther)
				return
			}
			ctx := r.Context()
			ctx = context.WithValue(ctx, "userID", session.UserID)

			// Получаем роль пользователя и добавляем в контекст
			user, err := repo.GetUserByID(session.UserID)
			if err == nil {
				ctx = context.WithValue(ctx, "role", user.Role)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
