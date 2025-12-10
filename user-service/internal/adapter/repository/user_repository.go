package repository

import (
	"context"
	"errors"
	"fmt"
	"time"
	"user-service/internal/core/domain/entity"
	"user-service/internal/core/domain/model"
	"user-service/utils/message"

	"github.com/labstack/gommon/log"
	"gorm.io/gorm"
)

type UserRepositoryInterface interface {
	GetUserByEmail(ctx context.Context, email string) (*entity.UserEntity, error)
	CreateUserAccount(ctx context.Context, user entity.UserEntity) error
	UpdateUserVerified(ctx context.Context, userID int64) (*entity.UserEntity, error)
	UpdatePasswordByID(ctx context.Context, user entity.UserEntity) error
	GetUserByID(ctx context.Context, userID int64) (*entity.UserEntity, error)
	UpdateDataUser(ctx context.Context, user entity.UserEntity) error

	// Customer
	GetAllCustomers(ctx context.Context, query entity.QueryStringCustomer) ([]entity.UserEntity, int64, int64, error)
	GetCustomerByID(ctx context.Context, userID int64) (*entity.UserEntity, error)
	CreateCustomer(ctx context.Context, user entity.UserEntity) error
	UpdateCustomer(ctx context.Context, user entity.UserEntity) error
	DeleteCustomer(ctx context.Context, customerID int64) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepositoryInterface {
	return &userRepository{db: db}
}

func (u *userRepository) GetUserByEmail(ctx context.Context, email string) (*entity.UserEntity, error) {
	modelUser := model.User{}

	if err := u.db.WithContext(ctx).Where("email = ?", email).Where("is_verified = ?", true).Preload("Roles").First(&modelUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Errorf("[UserRepository-1] GetUserByEmail: User not found")
			return nil, message.ErrUserNotFound
		}
		log.Errorf("[UserRepository-1] GetUserByEmail: %v", err)
		return nil, err
	}

	return &entity.UserEntity{
		ID:         modelUser.ID,
		Name:       modelUser.Name,
		Email:      email,
		Password:   modelUser.Password,
		RoleName:   modelUser.Roles[0].Name,
		Address:    modelUser.Address,
		Lat:        modelUser.Lat,
		Lng:        modelUser.Lng,
		Phone:      modelUser.Phone,
		Photo:      modelUser.Photo,
		IsVerified: modelUser.IsVerified,
	}, nil

}

func (u *userRepository) CreateUserAccount(ctx context.Context, user entity.UserEntity) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		modelRole := model.Role{}
		err := u.db.WithContext(ctx).Where("name = ?", "Customer").First(&modelRole).Error
		if err != nil {
			log.Errorf("[UserRepository-1] CreateUserAccount: %v", err)
			return err
		}

		modelUser := model.User{
			Name:     user.Name,
			Email:    user.Email,
			Password: user.Password,
			Roles:    []model.Role{modelRole},
		}

		if err = u.db.WithContext(ctx).Create(&modelUser).Error; err != nil {
			log.Errorf("[UserRepository-2] CreateUserAccount: %v", err)
			return err
		}

		modelVerify := model.VerificationToken{
			UserID:    modelUser.ID,
			Token:     user.Token,
			TokenType: "email_verification",
			ExpiresAt: time.Now().Add(time.Hour * 24),
		}

		if err = u.db.WithContext(ctx).Create(&modelVerify).Error; err != nil {
			log.Errorf("[UserRepository-3] CreateUserAccount: %v", err)
			return err
		}

		return nil
	})
}

func (u *userRepository) UpdateUserVerified(ctx context.Context, userID int64) (*entity.UserEntity, error) {
	tx := u.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Update("is_verified", true)

	if tx.Error != nil {
		log.Errorf("[UserRepository-1] UpdateUserVerified: %v", tx.Error)
		return nil, tx.Error
	}

	if tx.RowsAffected == 0 {
		return nil, message.ErrUserNotFound
	}

	modelUser := model.User{}
	err := u.db.WithContext(ctx).Where("id = ?", userID).Preload("Roles").First(&modelUser).Error
	if err != nil {
		log.Errorf("[UserRepository-2] UpdateUserVerified: %v", err)
		return nil, err
	}

	result := &entity.UserEntity{
		ID:         modelUser.ID,
		Name:       modelUser.Name,
		Email:      modelUser.Email,
		IsVerified: modelUser.IsVerified,
		RoleName:   modelUser.Roles[0].Name,
		Photo:      modelUser.Photo,
		Address:    modelUser.Address,
		Lat:        modelUser.Lat,
		Lng:        modelUser.Lng,
		Phone:      modelUser.Phone,
	}

	return result, nil
}

func (u *userRepository) UpdatePasswordByID(ctx context.Context, user entity.UserEntity) error {
	modelUser := model.User{}

	if err := u.db.WithContext(ctx).Where("id = ?", user.ID).First(&modelUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Errorf("[UserRepository-1] UpdatePasswordById: User not found")
			return message.ErrUserNotFound
		}
		log.Errorf("[UserRepository-2] UpdatePasswordById: %v", err)
		return err
	}

	modelUser.Password = user.Password
	if err := u.db.WithContext(ctx).Save(&modelUser).Error; err != nil {
		log.Errorf("[UserRepository-3] UpdatePasswordById: %v", err)
		return err
	}

	return nil
}

func (u *userRepository) GetUserByID(ctx context.Context, userID int64) (*entity.UserEntity, error) {
	modelUser := model.User{}

	tx := u.db.WithContext(ctx).Where("id = ?", userID).Where("is_verified = ?", true).Preload("Roles").First(&modelUser)

	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			log.Errorf("[UserRepository-1] GetUserByID: User not found")
			return nil, message.ErrUserNotFound
		}
		log.Errorf("[UserRepository-2] GetUserByID: %v", tx.Error)
		return nil, tx.Error
	}

	return &entity.UserEntity{
		ID:         modelUser.ID,
		Name:       modelUser.Name,
		Email:      modelUser.Email,
		Photo:      modelUser.Photo,
		Address:    modelUser.Address,
		Lat:        modelUser.Lat,
		Lng:        modelUser.Lng,
		Phone:      modelUser.Phone,
		RoleName:   modelUser.Roles[0].Name,
		IsVerified: modelUser.IsVerified,
	}, nil
}

func (u *userRepository) UpdateDataUser(ctx context.Context, user entity.UserEntity) error {
	modelUser := model.User{
		Name:    user.Name,
		Email:   user.Email,
		Photo:   user.Photo,
		Address: user.Address,
		Lat:     user.Lat,
		Lng:     user.Lng,
		Phone:   user.Phone,
	}

	tx := u.db.WithContext(ctx).Model(&modelUser).Where("id = ?", user.ID).Where("is_verified = ?", true).Updates(&modelUser)

	if tx.Error != nil {
		log.Errorf("[UserRepository-1] UpdateDataUser: %v", tx.Error)
		return tx.Error
	}

	if tx.RowsAffected == 0 {
		log.Error("[UserRepository-2] UpdateDataUser: User not found")
		return message.ErrUserNotFound
	}

	return nil
}

func (u *userRepository) GetAllCustomers(ctx context.Context, query entity.QueryStringCustomer) ([]entity.UserEntity, int64, int64, error) {
	modelUser := []model.User{}
	var count int64

	allowedSort := map[string]bool{"name": true, "email": true, "phone": true}
	orderBy := "created_at"
	if allowedSort[query.OrderBy] {
		orderBy = query.OrderBy
	}

	orderClause := fmt.Sprintf("%s %s", orderBy, query.OrderType)

	search := "%" + query.Search + "%"
	limit := int(query.Limit)
	offset := int((query.Page - 1) * query.Limit)

	q := u.db.WithContext(ctx).
		Model(&model.User{}).
		Joins("JOIN user_role ON user_role.user_id = users.id").
		Joins("JOIN roles ON roles.id = user_role.role_id").
		Where("roles.name = ?", "Customer").
		Where("users.name ILIKE ? OR users.email ILIKE ? OR users.phone ILIKE ?", search, search, search)

	if err := q.Model(&modelUser).Count(&count).Error; err != nil {
		log.Errorf("[UserRepository-1] GetAllCustomer: %v", err)
		return nil, 0, 0, err
	}

	total := (count + query.Limit - 1) / query.Limit

	if err := q.Preload("Roles").
		Order(orderClause).
		Limit(limit).
		Offset(offset).
		Find(&modelUser).Error; err != nil {
		log.Errorf("[UserRepository-2] GetAllCustomer: %v", err)
		return nil, 0, 0, err
	}

	result := make([]entity.UserEntity, 0, len(modelUser))

	for _, user := range modelUser {

		result = append(result, entity.UserEntity{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
			Photo: user.Photo,
			Phone: user.Phone,
		})
	}

	return result, count, total, nil
}

func (u *userRepository) GetCustomerByID(ctx context.Context, userID int64) (*entity.UserEntity, error) {
	modelUser := model.User{}
	tx := u.db.WithContext(ctx).Where("id = ?", userID).Preload("Roles").First(&modelUser)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			log.Errorf("[UserRepository-1] GetCustomerByID: User not found")
			return nil, message.ErrUserNotFound
		}
		log.Errorf("[UserRepository-2] GetCustomerByID: %v", tx.Error)
		return nil, tx.Error
	}

	if len(modelUser.Roles) == 0 || modelUser.Roles[0].Name != "Customer" {
		log.Errorf("[UserRepository-3] GetCustomerByID: User not found, user role is not Customer")
		return nil, message.ErrUserNotFound
	}

	return &entity.UserEntity{
		ID:       modelUser.ID,
		Name:     modelUser.Name,
		Email:    modelUser.Email,
		Photo:    modelUser.Photo,
		Phone:    modelUser.Phone,
		RoleName: modelUser.Roles[0].Name,
		RoleID:   modelUser.Roles[0].ID,
		Address:  modelUser.Address,
		Lat:      modelUser.Lat,
		Lng:      modelUser.Lng,
	}, nil

}

func (u *userRepository) CreateCustomer(ctx context.Context, user entity.UserEntity) error {
	modelRole := model.Role{}
	err := u.db.WithContext(ctx).Where("name = ?", "Customer").First(&modelRole).Error
	if err != nil {
		log.Errorf("[UserRepository-1] CreateCustomer: %v", err)
		return err
	}

	modelUser := model.User{
		Name:       user.Name,
		Email:      user.Email,
		Password:   user.Password,
		Roles:      []model.Role{modelRole},
		Address:    user.Address,
		Lat:        user.Lat,
		Lng:        user.Lng,
		Phone:      user.Phone,
		Photo:      user.Photo,
		IsVerified: true,
	}

	if err = u.db.WithContext(ctx).Create(&modelUser).Error; err != nil {
		log.Errorf("[UserRepository-2] CreateCustomer: %v", err)
		return err
	}

	return nil
}

func (u *userRepository) UpdateCustomer(ctx context.Context, user entity.UserEntity) error {

	var count int64
	err := u.db.WithContext(ctx).Table("users").
		Joins("JOIN user_role ON user_role.user_id = users.id ").
		Joins("JOIN roles ON user_role.role_id = roles.id").
		Where("users.id = ? AND roles.name = ?", user.ID, "Customer").
		Count(&count).Error

	if err != nil {
		log.Errorf("[UserRepository - 1] UpdateCustomer: %v", err)
		return err
	}

	if count == 0 {
		log.Errorf("[UserRepository - 2] UpdateCustomer: User not found")
		return message.ErrCustomerNotFound
	}

	modelUser := model.User{
		Name:     user.Name,
		Email:    user.Email,
		Password: user.Password,
		Address:  user.Address,
		Phone:    user.Phone,
		Photo:    user.Photo,
		Lat:      user.Lat,
		Lng:      user.Lng,
	}

	updateColumns := []string{"Name", "Email", "Phone", "Address", "Photo", "Lat", "Lng"}

	if modelUser.Password != "" {
		updateColumns = append(updateColumns, "Password")
	}

	err = u.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", user.ID).
		Select(updateColumns).
		Updates(&modelUser).Error

	return err
}

func (u *userRepository) DeleteCustomer(ctx context.Context, customerID int64) error {

	var count int64
	err := u.db.WithContext(ctx).Table("users").
		Joins("JOIN user_role ON user_role.user_id = users.id").
		Joins("JOIN roles ON roles.id = user_role.role_id ").
		Where("users.id = ? AND roles.name = ?", customerID, "Customer").Count(&count).Error

	if err != nil {
		log.Errorf("[UserRepository - 1] DeleteCustomer: %v", err)
		return err
	}

	if count == 0 {
		log.Errorf("[UserRepository - 2] DeleteCustomer: %v", err)
		return message.ErrCustomerNotFound
	}

	tx := u.db.WithContext(ctx).Where("id = ?", customerID).Delete(&model.User{})

	if tx.Error != nil {
		log.Errorf("[UserRepository - 3] DeleteCustomer: %v", err)
		return tx.Error
	}

	if tx.RowsAffected == 0 {
		log.Errorf("[UserRepository - 4] DeleteCustomer: %v", err)
		return message.ErrCustomerNotFound
	}

	return nil
}
