package db

import (
	"forum/internal/config"
	"forum/internal/models"
	"testing"
)

func setupTestRepo(t *testing.T) *Repository {
	repo, err := NewRepository(&config.Config{DBPath: ":memory:"})
	if err != nil {
		t.Fatalf("Ошибка создания репозитория: %v", err)
	}
	if err := repo.RunMigrations(); err != nil {
		t.Fatalf("Ошибка миграций: %v", err)
	}
	return repo
}

func TestCreateUser(t *testing.T) {
	repo := setupTestRepo(t)
	defer repo.Close()

	user := &models.User{
		Email:    "test@example.com",
		Username: "testuser",
	}

	err := repo.CreateUser(user, "password123")
	if err != nil {
		t.Errorf("Ошибка создания пользователя: %v", err)
	}
}

func TestCreatePost(t *testing.T) {
	repo := setupTestRepo(t)
	defer repo.Close()
	user := &models.User{Email: "a@b.c", Username: "a"}
	repo.CreateUser(user, "pass")
	u, _ := repo.GetUserByEmail("a@b.c")
	post := &models.Post{UserID: u.ID, Title: "Test Post", Content: "Hello"}
	_, err := repo.CreatePost(post)
	if err != nil {
		t.Errorf("Ошибка создания поста: %v", err)
	}
}

func TestCreateComment(t *testing.T) {
	repo := setupTestRepo(t)
	defer repo.Close()
	user := &models.User{Email: "b@b.c", Username: "b"}
	repo.CreateUser(user, "pass")
	u, _ := repo.GetUserByEmail("b@b.c")
	post := &models.Post{UserID: u.ID, Title: "Test Post", Content: "Hello"}
	pid, _ := repo.CreatePost(post)
	comment := &models.Comment{PostID: int(pid), UserID: u.ID, Content: "Nice!"}
	err := repo.CreateComment(comment)
	if err != nil {
		t.Errorf("Ошибка создания комментария: %v", err)
	}
}

func TestCreateLike(t *testing.T) {
	repo := setupTestRepo(t)
	defer repo.Close()
	user := &models.User{Email: "c@b.c", Username: "c"}
	repo.CreateUser(user, "pass")
	u, _ := repo.GetUserByEmail("c@b.c")
	post := &models.Post{UserID: u.ID, Title: "Test Post", Content: "Hello"}
	pid, _ := repo.CreatePost(post)
	like := &models.Like{UserID: u.ID, PostID: &[]int{int(pid)}[0], IsLike: true}
	err := repo.CreateLike(like)
	if err != nil {
		t.Errorf("Ошибка создания лайка: %v", err)
	}
}

func TestCreateReport(t *testing.T) {
	repo := setupTestRepo(t)
	defer repo.Close()
	user := &models.User{Email: "d@b.c", Username: "d"}
	repo.CreateUser(user, "pass")
	u, _ := repo.GetUserByEmail("d@b.c")
	post := &models.Post{UserID: u.ID, Title: "Test Post", Content: "Hello"}
	pid, _ := repo.CreatePost(post)
	err := repo.CreateReport(u.ID, &[]int{int(pid)}[0], nil, "Спам")
	if err != nil {
		t.Errorf("Ошибка создания жалобы: %v", err)
	}
}
