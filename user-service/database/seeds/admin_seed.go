package seeds

import (
	"log"
	"user-service/internal/core/domain/model"
	"user-service/utils/conv"

	"gorm.io/gorm"
)

func SeedAdmin(db *gorm.DB) {
	password := "rahasia"
	bytes, err := conv.HashPassword(password)
	if err != nil {
		log.Fatalf("%s: %v", err.Error(), err)
	}

	modelRole := model.Role{}
	err = db.Where("name = ?", "Super Admin").First(&modelRole).Error
	if err != nil {
		log.Fatalf("%s: %v", err.Error(), err)
	}

	admin := model.User{
		Name:       "Super Admin",
		Email:      "superadmin@mail.com",
		Password:   bytes,
		IsVerified: true,
		Roles:      []model.Role{modelRole},
	}

	if err := db.FirstOrCreate(&admin, model.User{Email: admin.Email}).Error; err != nil {
		log.Fatalf("Failed to create admin user: %v", err)
	}

	log.Println("Default super admin created or already exists")
}
