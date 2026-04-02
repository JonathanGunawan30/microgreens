package service

import (
	"context"
	"errors"
	"testing"
	"user-service/internal/core/domain/entity"
	service2 "user-service/internal/core/service"
	"user-service/internal/mocks"

	"github.com/stretchr/testify/assert"
)

func TestGetAllRole(t *testing.T) {
	mockRepo := new(mocks.RoleRepositoryInterface)
	roleService := service2.NewRoleService(mockRepo)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		expectedRoles := []entity.RoleEntity{
			{ID: 1, Name: "Admin"},
			{ID: 2, Name: "User"},
		}
		mockRepo.On("GetAllRole", ctx, "").Return(expectedRoles, nil).Once()

		roles, err := roleService.GetAllRole(ctx, "")

		assert.NoError(t, err)
		assert.Equal(t, expectedRoles, roles)
		mockRepo.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		mockRepo.On("GetAllRole", ctx, "search").Return(nil, errors.New("db error")).Once()

		roles, err := roleService.GetAllRole(ctx, "search")

		assert.Error(t, err)
		assert.Nil(t, roles)
		mockRepo.AssertExpectations(t)
	})
}

func TestGetRoleByID(t *testing.T) {
	mockRepo := new(mocks.RoleRepositoryInterface)
	roleService := service2.NewRoleService(mockRepo)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		expectedRole := &entity.RoleEntity{ID: 1, Name: "Admin"}
		mockRepo.On("GetRoleByID", ctx, int64(1)).Return(expectedRole, nil).Once()

		role, err := roleService.GetRoleByID(ctx, 1)

		assert.NoError(t, err)
		assert.Equal(t, expectedRole, role)
		mockRepo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo.On("GetRoleByID", ctx, int64(2)).Return(nil, errors.New("not found")).Once()

		role, err := roleService.GetRoleByID(ctx, 2)

		assert.Error(t, err)
		assert.Nil(t, role)
		mockRepo.AssertExpectations(t)
	})
}

func TestCreateRole(t *testing.T) {
	mockRepo := new(mocks.RoleRepositoryInterface)
	roleService := service2.NewRoleService(mockRepo)
	ctx := context.Background()
	role := entity.RoleEntity{Name: "NewRole"}

	t.Run("success", func(t *testing.T) {
		mockRepo.On("CreateRole", ctx, role).Return(nil).Once()

		err := roleService.CreateRole(ctx, role)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		mockRepo.On("CreateRole", ctx, role).Return(errors.New("creation failed")).Once()

		err := roleService.CreateRole(ctx, role)

		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestUpdateRole(t *testing.T) {
	mockRepo := new(mocks.RoleRepositoryInterface)
	roleService := service2.NewRoleService(mockRepo)
	ctx := context.Background()
	role := entity.RoleEntity{ID: 1, Name: "UpdatedRole"}

	t.Run("success", func(t *testing.T) {
		mockRepo.On("UpdateRole", ctx, role).Return(nil).Once()

		err := roleService.UpdateRole(ctx, role)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		mockRepo.On("UpdateRole", ctx, role).Return(errors.New("update failed")).Once()

		err := roleService.UpdateRole(ctx, role)

		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestDeleteRoleByID(t *testing.T) {
	mockRepo := new(mocks.RoleRepositoryInterface)
	roleService := service2.NewRoleService(mockRepo)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo.On("DeleteRoleByID", ctx, int64(1)).Return(nil).Once()

		err := roleService.DeleteRoleByID(ctx, 1)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		mockRepo.On("DeleteRoleByID", ctx, int64(2)).Return(errors.New("delete failed")).Once()

		err := roleService.DeleteRoleByID(ctx, 2)

		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})
}
