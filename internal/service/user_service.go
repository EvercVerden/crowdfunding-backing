package service

import (
	"crowdfunding-backend/internal/errors"
	"crowdfunding-backend/internal/model"
	"crowdfunding-backend/internal/repository/interfaces"
	"crowdfunding-backend/internal/util"
	"database/sql"
	stderrors "errors"
	"fmt"
	"log"
	"sync"
	"time"

	"go.uber.org/zap"

	"golang.org/x/crypto/bcrypt"
)

// UserService 处理与用户相关的业务逻辑
type UserService struct {
	userRepo interfaces.UserRepository
	// 添加邮件服务
	emailService   *EmailService
	tokenBlacklist map[string]time.Time
	blacklistMutex sync.RWMutex
}

// NewUserService 创建一个新的 UserService 实例
func NewUserService(userRepo interfaces.UserRepository) *UserService {
	return &UserService{
		userRepo:       userRepo,
		emailService:   NewEmailService(userRepo),
		tokenBlacklist: make(map[string]time.Time),
	}
}

// IsUsernameTaken 检查用户名是否已被使用
func (s *UserService) IsUsernameTaken(username string) (bool, error) {
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return user != nil, nil
}

// Register 注册新用户
func (s *UserService) Register(user *model.User) error {
	// 检查用户名是否已被使用
	taken, err := s.IsUsernameTaken(user.Username)
	if err != nil {
		return err
	}
	if taken {
		return errors.New(errors.ErrUserExists, "username already exists")
	}

	// 生成密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.PasswordHash), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hashedPassword)

	// 创建用户
	err = s.userRepo.Create(user)
	if err != nil {
		return err
	}

	// 发送验证邮件
	err = s.emailService.SendVerificationEmail(user.Email, user.Username)
	if err != nil {
		util.Logger.Error("发送验证邮件失败", zap.Error(err))
		// 考虑是否需要回滚用户创建
	}

	return nil
}

// Login 用户登录
func (s *UserService) Login(email, password string) (*model.User, error) {
	log.Printf("尝试用户登录：%s", email)

	// 查找用户
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		log.Printf("用户登录失败，未找到用户：%v", err)
		return nil, err
	}

	// 验证密码
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		log.Printf("用户登录失败，密码不正确：%v", err)
		return nil, err
	}

	log.Printf("用户登录成功：ID=%d", user.ID)
	return user, nil
}

// GetUserByID 通过ID获取用户信息
func (s *UserService) GetUserByID(id int) (*model.User, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	if user == nil {
		return nil, stderrors.New("用户不存在")
	}
	return user, nil
}

// UpdateUser 更新用户信息
func (s *UserService) UpdateUser(user *model.User) error {
	existingUser, err := s.userRepo.FindByID(user.ID)
	if err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}
	if existingUser == nil {
		return stderrors.New("用户不存在")
	}

	// 只更新允许修改的字段
	existingUser.Username = user.Username
	existingUser.Bio = user.Bio

	if err := s.userRepo.Update(existingUser); err != nil {
		return fmt.Errorf("更新用户失败: %w", err)
	}
	return nil
}

// 添加验证邮箱的方法
func (s *UserService) VerifyEmail(token string) error {
	userID, err := s.emailService.VerifyEmailToken(token)
	if err != nil {
		util.Logger.Error("验证邮箱令牌失败", zap.Error(err))
		return err
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		util.Logger.Error("查找用户失败", zap.Error(err), zap.Int("user_id", userID))
		return err
	}
	if user == nil {
		return errors.New(errors.ErrUserNotFound, "user not found")
	}

	if user.IsVerified {
		return errors.New(errors.ErrResourceExists, "email already verified")
	}

	user.IsVerified = true
	if err := s.userRepo.Update(user); err != nil {
		util.Logger.Error("更新用户验证状态失败", zap.Error(err), zap.Int("user_id", user.ID))
		return err
	}

	util.Logger.Info("邮箱验证成功", zap.Int("user_id", user.ID))
	return nil
}

func (s *UserService) RequestPasswordReset(email string) error {
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New(errors.ErrUserNotFound, "user not found")
	}
	return s.emailService.SendPasswordResetEmail(email)
}

func (s *UserService) ResetPassword(token, newPassword string) error {
	email, err := s.emailService.VerifyPasswordResetToken(token)
	if err != nil {
		util.Logger.Error("验证密码重置令牌失败", zap.Error(err))
		return err
	}

	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		util.Logger.Error("查找用户失败", zap.Error(err), zap.String("email", email))
		return err
	}
	if user == nil {
		return errors.New(errors.ErrUserNotFound, "user not found")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		util.Logger.Error("生成密码哈希失败", zap.Error(err))
		return err
	}

	user.PasswordHash = string(hashedPassword)
	if err := s.userRepo.Update(user); err != nil {
		util.Logger.Error("更用户密码失败", zap.Error(err), zap.Int("user_id", user.ID))
		return err
	}

	util.Logger.Info("密码重置成功", zap.Int("user_id", user.ID))
	return nil
}

func (s *UserService) Logout(userID int) error {
	token, err := util.GenerateToken(userID)
	if err != nil {
		return err
	}
	s.blacklistMutex.Lock()
	s.tokenBlacklist[token] = time.Now().Add(24 * time.Hour) // 令牌在黑名单中保留24小时
	s.blacklistMutex.Unlock()
	util.Logger.Info("用户注销，令牌已加入黑名单", zap.Int("user_id", userID))
	return nil
}

func (s *UserService) IsTokenBlacklisted(token string) bool {
	s.blacklistMutex.RLock()
	defer s.blacklistMutex.RUnlock()
	expiry, exists := s.tokenBlacklist[token]
	if !exists {
		return false
	}
	if time.Now().After(expiry) {
		delete(s.tokenBlacklist, token)
		return false
	}
	return true
}

type UserServiceInterface interface {
	Register(user *model.User) error
	Login(email, password string) (*model.User, error)
	GetUserByID(id int) (*model.User, error)
	UpdateUser(user *model.User) error
	VerifyEmail(token string) error
	RequestPasswordReset(email string) error
	ResetPassword(token, newPassword string) error
	Logout(userID int) error
	IsTokenBlacklisted(token string) bool
}

// 确保 UserService 实现了 UserServiceInterface
var _ UserServiceInterface = (*UserService)(nil)

func (s *UserService) IsAdmin(userID int) (bool, error) {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return false, err
	}
	return user.Role == "admin", nil
}

func (s *UserService) GetUsers(page, pageSize int) ([]*model.User, error) {
	// 实现获取用户列表的逻辑
	return s.userRepo.FindAll(page, pageSize)
}

func (s *UserService) UpdateUserRole(userID int, newRole string) error {
	// 实现更新用户角色的逻辑
	user, err := s.GetUserByID(userID)
	if err != nil {
		return err
	}
	user.Role = newRole
	return s.userRepo.Update(user)
}

// UpdateAvatar 更新用户头像
func (s *UserService) UpdateAvatar(userID int, avatarURL string) error {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return err
	}

	user.AvatarURL = avatarURL
	return s.userRepo.Update(user)
}

// DeleteAccount 注销用户账户
func (s *UserService) DeleteAccount(userID int) error {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return err
	}

	// 软删除：设置 DeletedAt 字段
	now := time.Now()
	user.DeletedAt = &now

	return s.userRepo.Update(user)
}

// 在 UserService 结构体中添加地址相关的方法
func (s *UserService) CreateAddress(address *model.UserAddress) error {
	util.Logger.Info("Service: 开始创建地址",
		zap.Int("user_id", address.UserID),
		zap.Any("address", address))

	// 验证用户是否存在
	user, err := s.GetUserByID(address.UserID)
	if err != nil {
		util.Logger.Error("验证用户失败",
			zap.Error(err),
			zap.Int("user_id", address.UserID))
		return fmt.Errorf("failed to validate user: %w", err)
	}
	if user == nil {
		util.Logger.Error("用户不存在",
			zap.Int("user_id", address.UserID))
		return errors.New(errors.ErrUserNotFound, "user not found")
	}

	// 验证地址数据
	if err := validateAddress(address); err != nil {
		util.Logger.Error("地址数据验证失败",
			zap.Error(err),
			zap.Any("address", address))
		return fmt.Errorf("address validation failed: %w", err)
	}

	// 如果是默认地址，先取消其他默认地址
	if address.IsDefault {
		if err := s.handleDefaultAddress(address.UserID); err != nil {
			util.Logger.Error("处理默认地址失败",
				zap.Error(err),
				zap.Int("user_id", address.UserID))
			return fmt.Errorf("failed to handle default address: %w", err)
		}
	}

	if err := s.userRepo.CreateAddress(address); err != nil {
		util.Logger.Error("数据库创建地址失败",
			zap.Error(err),
			zap.Any("address", address))
		return fmt.Errorf("failed to create address in database: %w", err)
	}

	util.Logger.Info("地址创建成功",
		zap.Int("address_id", address.ID),
		zap.Int("user_id", address.UserID))
	return nil
}

func validateAddress(address *model.UserAddress) error {
	if address.ReceiverName == "" {
		return errors.New(errors.ErrValidation, "receiver name is required")
	}
	if address.Phone == "" {
		return errors.New(errors.ErrValidation, "phone is required")
	}
	if address.Province == "" || address.City == "" || address.District == "" {
		return errors.New(errors.ErrValidation, "incomplete address")
	}
	if address.DetailAddress == "" {
		return errors.New(errors.ErrValidation, "detail address is required")
	}
	return nil
}

func (s *UserService) handleDefaultAddress(userID int) error {
	addresses, err := s.userRepo.ListUserAddresses(userID)
	if err != nil {
		return err
	}

	for _, addr := range addresses {
		if addr.IsDefault {
			addr.IsDefault = false
			if err := s.userRepo.UpdateAddress(addr); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *UserService) UpdateAddress(address *model.UserAddress) error {
	return s.userRepo.UpdateAddress(address)
}

func (s *UserService) DeleteAddress(id int) error {
	return s.userRepo.DeleteAddress(id)
}

func (s *UserService) GetAddressByID(id int) (*model.UserAddress, error) {
	return s.userRepo.GetAddressByID(id)
}

func (s *UserService) ListUserAddresses(userID int) ([]*model.UserAddress, error) {
	return s.userRepo.ListUserAddresses(userID)
}

func (s *UserService) SetDefaultAddress(userID, addressID int) error {
	return s.userRepo.SetDefaultAddress(userID, addressID)
}
