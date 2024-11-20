package interfaces

import "crowdfunding-backend/internal/model"

// UserRepository 接口定义了用户仓库应该实现的方法
type UserRepository interface {
	Create(user *model.User) error
	FindByID(id int) (*model.User, error)
	FindByEmail(email string) (*model.User, error)
	FindByUsername(username string) (*model.User, error)
	Update(user *model.User) error
	Delete(id int) error
	Count() (int, error)
	FindAll(page, pageSize int) ([]*model.User, error)
	CreateAddress(address *model.UserAddress) error
	UpdateAddress(address *model.UserAddress) error
	DeleteAddress(id int) error
	GetAddressByID(id int) (*model.UserAddress, error)
	ListUserAddresses(userID int) ([]*model.UserAddress, error)
	SetDefaultAddress(userID, addressID int) error
}
