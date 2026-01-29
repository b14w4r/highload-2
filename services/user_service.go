package services

import (
	"errors"
	"sync"
	"sync/atomic"

	"go-microservice/models"
)

var (
	ErrNotFound = errors.New("user not found")
)

type UserService interface {
	List() []models.User
	Get(id int) (models.User, error)
	Create(u models.User) (models.User, error)
	Update(id int, u models.User) (models.User, error)
	Delete(id int) error
}

type InMemoryUserService struct {
	mu     sync.RWMutex
	nextID int64
	items  map[int]models.User
}

func NewInMemoryUserService() *InMemoryUserService {
	return &InMemoryUserService{
		nextID: 0,
		items:  make(map[int]models.User),
	}
}

func (s *InMemoryUserService) List() []models.User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]models.User, 0, len(s.items))
	for _, u := range s.items {
		out = append(out, u)
	}
	return out
}

func (s *InMemoryUserService) Get(id int) (models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.items[id]
	if !ok {
		return models.User{}, ErrNotFound
	}
	return u, nil
}

func (s *InMemoryUserService) Create(u models.User) (models.User, error) {
	if err := u.Validate(); err != nil {
		return models.User{}, err
	}

	id := int(atomic.AddInt64(&s.nextID, 1))
	u.ID = id

	s.mu.Lock()
	s.items[id] = u
	s.mu.Unlock()

	return u, nil
}

func (s *InMemoryUserService) Update(id int, u models.User) (models.User, error) {
	if err := u.Validate(); err != nil {
		return models.User{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[id]; !ok {
		return models.User{}, ErrNotFound
	}

	u.ID = id
	s.items[id] = u
	return u, nil
}

func (s *InMemoryUserService) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[id]; !ok {
		return ErrNotFound
	}
	delete(s.items, id)
	return nil
}

