package handlers

import (
	"encoding/json"
	"forum/internal/db"
	"forum/internal/models"
	"log"
	"net/http"
	"strconv"
)

type LikeHandler struct {
	repo        *db.Repository
	log         *log.Logger
	projectRoot string
}

func NewLikeHandler(repo *db.Repository, log *log.Logger, projectRoot string) *LikeHandler {
	return &LikeHandler{repo: repo, log: log, projectRoot: projectRoot}
}

func (h *LikeHandler) Like(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	postIDStr := r.URL.Query().Get("post_id")
	commentIDStr := r.URL.Query().Get("comment_id")
	isLike, err := strconv.ParseBool(r.URL.Query().Get("is_like"))
	if err != nil {
		http.Error(w, "Invalid is_like parameter", http.StatusBadRequest)
		return
	}

	like := &models.Like{
		UserID: userID,
		IsLike: isLike,
	}

	var postID, commentID int
	if postIDStr != "" {
		postID, err = strconv.Atoi(postIDStr)
		if err != nil {
			http.Error(w, "Invalid post ID", http.StatusBadRequest)
			return
		}
		like.PostID = &postID
	} else if commentIDStr != "" {
		commentID, err = strconv.Atoi(commentIDStr)
		if err != nil {
			http.Error(w, "Invalid comment ID", http.StatusBadRequest)
			return
		}
		like.CommentID = &commentID
	} else {
		http.Error(w, "Post ID or comment ID missing", http.StatusBadRequest)
		return
	}

	// Check existence of post or comment
	if like.PostID != nil {
		exists, err := h.repo.PostExists(*like.PostID)
		if err != nil {
			h.log.Printf("Error checking post: %v", err)
			http.Error(w, "Error checking post", http.StatusInternalServerError)
			return
		}
		if !exists {
			http.Error(w, "Post does not exist", http.StatusNotFound)
			return
		}
	} else if like.CommentID != nil {
		comment, err := h.repo.GetCommentByID(*like.CommentID)
		if err != nil {
			h.log.Printf("Error checking comment: %v", err)
			http.Error(w, "Comment does not exist", http.StatusNotFound)
			return
		}
		// Additional check for post existence to which the comment belongs
		exists, err := h.repo.PostExists(comment.PostID)
		if err != nil {
			h.log.Printf("Error checking post of comment: %v", err)
			http.Error(w, "Error checking post", http.StatusInternalServerError)
			return
		}
		if !exists {
			http.Error(w, "Comment post does not exist", http.StatusNotFound)
			return
		}
	}

	if err := h.repo.CreateLike(like); err != nil {
		h.log.Printf("Error processing like: %v", err)
		http.Error(w, "Error processing like", http.StatusInternalServerError)
		return
	}

	// Notifications
	if like.PostID != nil {
		post, err := h.repo.GetPostByID(*like.PostID)
		if err == nil && post.UserID != userID {
			nType := "like"
			if !like.IsLike {
				nType = "dislike"
			}
			fromUserID := userID
			h.repo.CreateNotification(post.UserID, nType, &fromUserID, like.PostID, nil)
		}
	} else if like.CommentID != nil {
		comment, err := h.repo.GetCommentByID(*like.CommentID)
		if err == nil && comment.UserID != userID {
			nType := "like"
			if !like.IsLike {
				nType = "dislike"
			}
			fromUserID := userID
			h.repo.CreateNotification(comment.UserID, nType, &fromUserID, nil, like.CommentID)
		}
	}

	var likes, dislikes int
	if like.PostID != nil {
		likes, dislikes, _ = h.repo.GetLikesDislikes(*like.PostID)
	} else if like.CommentID != nil {
		likes, dislikes, _ = h.repo.GetCommentLikesDislikes(*like.CommentID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"likes":    likes,
		"dislikes": dislikes,
	})
}
