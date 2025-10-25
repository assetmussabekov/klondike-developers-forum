package models

import "time"

// User represents a forum user
type User struct {
	ID           int
	Email        string
	Username     string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
}

// Post represents a forum post
type Post struct {
	ID        int
	UserID    int
	Title     string
	Content   string
	CreatedAt time.Time
}

// Comment represents a comment to a post
type Comment struct {
	ID        int
	PostID    int
	UserID    int
	Content   string
	CreatedAt time.Time
}

// Category represents a post category
type Category struct {
	ID   int
	Name string
}

// PostCategory links a post and a category
type PostCategory struct {
	PostID     int
	CategoryID int
}

// Like represents a like or dislike to a post or comment
type Like struct {
	ID        int
	UserID    int
	PostID    *int // может быть nil
	CommentID *int // может быть nil
	IsLike    bool
	CreatedAt time.Time
}

// Image represents an image attached to a post
type Image struct {
	ID         int
	PostID     int
	FilePath   string
	UploadedAt time.Time
}

// Notification represents a notification for a user
type Notification struct {
	ID         int
	UserID     int
	Type       string
	FromUserID *int // может быть nil
	PostID     *int // может быть nil
	CommentID  *int // может быть nil
	CreatedAt  time.Time
	IsRead     bool
}

// Report represents a report on a post or comment
type Report struct {
	ID         int
	ReporterID int
	PostID     *int // может быть nil
	CommentID  *int // может быть nil
	Reason     string
	CreatedAt  time.Time
	Status     string
}

// Session represents a user session
type Session struct {
	SessionID string
	UserID    int
	Expires   time.Time
}
