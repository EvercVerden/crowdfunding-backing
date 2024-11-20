package mysql

import (
	"crowdfunding-backend/internal/model"
	"crowdfunding-backend/internal/util"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"go.uber.org/zap"
)

// userRepository 实现了 UserRepository 接口
type userRepository struct {
	db *sql.DB
}

// NewUserRepository 创建一个新的 userRepository 实例
func NewUserRepository(db *sql.DB) *userRepository {
	return &userRepository{db}
}

// Create 创建一个新用户
func (r *userRepository) Create(user *model.User) error {
	log.Printf("尝试创建新用户：%s", user.Email)
	query := `INSERT INTO users (username, email, password_hash, avatar_url, bio, is_verified) 
              VALUES (?, ?, ?, ?, ?, ?)`
	result, err := r.db.Exec(query, user.Username, user.Email, user.PasswordHash, user.AvatarURL, user.Bio, user.IsVerified)
	if err != nil {
		log.Printf("创建用户失败：%v", err)
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		log.Printf("获取新用户ID失败：%v", err)
		return err
	}
	user.ID = int(id)
	user.Role = "user" // 设置默认角色
	log.Printf("用户创建成功：ID=%d", user.ID)
	return nil
}

// FindByID 通过ID查找用户
func (r *userRepository) FindByID(id int) (*model.User, error) {
	log.Printf("尝试通过ID查找用户：%d", id)
	query := `SELECT id, username, email, password_hash, avatar_url, bio, role, is_verified, created_at, updated_at 
              FROM users WHERE id = ?`
	var user model.User
	err := r.db.QueryRow(query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.AvatarURL, &user.Bio,
		&user.Role, &user.IsVerified, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		log.Printf("查找用户失败：%v", err)
		return nil, err
	}
	log.Printf("用户查找成功：ID=%d", user.ID)
	return &user, nil
}

// FindByEmail 通过邮箱查找用户
func (r *userRepository) FindByEmail(email string) (*model.User, error) {
	log.Printf("尝试通过邮箱查找用户：%s", email)
	query := `SELECT id, username, email, password_hash, avatar_url, bio, role, is_verified, created_at, updated_at 
              FROM users WHERE email = ?`
	var user model.User
	err := r.db.QueryRow(query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.AvatarURL, &user.Bio,
		&user.Role, &user.IsVerified, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		log.Printf("查找用户失败：%v", err)
		return nil, err
	}
	log.Printf("用户查找成功：ID=%d", user.ID)
	return &user, nil
}

// Update 更新用户信息
func (r *userRepository) Update(user *model.User) error {
	_, err := r.db.Exec(`
		UPDATE users 
		SET username = ?, email = ?, avatar_url = ?, bio = ?, updated_at = ?
		WHERE id = ?`,
		user.Username, user.Email, user.AvatarURL, user.Bio, time.Now(), user.ID)
	return err
}

// Delete 删除用户
func (r *userRepository) Delete(id int) error {
	log.Printf("尝试删除用户：ID=%d", id)
	query := `DELETE FROM users WHERE id = ?`
	_, err := r.db.Exec(query, id)
	if err != nil {
		log.Printf("删除用户失败：%v", err)
		return err
	}
	log.Printf("用户删除成功：ID=%d", id)
	return nil
}

// FindByUsername 通过用户名查找用户
func (r *userRepository) FindByUsername(username string) (*model.User, error) {
	query := `SELECT id, username, email, password_hash, avatar_url, bio, role, is_verified, created_at, updated_at 
              FROM users WHERE username = ?`
	var user model.User
	err := r.db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.AvatarURL, &user.Bio,
		&user.Role, &user.IsVerified, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// Count 返回用户总数
func (r *userRepository) Count() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// FindAll 返回分页的用户列表
func (r *userRepository) FindAll(page, pageSize int) ([]*model.User, error) {
	offset := (page - 1) * pageSize
	query := `SELECT id, username, email, password_hash, avatar_url, bio, role, is_verified, created_at, updated_at 
              FROM users LIMIT ? OFFSET ?`
	rows, err := r.db.Query(query, pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		var user model.User
		err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.AvatarURL, &user.Bio,
			&user.Role, &user.IsVerified, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	return users, nil
}

// CreateAddress 创建一个新地址
func (r *userRepository) CreateAddress(address *model.UserAddress) error {
	util.Logger.Info("Repository: 开始创建地址",
		zap.Int("user_id", address.UserID))

	// 检查用户是否存在
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)",
		address.UserID).Scan(&exists)
	if err != nil {
		util.Logger.Error("检查用户存在性失败",
			zap.Error(err),
			zap.Int("user_id", address.UserID))
		return fmt.Errorf("failed to check user existence: %w", err)
	}
	if !exists {
		util.Logger.Error("用户不存在",
			zap.Int("user_id", address.UserID))
		return errors.New("user not found")
	}

	query := `INSERT INTO user_addresses 
              (user_id, receiver_name, phone, province, city, district, detail_address, is_default) 
              VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	// 打印完整的 SQL 和参数
	util.Logger.Debug("准备执行SQL",
		zap.String("query", query),
		zap.Any("params", []interface{}{
			address.UserID, address.ReceiverName, address.Phone,
			address.Province, address.City, address.District,
			address.DetailAddress, address.IsDefault,
		}))

	result, err := r.db.Exec(query,
		address.UserID, address.ReceiverName, address.Phone,
		address.Province, address.City, address.District,
		address.DetailAddress, address.IsDefault)

	if err != nil {
		util.Logger.Error("执行SQL失败",
			zap.Error(err),
			zap.String("query", query),
			zap.String("error_type", fmt.Sprintf("%T", err)))
		return fmt.Errorf("failed to execute SQL: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		util.Logger.Error("获取新地址ID失败",
			zap.Error(err),
			zap.String("error_type", fmt.Sprintf("%T", err)))
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	address.ID = int(id)
	util.Logger.Info("地址创建成功",
		zap.Int("address_id", address.ID),
		zap.Int("user_id", address.UserID))
	return nil
}

// UpdateAddress 更新地址信息
func (r *userRepository) UpdateAddress(address *model.UserAddress) error {
	query := `UPDATE user_addresses 
              SET receiver_name = ?, phone = ?, province = ?, city = ?, 
                  district = ?, detail_address = ?, is_default = ?
              WHERE id = ? AND user_id = ?`
	_, err := r.db.Exec(query,
		address.ReceiverName, address.Phone,
		address.Province, address.City, address.District,
		address.DetailAddress, address.IsDefault,
		address.ID, address.UserID)
	return err
}

// DeleteAddress 删除地址
func (r *userRepository) DeleteAddress(id int) error {
	query := `DELETE FROM user_addresses WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

// GetAddressByID 通过ID查找地址
func (r *userRepository) GetAddressByID(id int) (*model.UserAddress, error) {
	var address model.UserAddress
	query := `SELECT id, user_id, receiver_name, phone, province, city, district, 
                     detail_address, is_default, created_at, updated_at 
              FROM user_addresses WHERE id = ?`
	err := r.db.QueryRow(query, id).Scan(
		&address.ID, &address.UserID, &address.ReceiverName,
		&address.Phone, &address.Province, &address.City,
		&address.District, &address.DetailAddress, &address.IsDefault,
		&address.CreatedAt, &address.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &address, nil
}

// ListUserAddresses 返回用户的地址列表
func (r *userRepository) ListUserAddresses(userID int) ([]*model.UserAddress, error) {
	util.Logger.Info("开始获取用户地址列表", zap.Int("user_id", userID))

	query := `SELECT id, user_id, receiver_name, phone, province, city, district, 
                     detail_address, is_default, created_at, updated_at 
              FROM user_addresses 
              WHERE user_id = ? 
              ORDER BY is_default DESC, created_at DESC`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		util.Logger.Error("查询用户地址失败",
			zap.Error(err),
			zap.Int("user_id", userID))
		return nil, fmt.Errorf("failed to query addresses: %w", err)
	}
	defer rows.Close()

	var addresses []*model.UserAddress
	for rows.Next() {
		var address model.UserAddress
		err := rows.Scan(
			&address.ID, &address.UserID, &address.ReceiverName,
			&address.Phone, &address.Province, &address.City,
			&address.District, &address.DetailAddress, &address.IsDefault,
			&address.CreatedAt, &address.UpdatedAt)
		if err != nil {
			util.Logger.Error("扫描地址数据失败",
				zap.Error(err),
				zap.Int("user_id", userID))
			return nil, fmt.Errorf("failed to scan address: %w", err)
		}
		addresses = append(addresses, &address)
	}

	if err = rows.Err(); err != nil {
		util.Logger.Error("遍历地址数据失败",
			zap.Error(err),
			zap.Int("user_id", userID))
		return nil, fmt.Errorf("failed to iterate addresses: %w", err)
	}

	util.Logger.Info("成功获取用户地址列表",
		zap.Int("user_id", userID),
		zap.Int("count", len(addresses)))
	return addresses, nil
}

// SetDefaultAddress 设置默认地址
func (r *userRepository) SetDefaultAddress(userID, addressID int) error {
	util.Logger.Info("开始设置默认地址",
		zap.Int("user_id", userID),
		zap.Int("address_id", addressID))

	tx, err := r.db.Begin()
	if err != nil {
		util.Logger.Error("开始事务失败", zap.Error(err))
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 先取消所有默认地址
	_, err = tx.Exec(`UPDATE user_addresses SET is_default = false WHERE user_id = ?`, userID)
	if err != nil {
		util.Logger.Error("取消默认地址失败",
			zap.Error(err),
			zap.Int("user_id", userID))
		return fmt.Errorf("failed to unset default addresses: %w", err)
	}

	// 设置新的默认地址
	result, err := tx.Exec(`UPDATE user_addresses SET is_default = true WHERE id = ? AND user_id = ?`,
		addressID, userID)
	if err != nil {
		util.Logger.Error("设置默认地址失败",
			zap.Error(err),
			zap.Int("address_id", addressID))
		return fmt.Errorf("failed to set default address: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		util.Logger.Error("获取影响行数失败", zap.Error(err))
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if affected == 0 {
		util.Logger.Error("地址不存在或不属于该用户",
			zap.Int("user_id", userID),
			zap.Int("address_id", addressID))
		return errors.New("address not found or does not belong to user")
	}

	if err := tx.Commit(); err != nil {
		util.Logger.Error("提交事务失败", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	util.Logger.Info("成功设置默认地址",
		zap.Int("user_id", userID),
		zap.Int("address_id", addressID))
	return nil
}
