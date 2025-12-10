package service

import (
	"context"
	"user-service/internal/adapter/repository"
	"user-service/internal/core/domain/entity"
)

type RoleServiceInterface interface {
	GetAllRole(ctx context.Context, search string) ([]entity.RoleEntity, error)
	GetRoleByID(ctx context.Context, id int64) (*entity.RoleEntity, error)
	CreateRole(ctx context.Context, role entity.RoleEntity) error
	UpdateRole(ctx context.Context, role entity.RoleEntity) error
	DeleteRoleByID(ctx context.Context, id int64) error
}

type roleService struct {
	repo repository.RoleRepositoryInterface
}

func NewRoleService(repo repository.RoleRepositoryInterface) RoleServiceInterface {
	return &roleService{repo: repo}
}

func (r *roleService) GetAllRole(ctx context.Context, search string) ([]entity.RoleEntity, error) {
	return r.repo.GetAllRole(ctx, search)
}

func (r *roleService) GetRoleByID(ctx context.Context, id int64) (*entity.RoleEntity, error) {
	return r.repo.GetRoleByID(ctx, id)
}

func (r *roleService) CreateRole(ctx context.Context, role entity.RoleEntity) error {
	return r.repo.CreateRole(ctx, role)
}

func (r *roleService) UpdateRole(ctx context.Context, role entity.RoleEntity) error {
	return r.repo.UpdateRole(ctx, role)
}

func (r *roleService) DeleteRoleByID(ctx context.Context, id int64) error {
	return r.repo.DeleteRoleByID(ctx, id)
}
