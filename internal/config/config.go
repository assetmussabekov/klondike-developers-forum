package config

import (
	"os"
	"path/filepath"
)

// Config хранит конфигурацию приложения.
type Config struct {
	Port        string
	DBPath      string
	ProjectRoot string
}

// Load загружает конфигурацию из переменных окружения или использует значения по умолчанию.
func Load() *Config {
	// Ищем корень проекта, двигаясь вверх от текущей директории до нахождения go.mod
	dir, err := os.Getwd()
	if err != nil {
		// В случае ошибки используем текущую директорию как запасной вариант
		dir = "."
	}

	var projectRoot string
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			projectRoot = dir // Нашли корень проекта
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			projectRoot = "." // Дошли до корня файловой системы, используем текущую директорию
			break
		}
		dir = parent
	}

	absDBPath := filepath.Join(projectRoot, "forum.db")

	return &Config{
		Port:        getEnv("PORT", "8080"),
		DBPath:      getEnv("DB_PATH", absDBPath),
		ProjectRoot: projectRoot,
	}
}

// getEnv получает значение переменной окружения или возвращает значение по умолчанию.
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
