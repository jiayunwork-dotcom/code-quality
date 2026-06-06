package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserService struct {
	db     *sql.DB
	cache  map[string]*User
	logger *log.Logger
}

func NewUserService(db *sql.DB) *UserService {
	return &UserService{
		db:    db,
		cache: make(map[string]*User),
	}
}

func (s *UserService) ProcessUsers(users []User, action string, options map[string]interface{},
	validate bool, transform bool, filter bool, sort bool, paginate bool,
	page int, pageSize int) ([]User, error) {
	/*长参数列表函数*/
	var result []User

	for _, user := range users {
		if validate {
			if user.ID <= 0 {
				return nil, fmt.Errorf("invalid user id: %d", user.ID)
			}
			if user.Name == "" {
				return nil, fmt.Errorf("user name is required")
			}
			if user.Email == "" {
				return nil, fmt.Errorf("user email is required")
			}
			if !strings.Contains(user.Email, "@") {
				return nil, fmt.Errorf("invalid email format: %s", user.Email)
			}
		}

		if action == "activate" {
			user.Status = "active"
		} else if action == "deactivate" {
			user.Status = "inactive"
		} else if action == "suspend" {
			user.Status = "suspended"
		} else if action == "delete" {
			continue
		}

		if transform {
			user.Name = strings.ToUpper(user.Name)
			user.Email = strings.ToLower(user.Email)
			if options["prefix"] != nil {
				user.Name = options["prefix"].(string) + user.Name
			}
		}

		if filter {
			statusFilter := options["status"]
			if statusFilter != nil && user.Status != statusFilter.(string) {
				continue
			}
			roleFilter := options["role"]
			if roleFilter != nil && user.Role != roleFilter.(string) {
				continue
			}
		}

		user.UpdatedAt = time.Now()
		result = append(result, user)
	}

	if sort {
		sortBy := "id"
		if options["sort_by"] != nil {
			sortBy = options["sort_by"].(string)
		}
		sortOrder := "asc"
		if options["sort_order"] != nil {
			sortOrder = options["sort_order"].(string)
		}

		for i := 0; i < len(result); i++ {
			for j := i + 1; j < len(result); j++ {
				var compare bool
				switch sortBy {
				case "name":
					if sortOrder == "asc" {
						compare = result[i].Name > result[j].Name
					} else {
						compare = result[i].Name < result[j].Name
					}
				case "email":
					if sortOrder == "asc" {
						compare = result[i].Email > result[j].Email
					} else {
						compare = result[i].Email < result[j].Email
					}
				case "created_at":
					if sortOrder == "asc" {
						compare = result[i].CreatedAt.After(result[j].CreatedAt)
					} else {
						compare = result[i].CreatedAt.Before(result[j].CreatedAt)
					}
				default:
					if sortOrder == "asc" {
						compare = result[i].ID > result[j].ID
					} else {
						compare = result[i].ID < result[j].ID
					}
				}
				if compare {
					result[i], result[j] = result[j], result[i]
				}
			}
		}
	}

	if paginate {
		start := (page - 1) * pageSize
		end := start + pageSize
		if start >= len(result) {
			return []User{}, nil
		}
		if end > len(result) {
			end = len(result)
		}
		result = result[start:end]
	}

	return result, nil
}

func (s *UserService) GetUser(id int, useCache bool, useDB bool, useAPI bool,
	apiURL string, apiKey string, timeout int, retries int) (*User, error) {

	cacheKey := strconv.Itoa(id)

	if useCache {
		if user, ok := s.cache[cacheKey]; ok {
			return user, nil
		}
	}

	var user *User
	var err error

	if useDB {
		user, err = s.fetchFromDB(id)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
	}

	if user == nil && useAPI {
		for i := 0; i < retries; i++ {
			user, err = s.fetchFromAPI(id, apiURL, apiKey, timeout)
			if err == nil {
				break
			}
			time.Sleep(time.Second * time.Duration(i+1))
		}
		if err != nil {
			return nil, fmt.Errorf("failed after %d retries: %w", retries, err)
		}
	}

	if user != nil && useCache {
		s.cache[cacheKey] = user
	}

	return user, nil
}

func (s *UserService) fetchFromDB(id int) (*User, error) {
	return &User{ID: id, Name: "Test"}, nil
}

func (s *UserService) fetchFromAPI(id int, url, key string, timeout int) (*User, error) {
	return &User{ID: id, Name: "API User"}, nil
}

func handleUserRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "User handler")
}
