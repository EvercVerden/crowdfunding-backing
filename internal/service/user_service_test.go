package service

import (
	"crowdfunding-backend/internal/model"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserRepository 是 UserRepository 接口的模拟实现
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(user *model.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) FindByID(id int) (*model.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) FindByEmail(email string) (*model.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) FindByUsername(username string) (*model.User, error) {
	args := m.Called(username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) Update(user *model.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserRepository) Count() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

func (m *MockUserRepository) FindAll(page, pageSize int) ([]*model.User, error) {
	args := m.Called(page, pageSize)
	return args.Get(0).([]*model.User), args.Error(1)
}

func (m *MockUserRepository) CreateAddress(address *model.UserAddress) error {
	args := m.Called(address)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateAddress(address *model.UserAddress) error {
	args := m.Called(address)
	return args.Error(0)
}

func (m *MockUserRepository) DeleteAddress(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserRepository) GetAddressByID(id int) (*model.UserAddress, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.UserAddress), args.Error(1)
}

func (m *MockUserRepository) ListUserAddresses(userID int) ([]*model.UserAddress, error) {
	args := m.Called(userID)
	return args.Get(0).([]*model.UserAddress), args.Error(1)
}

func (m *MockUserRepository) SetDefaultAddress(userID, addressID int) error {
	args := m.Called(userID, addressID)
	return args.Error(0)
}

// TestRegister 测试用户注册功能
func TestRegister(t *testing.T) {
	mockRepo := new(MockUserRepository)
	service := NewUserService(mockRepo)

	user := &model.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "password123",
	}

	// 测试成功注册
	mockRepo.On("FindByUsername", "testuser").Return(nil, nil)
	mockRepo.On("Create", mock.AnythingOfType("*model.User")).Return(nil)

	err := service.Register(user)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)

	// 测试用户名已存在
	mockRepo.On("FindByUsername", "existinguser").Return(&model.User{}, nil)
	user.Username = "existinguser"
	err = service.Register(user)
	assert.Error(t, err)
	assert.Equal(t, "username already exists", err.Error())
}



// TestUpdateProfile 测试更新用户资料功能
func TestUpdateProfile(t *testing.T) {
	mockRepo := new(MockUserRepository)
	service := NewUserService(mockRepo)

	user := &model.User{
		ID:       1,
		Username: "updateduser",
		Bio:      "Updated bio",
	}

	// 测试成功更新
	mockRepo.On("FindByID", 1).Return(user, nil)
	mockRepo.On("Update", mock.AnythingOfType("*model.User")).Return(nil)

	err := service.UpdateUser(user)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)

	// 测试用户不存在
	mockRepo.On("FindByID", 999).Return(nil, nil)
	user.ID = 999
	err = service.UpdateUser(user)
	assert.Error(t, err)
}

// TestCreateAddress 测试创建地址功能
func TestCreateAddress(t *testing.T) {
	mockRepo := new(MockUserRepository)
	service := NewUserService(mockRepo)

	address := &model.UserAddress{
		UserID:        1,
		ReceiverName:  "Test User",
		Phone:         "1234567890",
		Province:      "Test Province",
		City:          "Test City",
		District:      "Test District",
		DetailAddress: "Test Address",
		IsDefault:     true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// 测试成功创建地址
	mockRepo.On("CreateAddress", address).Return(nil)
	err := service.CreateAddress(address)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// 可以继续添加更多测试用例...
