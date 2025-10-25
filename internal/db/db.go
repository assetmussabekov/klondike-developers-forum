package db

import (
	"database/sql"
	"forum/internal/config"
	"forum/internal/models"
	"log"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

// Repository provides methods for working with the database.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new repository.
func NewRepository(cfg *config.Config) (*Repository, error) {
	db, err := sql.Open("sqlite3", cfg.DBPath)
	if err != nil {
		return nil, err
	}
	return &Repository{db: db}, nil
}

// Close closes the database connection.
func (r *Repository) Close() error {
	return r.db.Close()
}

// CreateUser creates a new user with password hashing.
func (r *Repository) CreateUser(user *models.User, plainPassword string) error {
	// Convert email and username to lowercase for consistency
	user.Email = strings.ToLower(user.Email)
	user.Username = strings.ToLower(user.Username)
	hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hash)
	log.Println("Was")
	_, err = r.db.Exec("INSERT INTO users (email, username, password_hash) VALUES (?, ?, ?)",
		user.Email, user.Username, user.PasswordHash)
	return err
}

// IsEmailOrUsernameTaken checks if an email or username is already taken.
func (r *Repository) IsEmailOrUsernameTaken(email, username string) (bool, error) {
	email = strings.ToLower(email)
	username = strings.ToLower(username)
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ? OR username = ?", email, username).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetUserByEmail retrieves a user by email.
func (r *Repository) GetUserByEmail(email string) (*models.User, error) {
	email = strings.ToLower(email)
	user := &models.User{}
	err := r.db.QueryRow("SELECT id, email, username, password_hash, role, created_at FROM users WHERE email = ?", email).
		Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserByID retrieves a user by ID.
func (r *Repository) GetUserByID(userID int) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRow("SELECT id, email, username, password_hash, role, created_at FROM users WHERE id = ?", userID).
		Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserByUsername retrieves a user by username.
func (r *Repository) GetUserByUsername(username string) (*models.User, error) {
	username = strings.ToLower(username)
	user := &models.User{}
	err := r.db.QueryRow("SELECT id, email, username, password_hash FROM users WHERE username = ?", username).
		Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// CreateSession creates a new session.
func (r *Repository) CreateSession(session *models.Session) error {
	_, err := r.db.Exec("INSERT INTO sessions (session_id, user_id, expires) VALUES (?, ?, ?)",
		session.SessionID, session.UserID, session.Expires)
	return err
}

// GetSession retrieves a session by ID.
func (r *Repository) GetSession(sessionID string) (*models.Session, error) {
	session := &models.Session{}
	err := r.db.QueryRow("SELECT session_id, user_id, expires FROM sessions WHERE session_id = ? AND expires > ?",
		sessionID, time.Now()).Scan(&session.SessionID, &session.UserID, &session.Expires)
	if err != nil {
		return nil, err
	}
	return session, nil
}

// DeleteSession deletes a session.
func (r *Repository) DeleteSession(sessionID string) error {
	_, err := r.db.Exec("DELETE FROM sessions WHERE session_id = ?", sessionID)
	return err
}

// DeleteUserSessions deletes all user sessions except the current one
func (r *Repository) DeleteUserSessions(userID int, exceptSessionID string) error {
	_, err := r.db.Exec("DELETE FROM sessions WHERE user_id = ? AND session_id != ?", userID, exceptSessionID)
	return err
}

// CleanExpiredSessions deletes all expired sessions
func (r *Repository) CleanExpiredSessions() error {
	_, err := r.db.Exec("DELETE FROM sessions WHERE expires < ?", time.Now())
	return err
}

// CreatePost creates a new post.
func (r *Repository) CreatePost(post *models.Post) (int64, error) {
	result, err := r.db.Exec("INSERT INTO posts (user_id, title, content, created_at) VALUES (?, ?, ?, ?)",
		post.UserID, post.Title, post.Content, time.Now())
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// AddPostCategory links a post to a category.
func (r *Repository) AddPostCategory(postID, categoryID int) error {
	_, err := r.db.Exec("INSERT INTO post_categories (post_id, category_id) VALUES (?, ?)", postID, categoryID)
	return err
}

// GetPosts returns a list of posts with filtering.
func (r *Repository) GetPosts(categoryID, sortBy string) ([]*models.Post, error) {
	query := `SELECT p.id, p.user_id, p.title, p.content, p.created_at FROM posts p`
	var args []interface{}

	if categoryID != "" {
		query += ` JOIN post_categories pc ON p.id = pc.post_id WHERE pc.category_id = ?`
		args = append(args, categoryID)
	}

	if sortBy == "date" || sortBy == "" {
		query += ` ORDER BY p.created_at DESC`
	} else if sortBy == "likes" {
		query += ` LEFT JOIN likes l ON p.id = l.post_id
                   GROUP BY p.id ORDER BY COUNT(CASE WHEN l.is_like = 1 THEN 1 END) DESC`
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*models.Post
	for rows.Next() {
		post := &models.Post{}
		err := rows.Scan(&post.ID, &post.UserID, &post.Title, &post.Content, &post.CreatedAt)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, nil
}

// GetPostByID retrieves a post by ID.
func (r *Repository) GetPostByID(postID int) (*models.Post, error) {
	post := &models.Post{}
	err := r.db.QueryRow(`SELECT p.id, p.user_id, p.title, p.content, p.created_at
                          FROM posts p WHERE p.id = ?`, postID).
		Scan(&post.ID, &post.UserID, &post.Title, &post.Content, &post.CreatedAt)
	if err != nil {
		return nil, err
	}
	return post, nil
}

// CreateLike creates or deletes a like.
func (r *Repository) CreateLike(like *models.Like) error {
	var existingID int
	var query string
	var args []interface{}

	if like.PostID != nil {
		query = "SELECT id FROM likes WHERE user_id = ? AND post_id = ?"
		args = []interface{}{like.UserID, *like.PostID}
	} else if like.CommentID != nil {
		query = "SELECT id FROM likes WHERE user_id = ? AND comment_id = ?"
		args = []interface{}{like.UserID, *like.CommentID}
	} else {
		return nil // Do nothing if both IDs are nil
	}

	err := r.db.QueryRow(query, args...).Scan(&existingID)

	if err == nil {
		// Like exists, delete it
		_, err = r.db.Exec("DELETE FROM likes WHERE id = ?", existingID)
		return err
	}

	// Like does not exist, add a new one
	_, err = r.db.Exec("INSERT INTO likes (user_id, post_id, comment_id, is_like) VALUES (?, ?, ?, ?)",
		like.UserID, like.PostID, like.CommentID, like.IsLike)
	return err
}

// GetLikesDislikes returns the number of likes and dislikes for a post.
func (r *Repository) GetLikesDislikes(postID int) (likes, dislikes int, err error) {
	err = r.db.QueryRow(`SELECT COUNT(CASE WHEN is_like = 1 THEN 1 END),
                                COUNT(CASE WHEN is_like = 0 THEN 1 END)
                         FROM likes WHERE post_id = ?`, postID).
		Scan(&likes, &dislikes)
	return
}

// GetCommentLikesDislikes returns the number of likes and dislikes for a comment.
func (r *Repository) GetCommentLikesDislikes(commentID int) (likes, dislikes int, err error) {
	err = r.db.QueryRow(`SELECT COUNT(CASE WHEN is_like = 1 THEN 1 END),
                                COUNT(CASE WHEN is_like = 0 THEN 1 END)
                         FROM likes WHERE comment_id = ?`, commentID).
		Scan(&likes, &dislikes)
	return
}

// CreateComment creates a new comment.
func (r *Repository) CreateComment(comment *models.Comment) error {
	_, err := r.db.Exec("INSERT INTO comments (post_id, user_id, content, created_at) VALUES (?, ?, ?, ?)",
		comment.PostID, comment.UserID, comment.Content, time.Now())
	return err
}

// GetCommentsByPostID returns comments for a post.
func (r *Repository) GetCommentsByPostID(postID int) ([]*models.Comment, error) {
	rows, err := r.db.Query(`SELECT c.id, c.post_id, c.user_id, c.content, c.created_at
                             FROM comments c
                             WHERE c.post_id = ? ORDER BY c.created_at DESC`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*models.Comment
	for rows.Next() {
		comment := &models.Comment{}
		err := rows.Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.Content, &comment.CreatedAt)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	return comments, nil
}

// DeleteComment deletes a comment by ID
func (r *Repository) DeleteComment(commentID int) error {
	_, err := r.db.Exec("DELETE FROM comments WHERE id = ?", commentID)
	return err
}

// UpdateComment updates the comment text and updated_at
func (r *Repository) UpdateComment(commentID int, newContent string) error {
	_, err := r.db.Exec("UPDATE comments SET content = ?, created_at = CURRENT_TIMESTAMP WHERE id = ?", newContent, commentID)
	return err
}

// GetCommentByID returns a comment by ID
func (r *Repository) GetCommentByID(commentID int) (*models.Comment, error) {
	row := r.db.QueryRow("SELECT id, post_id, user_id, content FROM comments WHERE id = ?", commentID)
	c := &models.Comment{}
	if err := row.Scan(&c.ID, &c.PostID, &c.UserID, &c.Content); err != nil {
		return nil, err
	}
	return c, nil
}

// GetAllCategories returns all categories.
func (r *Repository) GetAllCategories() ([]*models.Category, error) {
	rows, err := r.db.Query("SELECT id, name FROM categories ORDER BY name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []*models.Category
	for rows.Next() {
		cat := &models.Category{}
		if err := rows.Scan(&cat.ID, &cat.Name); err != nil {
			return nil, err
		}
		categories = append(categories, cat)
	}
	return categories, nil
}

// CreateCategory creates a new category.
func (r *Repository) CreateCategory(name string) error {
	_, err := r.db.Exec("INSERT INTO categories (name) VALUES (?)", name)
	return err
}

// CategoryExists checks if a category with the specified ID exists.
func (r *Repository) CategoryExists(id int) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM categories WHERE id = ?)"
	err := r.db.QueryRow(query, id).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	return exists, nil
}

// PostExists checks if a post with the specified ID exists.
func (r *Repository) PostExists(id int) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM posts WHERE id = ?)"
	err := r.db.QueryRow(query, id).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	return exists, nil
}

// UpdatePost updates the title, content and updated_at of a post
func (r *Repository) UpdatePost(postID int, title, content string) error {
	_, err := r.db.Exec("UPDATE posts SET title = ?, content = ? WHERE id = ?", title, content, postID)
	return err
}

// DeletePost deletes a post by ID
func (r *Repository) DeletePost(postID int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	// Delete related data
	// Order is important due to foreign keys
	if _, err := tx.Exec("DELETE FROM comments WHERE post_id = ?", postID); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.Exec("DELETE FROM likes WHERE post_id = ?", postID); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.Exec("DELETE FROM post_categories WHERE post_id = ?", postID); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.Exec("DELETE FROM images WHERE post_id = ?", postID); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.Exec("DELETE FROM notifications WHERE post_id = ?", postID); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.Exec("DELETE FROM reports WHERE post_id = ?", postID); err != nil {
		tx.Rollback()
		return err
	}

	// Delete the post itself
	if _, err := tx.Exec("DELETE FROM posts WHERE id = ?", postID); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// AddImage adds an image to a post
func (r *Repository) AddImage(postID int, filePath string) error {
	_, err := r.db.Exec("INSERT INTO images (post_id, file_path, uploaded_at) VALUES (?, ?, CURRENT_TIMESTAMP)", postID, filePath)
	return err
}

// GetImagePathByPostID retrieves the path to the image for a post
func (r *Repository) GetImagePathByPostID(postID int) (string, error) {
	var path string
	err := r.db.QueryRow("SELECT file_path FROM images WHERE post_id = ? ORDER BY uploaded_at DESC LIMIT 1", postID).Scan(&path)
	if err != nil {
		return "", err
	}
	return path, nil
}

// CreateNotification creates a new notification
func (r *Repository) CreateNotification(userID int, notifType string, fromUserID *int, postID *int, commentID *int) error {
	_, err := r.db.Exec(`INSERT INTO notifications (user_id, type, from_user_id, post_id, comment_id, created_at, is_read) VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, 0)`,
		userID, notifType, fromUserID, postID, commentID)
	return err
}

// GetNotificationsByUser retrieves notifications for a user
func (r *Repository) GetNotificationsByUser(userID int) ([]*models.Notification, error) {
	rows, err := r.db.Query(`SELECT id, user_id, type, from_user_id, post_id, comment_id, created_at, is_read FROM notifications WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var notifs []*models.Notification
	for rows.Next() {
		n := &models.Notification{}
		err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.FromUserID, &n.PostID, &n.CommentID, &n.CreatedAt, &n.IsRead)
		if err != nil {
			return nil, err
		}
		notifs = append(notifs, n)
	}
	return notifs, nil
}

// MarkNotificationRead marks a notification as read
func (r *Repository) MarkNotificationRead(notificationID int) error {
	_, err := r.db.Exec(`UPDATE notifications SET is_read = 1 WHERE id = ?`, notificationID)
	return err
}

// CreateReport creates a report on a post or comment
func (r *Repository) CreateReport(reporterID int, postID *int, commentID *int, reason string) error {
	_, err := r.db.Exec(`INSERT INTO reports (reporter_id, post_id, comment_id, reason, created_at, status) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, 'open')`,
		reporterID, postID, commentID, reason)
	return err
}

// GetAllReports returns all reports
func (r *Repository) GetAllReports() ([]*models.Report, error) {
	rows, err := r.db.Query(`SELECT id, reporter_id, post_id, comment_id, reason, created_at, status FROM reports ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reports []*models.Report
	for rows.Next() {
		r := &models.Report{}
		err := rows.Scan(&r.ID, &r.ReporterID, &r.PostID, &r.CommentID, &r.Reason, &r.CreatedAt, &r.Status)
		if err != nil {
			return nil, err
		}
		reports = append(reports, r)
	}
	return reports, nil
}

// CloseReport closes a report
func (r *Repository) CloseReport(reportID int) error {
	_, err := r.db.Exec(`UPDATE reports SET status = 'closed' WHERE id = ?`, reportID)
	return err
}

// GetPostsByUser returns posts by a user
func (r *Repository) GetPostsByUser(userID int) ([]*models.Post, error) {
	rows, err := r.db.Query("SELECT id, user_id, title, content, created_at FROM posts WHERE user_id = ? ORDER BY created_at DESC", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []*models.Post
	for rows.Next() {
		p := &models.Post{}
		err := rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Content, &p.CreatedAt)
		if err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, nil
}

// GetCommentsByUser returns comments by a user
func (r *Repository) GetCommentsByUser(userID int) ([]*models.Comment, error) {
	rows, err := r.db.Query("SELECT id, post_id, user_id, content, created_at FROM comments WHERE user_id = ? ORDER BY created_at DESC", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var comments []*models.Comment
	for rows.Next() {
		c := &models.Comment{}
		err := rows.Scan(&c.ID, &c.PostID, &c.UserID, &c.Content, &c.CreatedAt)
		if err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, nil
}

// GetLikesByUser returns likes/dislikes by a user
func (r *Repository) GetLikesByUser(userID int) ([]*models.Like, error) {
	rows, err := r.db.Query("SELECT id, user_id, post_id, comment_id, is_like, created_at FROM likes WHERE user_id = ? ORDER BY created_at DESC", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var likes []*models.Like
	for rows.Next() {
		l := &models.Like{}
		err := rows.Scan(&l.ID, &l.UserID, &l.PostID, &l.CommentID, &l.IsLike, &l.CreatedAt)
		if err != nil {
			return nil, err
		}
		likes = append(likes, l)
	}
	return likes, nil
}

// GetFirstCategoryByPostID возвращает первую категорию для поста
func (r *Repository) GetFirstCategoryByPostID(postID int) (*models.Category, error) {
	cat := &models.Category{}
	err := r.db.QueryRow(`SELECT c.id, c.name FROM categories c JOIN post_categories pc ON c.id = pc.category_id WHERE pc.post_id = ? LIMIT 1`, postID).Scan(&cat.ID, &cat.Name)
	if err != nil {
		return nil, err
	}
	return cat, nil
}
