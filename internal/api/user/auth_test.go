package user

import (
	"bytes"
	"crowdfunding-backend/internal/model"
	"crowdfunding-backend/internal/service"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserService 是 UserServiceInterface 的模拟实现
type MockUserService struct {
	mock.Mock
}

// 实现 UserServiceInterface 的所有方法...
func (m *MockUserService) Register(user *model.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserService) Login(email, password string) (*model.User, error) {
	args := m.Called(email, password)
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserService) GetUserByID(id int) (*model.User, error) {
	args := m.Called(id)
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserService) UpdateUser(user *model.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserService) VerifyEmail(token string) error {
	args := m.Called(token)
	return args.Error(0)
}

func (m *MockUserService) RequestPasswordReset(email string) error {
	args := m.Called(email)
	return args.Error(0)
}

func (m *MockUserService) ResetPassword(token, newPassword string) error {
	args := m.Called(token, newPassword)
	return args.Error(0)
}

func (m *MockUserService) Logout(userID int) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockUserService) IsTokenBlacklisted(token string) bool {
	args := m.Called(token)
	return args.Bool(0)
}

// 确保 MockUserService 实现了 UserServiceInterface
var _ service.UserServiceInterface = (*MockUserService)(nil)

// TestRegister 测试注册处理器
func TestRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockUserService)
	handler := NewAuthHandler(mockService)

	router := gin.Default()
	router.POST("/register", handler.Register)

	// 模拟成功注册
	mockService.On("Register", mock.AnythingOfType("*model.User")).Return(nil)

	body := []byte(`{"username": "testuser", "email": "test@example.com", "password": "StrongP@ssw0rd"}`)
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockService.AssertExpectations(t)

	// 模拟注册失败（用户名已存在）
	mockService.On("Register", mock.AnythingOfType("*model.User")).Return(errors.New("username already exists"))

	req, _ = http.NewRequest("POST", "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockService.AssertExpectations(t)
}

// TestLogin 测试登录处理器
func TestLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockUserService)
	handler := NewAuthHandler(mockService)

	router := gin.Default()
	router.POST("/login", handler.Login)

	// 模拟成功登录
	mockUser := &model.User{ID: 1, Email: "test@example.com"}
	mockService.On("Login", "test@example.com", "password123").Return(mockUser, nil)

	body := []byte(`{"email": "test@example.com", "password": "password123"}`)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response, "token")
	mockService.AssertExpectations(t)

	// 模拟登录失败
	mockService.On("Login", "test@example.com", "wrongpassword").Return((*model.User)(nil), errors.New("invalid credentials"))

	body = []byte(`{"email": "test@example.com", "password": "wrongpassword"}`)
	req, _ = http.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockService.AssertExpectations(t)
}
