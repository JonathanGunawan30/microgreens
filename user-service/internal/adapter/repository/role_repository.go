package repository

import (
	"context"
	"errors"
	"user-service/internal/core/domain/entity"
	"user-service/internal/core/domain/model"
	"user-service/utils/message"

	"github.com/labstack/gommon/log"
	"gorm.io/gorm"
)

type RoleRepositoryInterface interface {
	GetAllRole(ctx context.Context, search string) ([]entity.RoleEntity, error)
	GetRoleByID(ctx context.Context, id int64) (*entity.RoleEntity, error)
	CreateRole(ctx context.Context, role entity.RoleEntity) error
	UpdateRole(ctx context.Context, role entity.RoleEntity) error
	DeleteRoleByID(ctx context.Context, id int64) error
}

type roleRepository struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) RoleRepositoryInterface {
	return &roleRepository{db: db}
}

func (r *roleRepository) GetAllRole(ctx context.Context, search string) ([]entity.RoleEntity, error) {
	var modelRoles []model.Role
	if err := r.db.WithContext(ctx).Where("name ILIKE ?", "%"+search+"%").Find(&modelRoles).Error; err != nil {
		log.Errorf("[RoleRepository-1] GetAllRole: %v", err)
		return nil, err
	}

	var roleEntities []entity.RoleEntity
	for _, modelRole := range modelRoles {
		roleEntities = append(roleEntities, entity.RoleEntity{
			ID:   modelRole.ID,
			Name: modelRole.Name,
		})
	}

	return roleEntities, nil
}

func (r *roleRepository) GetRoleByID(ctx context.Context, id int64) (*entity.RoleEntity, error) {
	modelRole := model.Role{}

	tx := r.db.WithContext(ctx).Where("id = ?", id).First(&modelRole)

	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			log.Errorf("[RoleRepository-1] GetRoleByID: Role not found")
			return nil, message.ErrRoleNotFound
		}
		log.Errorf("[RoleRepository-2] GetRoleByID: %v", tx.Error)
		return nil, tx.Error
	}

	return &entity.RoleEntity{
		ID:   modelRole.ID,
		Name: modelRole.Name,
	}, nil
}

func (r *roleRepository) CreateRole(ctx context.Context, role entity.RoleEntity) error {
	modelRole := model.Role{Name: role.Name}

	tx := r.db.WithContext(ctx).Create(&modelRole)

	if tx.Error != nil {
		log.Errorf("[RoleRepository-1] CreateRole: %v", tx.Error)
		return tx.Error
	}

	return nil
}

func (r *roleRepository) UpdateRole(ctx context.Context, role entity.RoleEntity) error {
	modelRole := model.Role{Name: role.Name}

	tx := r.db.WithContext(ctx).Where("id = ?", role.ID).Updates(&modelRole)

	if tx.Error != nil {
		log.Errorf("[RoleRepository-1] UpdateRole: %v", tx.Error)
		return tx.Error
	}

	if tx.RowsAffected == 0 {
		log.Error("[RoleRepository-2] UpdateRole: Role not found")
		return message.ErrRoleNotFound
	}

	return nil
}

func (r *roleRepository) DeleteRoleByID(ctx context.Context, id int64) error {
	var modelRole model.Role

	if err := r.db.WithContext(ctx).Where("id = ?", id).Preload("Users").First(&modelRole).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Errorf("[RoleRepository-1] DeleteRoleByID: role %d not found", id)
			return message.ErrRoleNotFound
		}
		log.Errorf("[RoleRepository-2] DeleteRoleByID query error: %v", err)
		return err
	}

	if len(modelRole.Users) > 0 {
		log.Errorf("[RoleRepository-3] DeleteRoleByID: role %d still associated with users", id)
		return message.ErrRoleAssociated
	}

	tx := r.db.WithContext(ctx).Where("id = ?", id).Delete(&modelRole)

	if tx.Error != nil {
		log.Errorf("[RoleRepository-4] DeleteRoleByID delete error: %v", tx.Error)
		return tx.Error
	}

	if tx.RowsAffected == 0 {
		log.Errorf("[RoleRepository-5] DeleteRoleByID: role %d not found on delete", id)
		return message.ErrRoleNotFound
	}

	return nil
}
