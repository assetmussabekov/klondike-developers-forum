package main

import (
	"forum/internal/config"
	"forum/internal/db"
	"forum/internal/handlers"
	"forum/internal/middleware"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger := log.New(os.Stdout, "forum: ", log.LstdFlags|log.Lshortfile)

	// Initialize repository
	repo, err := db.NewRepository(cfg)
	if err != nil {
		logger.Fatalf("Database initialization error: %v", err)
	}
	defer repo.Close()

	// Run migrations
	if err := repo.RunMigrations(); err != nil {
		logger.Fatalf("Migration error: %v", err)
	}

	// Start periodic session cleanup
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if err := repo.CleanExpiredSessions(); err != nil {
				logger.Printf("Session cleanup error: %v", err)
			}
		}
	}()

	// Create handlers
	authHandler := handlers.NewAuthHandler(repo, logger, cfg.ProjectRoot)
	postHandler := handlers.NewPostHandler(repo, logger, cfg.ProjectRoot)
	likeHandler := handlers.NewLikeHandler(repo, logger, cfg.ProjectRoot)
	commentHandler := handlers.NewCommentHandler(repo, logger, cfg.ProjectRoot)
	categoryHandler := handlers.NewCategoryHandler(repo, logger, cfg.ProjectRoot)
	notificationsHandler := handlers.NewNotificationsHandler(repo, logger, cfg.ProjectRoot)
	reportHandler := handlers.NewReportHandler(repo, logger, cfg.ProjectRoot)
	profileHandler := handlers.NewProfileHandler(repo, logger, cfg.ProjectRoot)

	// Set up routes
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			serveCustom404(w, r, cfg.ProjectRoot)
			return
		}
		postHandler.Posts(w, r)
	})
	mux.HandleFunc("/register", authHandler.Register)
	mux.HandleFunc("/login", authHandler.Login)
	mux.HandleFunc("/logout", authHandler.Logout)
	mux.HandleFunc("/posts", postHandler.Posts) // Posts page
	mux.HandleFunc("/post", postHandler.Post)   // Single post page
	mux.Handle("/create-post", middleware.AuthMiddleware(repo, logger)(http.HandlerFunc(postHandler.CreatePost)))
	mux.Handle("/create-post/", middleware.AuthMiddleware(repo, logger)(http.HandlerFunc(postHandler.CreatePost)))
	mux.Handle("/like", middleware.AuthMiddleware(repo, logger)(http.HandlerFunc(likeHandler.Like)))
	mux.Handle("/comment", middleware.AuthMiddleware(repo, logger)(http.HandlerFunc(commentHandler.AddComment)))
	mux.Handle("/delete-comment", middleware.AuthMiddleware(repo, logger)(http.HandlerFunc(commentHandler.DeleteComment)))
	mux.Handle("/edit-comment", middleware.AuthMiddleware(repo, logger)(http.HandlerFunc(commentHandler.EditComment)))
	mux.Handle("/categories", middleware.AuthMiddleware(repo, logger)(http.HandlerFunc(categoryHandler.CreateCategory)))
	mux.HandleFunc("/categories-list", categoryHandler.ListCategories)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join(cfg.ProjectRoot, "static")))))
	mux.Handle("/edit-post", middleware.AuthMiddleware(repo, logger)(http.HandlerFunc(postHandler.EditPost)))
	mux.Handle("/delete-post", middleware.AuthMiddleware(repo, logger)(http.HandlerFunc(postHandler.DeletePost)))
	mux.Handle("/notifications", middleware.AuthMiddleware(repo, logger)(http.HandlerFunc(notificationsHandler.ListNotifications)))
	mux.Handle("/report", middleware.AuthMiddleware(repo, logger)(http.HandlerFunc(reportHandler.ReportForm)))
	mux.Handle("/submit-report", middleware.AuthMiddleware(repo, logger)(http.HandlerFunc(reportHandler.SubmitReport)))
	mux.Handle("/reports", middleware.AuthMiddleware(repo, logger)(http.HandlerFunc(reportHandler.ListReports)))
	mux.Handle("/close-report", middleware.AuthMiddleware(repo, logger)(http.HandlerFunc(reportHandler.CloseReport)))
	mux.Handle("/profile", middleware.AuthMiddleware(repo, logger)(http.HandlerFunc(profileHandler.Activity)))

	// Start server
	logger.Printf("Server started at http://localhost:8080")

	if err := http.ListenAndServe(":"+cfg.Port, mux); err != nil {
		logger.Fatalf("Server start error: %v", err)
	}
}

// Custom function to serve 404.html
func serveCustom404(w http.ResponseWriter, r *http.Request, projectRoot string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	http.ServeFile(w, r, filepath.Join(projectRoot, "static/404.html"))
}
