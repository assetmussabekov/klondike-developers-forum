package handlers

import (
	"fmt"
	"forum/internal/db"
	"forum/internal/models"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// PostHandler handles requests related to posts.
type PostHandler struct {
	repo        *db.Repository
	log         *log.Logger
	projectRoot string
}

// NewPostHandler creates a new PostHandler.
func NewPostHandler(repo *db.Repository, log *log.Logger, projectRoot string) *PostHandler {
	return &PostHandler{repo: repo, log: log, projectRoot: projectRoot}
}

// Posts handles displaying the list of posts.
func (h *PostHandler) Posts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		renderError(w, http.StatusMethodNotAllowed, "405 Method Not Allowed", "Method not supported", h.projectRoot)
		return
	}
	categoryID := r.URL.Query().Get("category")
	sortBy := r.URL.Query().Get("sort")

	posts, err := h.repo.GetPosts(categoryID, sortBy)
	if err != nil {
		h.log.Printf("Error loading posts: %v", err)
		renderError(w, http.StatusInternalServerError, "500 Internal Server Error", "Internal server error", h.projectRoot)
		return
	}

	var postViews []*PostView
	for _, post := range posts {
		likes, dislikes, _ := h.repo.GetLikesDislikes(post.ID)
		user, err := h.repo.GetUserByID(post.UserID)
		username := ""
		if err == nil {
			username = user.Username
		}
		category, _ := h.repo.GetFirstCategoryByPostID(post.ID)
		postViews = append(postViews, &PostView{
			ID:        post.ID,
			Title:     post.Title,
			Content:   post.Content,
			CreatedAt: post.CreatedAt,
			Username:  username,
			Likes:     likes,
			Dislikes:  dislikes,
			Category:  category,
		})
	}

	// Manual user determination from cookie for main page and /posts
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

	tmplPath := "static/index.html"
	if r.URL.Path == "/posts" {
		tmplPath = "static/posts.html"
	}

	tmpl, err := template.ParseFiles(filepath.Join(h.projectRoot, tmplPath))
	if err != nil {
		h.log.Printf("Error loading template: %v", err)
		renderError(w, http.StatusInternalServerError, "500 Internal Server Error", "Internal server error", h.projectRoot)
		return
	}

	data := map[string]interface{}{
		"Posts":           postViews,
		"Error":           r.URL.Query().Get("error"),
		"Success":         r.URL.Query().Get("success"),
		"IsAuthenticated": isAuthenticated,
		"Username":        username,
	}
	if err := tmpl.Execute(w, data); err != nil {
		h.log.Printf("Error rendering template: %v", err)
		renderError(w, http.StatusInternalServerError, "500 Internal Server Error", "Internal server error", h.projectRoot)
	}
}

// PostView structure for passing post with likes/dislikes and username to template.
type PostView struct {
	ID        int
	UserID    int
	Title     string
	Content   string
	CreatedAt interface{}
	Username  string
	Likes     int
	Dislikes  int
	ImagePath string
	Category  *models.Category // Added Category field
}

// Post handles displaying a single post.
func (h *PostHandler) Post(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		return
	}
	postID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || postID <= 0 {
		http.Redirect(w, r, "/posts?error=Invalid post ID", http.StatusSeeOther)
		return
	}

	post, err := h.repo.GetPostByID(postID)
	if err != nil {
		h.log.Printf("Error loading post: %v", err)
		http.Redirect(w, r, "/posts?error=Post not found", http.StatusSeeOther)
		return
	}

	likes, dislikes, err := h.repo.GetLikesDislikes(post.ID)
	if err != nil {
		h.log.Printf("Error loading likes/dislikes: %v", err)
	}

	user, err := h.repo.GetUserByID(post.UserID)
	username := ""
	if err == nil {
		username = user.Username
	}

	imagePath, _ := h.repo.GetImagePathByPostID(postID)

	postView := &PostView{
		ID:        post.ID,
		UserID:    post.UserID,
		Title:     post.Title,
		Content:   post.Content,
		CreatedAt: post.CreatedAt,
		Username:  username,
		Likes:     likes,
		Dislikes:  dislikes,
		ImagePath: imagePath,
	}

	comments, err := h.repo.GetCommentsByPostID(postID)
	if err != nil {
		h.log.Printf("Error loading comments: %v", err)
		http.Redirect(w, r, "/posts?error=Error loading comments", http.StatusSeeOther)
		return
	}

	var commentViews []*CommentView
	for _, c := range comments {
		username := ""
		user, err := h.repo.GetUserByID(c.UserID)
		if err == nil {
			username = user.Username
		}

		likes, dislikes, _ := h.repo.GetCommentLikesDislikes(c.ID)

		commentViews = append(commentViews, &CommentView{
			ID:        c.ID,
			PostID:    c.PostID,
			UserID:    c.UserID,
			Username:  username,
			Content:   c.Content,
			CreatedAt: c.CreatedAt,
			Likes:     likes,
			Dislikes:  dislikes,
		})
	}

	// Manual user determination from cookie
	cookie, err := r.Cookie("session_id")
	isAuthenticated := false
	currentUsername := ""
	userID := 0
	role := ""
	if err == nil {
		session, err := h.repo.GetSession(cookie.Value)
		if err == nil {
			user, err := h.repo.GetUserByID(session.UserID)
			if err == nil {
				isAuthenticated = true
				currentUsername = user.Username
				userID = user.ID
				role = user.Role
			}
		}
	}

	tmpl, err := template.ParseFiles(filepath.Join(h.projectRoot, "static", "post.html"))
	if err != nil {
		h.log.Printf("Error loading template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Post":            postView,
		"Comments":        commentViews,
		"Error":           r.URL.Query().Get("error"),
		"Success":         r.URL.Query().Get("success"),
		"IsAuthenticated": isAuthenticated,
		"Username":        currentUsername,
		"UserID":          userID,
		"Role":            role,
	}
	if err := tmpl.Execute(w, data); err != nil {
		h.log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// CreatePost handles post creation.
func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	h.log.Printf("CreatePost called with method: %s", r.Method)
	if r.Method == http.MethodGet {
		userID, ok := r.Context().Value("userID").(int)
		if !ok {
			http.Redirect(w, r, "/login?error=Authentication required", http.StatusSeeOther)
			return
		}
		username := ""
		user, err := h.repo.GetUserByID(userID)
		if err == nil {
			username = user.Username
		}
		tmpl, err := template.ParseFiles(filepath.Join(h.projectRoot, "static", "create_post.html"))
		if err != nil {
			h.log.Printf("Template load error: %v", err)
			renderError(w, http.StatusInternalServerError, "500 Internal Server Error", "Internal server error", h.projectRoot)
			return
		}
		data := map[string]interface{}{
			"Error":           r.URL.Query().Get("error"),
			"Success":         r.URL.Query().Get("success"),
			"IsAuthenticated": true,
			"Username":        username,
		}
		tmpl.Execute(w, data)
		return
	}
	if r.Method == http.MethodPost {
		userID, ok := r.Context().Value("userID").(int)
		if !ok {
			http.Redirect(w, r, "/login?error=Authentication required", http.StatusSeeOther)
			return
		}
		isAuthenticated := true
		username := ""
		user, err := h.repo.GetUserByID(userID)
		if err == nil {
			username = user.Username
		}

		if r.Method == http.MethodPost {
			err := r.ParseMultipartForm(21 << 20) // 21 MB
			if err != nil {
				http.Redirect(w, r, "/create-post?error=Error loading form", http.StatusSeeOther)
				return
			}
			title := strings.TrimSpace(r.FormValue("title"))
			content := strings.TrimSpace(r.FormValue("content"))
			categoryIDs := r.Form["category_ids"]

			titleLen := utf8.RuneCountInString(title)
			contentLen := utf8.RuneCountInString(content)

			if titleLen < 5 || titleLen > 100 {
				http.Redirect(w, r, "/create-post?error=Title must be 5-100 characters", http.StatusSeeOther)
				return
			}
			if contentLen < 10 || contentLen > 5000 {
				http.Redirect(w, r, "/create-post?error=Content must be 10-5000 characters", http.StatusSeeOther)
				return
			}

			if title == "" || content == "" || len(categoryIDs) == 0 {
				http.Redirect(w, r, "/create-post?error=Fill all fields", http.StatusSeeOther)
				return
			}

			// Image handling
			var imagePath string
			file, header, err := r.FormFile("image")
			if err == nil && header != nil {
				defer file.Close()
				if header.Size > 20*1024*1024 {
					http.Redirect(w, r, "/create-post?error=Image too large (max 20 MB)", http.StatusSeeOther)
					return
				}
				ext := strings.ToLower(filepath.Ext(header.Filename))
				if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" {
					http.Redirect(w, r, "/create-post?error=Invalid image format", http.StatusSeeOther)
					return
				}
				buf := make([]byte, 512)
				_, err := file.Read(buf)
				if err != nil {
					http.Redirect(w, r, "/create-post?error=Error reading file", http.StatusSeeOther)
					return
				}
				filetype := http.DetectContentType(buf)
				if !strings.HasPrefix(filetype, "image/") {
					http.Redirect(w, r, "/create-post?error=File is not an image", http.StatusSeeOther)
					return
				}
				file.Seek(0, 0)
				os.MkdirAll("static/uploads", 0755)
				filename := filepath.Join("static/uploads", generateImageName(header.Filename))
				out, err := os.Create(filename)
				if err != nil {
					http.Redirect(w, r, "/create-post?error=Error saving file", http.StatusSeeOther)
					return
				}
				defer out.Close()
				_, err = io.Copy(out, file)
				if err != nil {
					http.Redirect(w, r, "/create-post?error=Error saving file", http.StatusSeeOther)
					return
				}
				imagePath = "/" + filename
			}

			post := &models.Post{
				UserID:  userID,
				Title:   title,
				Content: content,
			}

			postID, err := h.repo.CreatePost(post)
			if err != nil {
				h.log.Printf("Error creating post: %v", err)
				http.Redirect(w, r, "/create-post?error=Error creating post", http.StatusSeeOther)
				return
			}

			for _, catIDStr := range categoryIDs {
				catID, err := strconv.Atoi(catIDStr)
				if err != nil {
					h.log.Printf("Invalid category ID: %v", err)
					continue
				}

				exists, err := h.repo.CategoryExists(catID)
				if err != nil {
					h.log.Printf("Error checking category: %v", err)
					http.Redirect(w, r, "/create-post?error=Error checking category", http.StatusSeeOther)
					return
				}
				if !exists {
					http.Redirect(w, r, "/create-post?error=Selected non-existent category", http.StatusSeeOther)
					return
				}

				if err := h.repo.AddPostCategory(int(postID), catID); err != nil {
					h.log.Printf("Error adding category: %v", err)
					continue
				}
			}

			if imagePath != "" {
				if err := h.repo.AddImage(int(postID), imagePath); err != nil {
					h.log.Printf("Error saving image: %v", err)
				}
			}

			h.log.Printf("Post %s created by user %d", title, userID)
			http.Redirect(w, r, "/?success=Post successfully created", http.StatusSeeOther)
			return
		}

		tmpl, err := template.ParseFiles(filepath.Join(h.projectRoot, "static", "create_post.html"))
		if err != nil {
			h.log.Printf("Error loading template: %v", err)
			renderError(w, http.StatusInternalServerError, "500 Internal Server Error", "Internal server error", h.projectRoot)
			return
		}

		data := map[string]interface{}{
			"Error":           r.URL.Query().Get("error"),
			"Success":         r.URL.Query().Get("success"),
			"IsAuthenticated": isAuthenticated,
			"Username":        username,
		}
		if err := tmpl.Execute(w, data); err != nil {
			h.log.Printf("Error rendering template: %v", err)
			renderError(w, http.StatusInternalServerError, "500 Internal Server Error", "Internal server error", h.projectRoot)
		}
	}
	renderError(w, http.StatusMethodNotAllowed, "405 Method Not Allowed", "Method not allowed", h.projectRoot)
	h.log.Printf("405 returned for method: %s", r.Method)
}

// EditPost edits a post (only author, moderator, or admin)
func (h *PostHandler) EditPost(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login?error=Authentication required", http.StatusSeeOther)
		return
	}
	role, _ := r.Context().Value("role").(string)

	postID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || postID <= 0 {
		http.Redirect(w, r, "/posts?error=Invalid post ID", http.StatusSeeOther)
		return
	}

	post, err := h.repo.GetPostByID(postID)
	if err != nil {
		http.Redirect(w, r, "/posts?error=Post not found", http.StatusSeeOther)
		return
	}
	if post.UserID != userID && role != "admin" && role != "moderator" {
		renderError(w, http.StatusForbidden, "403 Forbidden", "No permission to edit", h.projectRoot)
		return
	}

	if r.Method == http.MethodGet {
		tmpl, err := template.ParseFiles(filepath.Join(h.projectRoot, "static", "edit_post.html"))
		if err != nil {
			h.log.Printf("Template load error: %v", err)
			renderError(w, http.StatusInternalServerError, "500 Internal Server Error", "Internal server error", h.projectRoot)
			return
		}
		data := map[string]interface{}{
			"Post":  post,
			"Error": r.URL.Query().Get("error"),
		}
		tmpl.Execute(w, data)
		return
	}
	if r.Method == http.MethodPost {
		title := strings.TrimSpace(r.FormValue("title"))
		content := strings.TrimSpace(r.FormValue("content"))

		titleLen := utf8.RuneCountInString(title)
		contentLen := utf8.RuneCountInString(content)

		if titleLen < 5 || titleLen > 100 {
			http.Redirect(w, r, "/edit-post?id="+strconv.Itoa(postID)+"&error=Title must be 5-100 characters", http.StatusSeeOther)
			return
		}
		if contentLen < 10 || contentLen > 5000 {
			http.Redirect(w, r, "/edit-post?id="+strconv.Itoa(postID)+"&error=Content must be 10-5000 characters", http.StatusSeeOther)
			return
		}
		if title == "" || content == "" {
			http.Redirect(w, r, "/edit-post?id="+strconv.Itoa(postID)+"&error=Fill all fields", http.StatusSeeOther)
			return
		}
		if err := h.repo.UpdatePost(postID, title, content); err != nil {
			h.log.Printf("Post update error: %v", err)
			http.Redirect(w, r, "/edit-post?id="+strconv.Itoa(postID)+"&error=Update error", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/post?id="+strconv.Itoa(postID)+"&success=Post updated", http.StatusSeeOther)
		return
	}
	renderError(w, http.StatusMethodNotAllowed, "405 Method Not Allowed", "Method not allowed", h.projectRoot)
}

// DeletePost deletes a post (only author, moderator, or admin)
func (h *PostHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		return
	}
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login?error=Authentication required", http.StatusSeeOther)
		return
	}
	role, _ := r.Context().Value("role").(string)

	postID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || postID <= 0 {
		http.Redirect(w, r, "/posts?error=Invalid post ID", http.StatusSeeOther)
		return
	}

	post, err := h.repo.GetPostByID(postID)
	if err != nil {
		http.Redirect(w, r, "/posts?error=Post not found", http.StatusSeeOther)
		return
	}
	if post.UserID != userID && role != "admin" && role != "moderator" {
		http.Error(w, "No permission to delete", http.StatusForbidden)
		return
	}
	if err := h.repo.DeletePost(postID); err != nil {
		h.log.Printf("Error deleting post: %v", err)
		http.Redirect(w, r, "/post?id="+strconv.Itoa(postID)+"&error=Error deleting post", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/posts?success=Post deleted", http.StatusSeeOther)
}

// Helper function to generate a unique file name
func generateImageName(original string) string {
	ext := filepath.Ext(original)
	name := strings.TrimSuffix(original, ext)
	hash := fmt.Sprintf("%x", time.Now().UnixNano())
	return name + "_" + hash + ext
}

type CommentView struct {
	ID        int
	PostID    int
	UserID    int
	Username  string
	Content   string
	CreatedAt interface{}
	Likes     int
	Dislikes  int
}
