package handler

import (
	"errors"
	"net/http"
	"user-service/config"
	"user-service/internal/adapter"
	"user-service/internal/adapter/handler/request"
	"user-service/internal/adapter/handler/response"
	"user-service/internal/core/domain/entity"
	"user-service/internal/core/service"
	"user-service/utils/conv"
	"user-service/utils/message"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/redis/go-redis/v9"
)

type UserHandlerInterface interface {
	SignIn(c echo.Context) error
	SignOut(c echo.Context) error
	CreateUserAccount(c echo.Context) error
	ForgotPassword(c echo.Context) error
	VerifyAccount(c echo.Context) error
	UpdatePassword(c echo.Context) error
	GetUserProfile(c echo.Context) error
	UpdateProfilePassword(c echo.Context) error
	UpdateDataUser(c echo.Context) error

	// Customers
	GetAllCustomers(c echo.Context) error
	GetCustomerByID(c echo.Context) error
	CreateCustomer(c echo.Context) error
	UpdateCustomer(c echo.Context) error
	DeleteCustomer(c echo.Context) error
}

type userHandler struct {
	userService service.UserServiceInterface
}

func NewUserHandler(e *echo.Echo, userService service.UserServiceInterface, cfg *config.Config, redisClient *redis.Client) UserHandlerInterface {
	userHandler := &userHandler{userService: userService}

	e.Use(middleware.Recover())
	// public route
	e.POST("/signin", userHandler.SignIn)
	e.POST("/signup", userHandler.CreateUserAccount)
	e.POST("/forgot-password", userHandler.ForgotPassword)
	e.GET("/verify-account", userHandler.VerifyAccount)
	e.PUT("/update-password", userHandler.UpdatePassword)

	mid := adapter.NewMiddlewareAdapter(cfg, redisClient)

	// admin group
	adminGroup := e.Group("/admin", mid.CheckToken(cfg.App.JwtSecretKey))
	adminGroup.GET("/customers", userHandler.GetAllCustomers)
	adminGroup.GET("/customers/:id", userHandler.GetCustomerByID)
	adminGroup.POST("/customers", userHandler.CreateCustomer)
	adminGroup.PUT("/customers/:id", userHandler.UpdateCustomer)
	adminGroup.DELETE("/customers/:id", userHandler.DeleteCustomer)

	// auth group
	authGroup := e.Group("/auth", mid.CheckToken(cfg.App.JwtSecretKey))
	authGroup.GET("/profile", userHandler.GetUserProfile)
	authGroup.PUT("/profile", userHandler.UpdateDataUser)
	authGroup.PATCH("/profile/password", userHandler.UpdateProfilePassword)
	authGroup.POST("/logout", userHandler.SignOut)

	return userHandler
}

// SignIn godoc
// @Summary User sign in
// @Description Authenticate user and return JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body request.SignInRequest true "Sign In Request"
// @Success 200 {object} response.DefaultResponse{data=response.SignInResponse} "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 422 {object} response.DefaultResponse "Validation Error"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /signin [post]
func (u *userHandler) SignIn(c echo.Context) error {
	var (
		req        = request.SignInRequest{}
		resp       = response.DefaultResponse{}
		respSignIn = response.SignInResponse{}
		ctx        = c.Request().Context()
	)

	if err := c.Bind(&req); err != nil {
		log.Errorf("[UserHandler - 1] SignIn: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if err := c.Validate(req); err != nil {
		log.Errorf("[UserHandler - 2] SignIn: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusUnprocessableEntity, resp)
	}

	reqEntity := entity.UserEntity{
		Email:    req.Email,
		Password: req.Password,
	}

	user, token, err := u.userService.SignIn(ctx, reqEntity)
	if err != nil {
		if errors.Is(err, message.ErrInvalidCredential) {
			log.Errorf("[UserHandler - 3] SignIn: %v", err)
			resp.Message = "username or password is wrong"
			resp.Data = nil
			return c.JSON(http.StatusUnauthorized, resp)
		}

		log.Errorf("[UserHandler - 4] SignIn: %v", err)
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	respSignIn.ID = user.ID
	respSignIn.Name = user.Name
	respSignIn.Email = user.Email
	respSignIn.Role = user.RoleName
	respSignIn.Address = user.Address
	respSignIn.Lat = user.Lat
	respSignIn.Lng = user.Lng
	respSignIn.Phone = user.Phone
	respSignIn.AccessToken = token

	resp.Message = "Success"
	resp.Data = respSignIn
	return c.JSON(http.StatusOK, resp)
}

// SignOut godoc
// @Summary User sign out
// @Description Revoke user session
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.DefaultResponse "Success"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /auth/logout [post]
func (u *userHandler) SignOut(c echo.Context) error {
	var (
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	jwtUserData, ok := c.Get("user").(entity.JwtUserData)
	if !ok {
		log.Errorf("[UserHandler - 1] UpdateDataUser: invalid user context")
		resp.Message = "invalid user context"
		resp.Data = nil
		return c.JSON(http.StatusUnauthorized, resp)
	}

	userID := jwtUserData.ID
	if userID == 0 {
		log.Errorf("[UserHandler - 2] UpdateDataUser: invalid user id")
		resp.Message = "invalid user id"
		resp.Data = nil
		return c.JSON(http.StatusUnauthorized, resp)
	}

	err := u.userService.SignOut(ctx, userID)
	if err != nil {
		log.Errorf("[UserHandler - 3] SignOut: %v", err)
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusOK, resp)

}

// CreateUserAccount godoc
// @Summary User sign up
// @Description Create a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param request body request.SignUpRequest true "Sign Up Request"
// @Success 201 {object} response.DefaultResponse "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request"
// @Failure 409 {object} response.DefaultResponse "Conflict"
// @Failure 422 {object} response.DefaultResponse "Validation Error"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /signup [post]
func (u *userHandler) CreateUserAccount(c echo.Context) error {
	var (
		req  = request.SignUpRequest{}
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	if err := c.Bind(&req); err != nil {
		log.Errorf("[UserHandler - 1] CreateUserAccount: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if err := c.Validate(req); err != nil {
		log.Errorf("[UserHandler - 2] CreateUserAccount: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusUnprocessableEntity, resp)
	}

	userEntity := entity.UserEntity{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	}

	err := u.userService.CreateUserAccount(ctx, userEntity)
	if err != nil {
		log.Errorf("[UserHandler - 3] CreateUserAccount: %v", err)

		if errors.Is(err, message.ErrEmailAlreadyExists) {
			resp.Message = "email already exists"
			resp.Data = nil
			return c.JSON(http.StatusConflict, resp)
		}

		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusCreated, resp)
}

// ForgotPassword godoc
// @Summary Forgot password
// @Description Send password reset link to email
// @Tags auth
// @Accept json
// @Produce json
// @Param request body request.ForgotPasswordRequest true "Forgot Password Request"
// @Success 200 {object} response.DefaultResponse "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request"
// @Failure 404 {object} response.DefaultResponse "Not Found"
// @Failure 422 {object} response.DefaultResponse "Validation Error"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /forgot-password [post]
func (u *userHandler) ForgotPassword(c echo.Context) error {
	var (
		req  = request.ForgotPasswordRequest{}
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	if err := c.Bind(&req); err != nil {
		log.Errorf("[UserHandler - 1] ForgotPassword: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if err := c.Validate(req); err != nil {
		log.Errorf("[UserHandler - 2] ForgotPassword: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusUnprocessableEntity, resp)
	}

	reqEntity := entity.UserEntity{
		Email: req.Email,
	}

	err := u.userService.ForgotPassword(ctx, reqEntity)
	if err != nil {
		if errors.Is(err, message.ErrUserNotFound) {
			resp.Message = "email not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}
		log.Errorf("[UserHandler - 3] ForgotPassword: %v", err)
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusOK, resp)

}

// VerifyAccount godoc
// @Summary Verify account
// @Description Verify user account using token from email
// @Tags auth
// @Accept json
// @Produce json
// @Param token query string true "Verification Token"
// @Success 200 {object} response.DefaultResponse{data=response.SignInResponse} "Success"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 404 {object} response.DefaultResponse "Not Found"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /verify-account [get]
func (u *userHandler) VerifyAccount(c echo.Context) error {
	var (
		resp       = response.DefaultResponse{}
		respSignIn = response.SignInResponse{}
		ctx        = c.Request().Context()
	)

	token := c.QueryParam("token")
	if token == "" {
		log.Errorf("[UserHandler - 1] VerifyAccount: missing or invalid token")
		resp.Message = "missing or invalid token"
		resp.Data = nil
		return c.JSON(http.StatusUnauthorized, resp)
	}

	user, err := u.userService.VerifyToken(ctx, token)

	if err != nil {
		switch {
		case errors.Is(err, message.ErrTokenNotFound):
			log.Errorf("[UserHandler - 2] VerifyAccount: %v", err)
			resp.Message = "token not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)

		case errors.Is(err, message.ErrTokenExpired):
			log.Errorf("[UserHandler - 3] VerifyAccount: %v", err)
			resp.Message = "token expired"
			resp.Data = nil
			return c.JSON(http.StatusUnauthorized, resp)

		case errors.Is(err, message.ErrUserNotFound):
			log.Errorf("[UserHandler - 4] VerifyAccount: %v", err)
			resp.Message = "user not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)

		case errors.Is(err, message.ErrSessionFailed):
			log.Errorf("[UserHandler - 5] VerifyAccount: %v", err)
			resp.Message = "session failed"
			resp.Data = nil
			return c.JSON(http.StatusInternalServerError, resp)

		default:
			log.Errorf("[UserHandler - 6] VerifyAccount: %v", err)
			resp.Message = "internal server error"
			resp.Data = nil
			return c.JSON(http.StatusInternalServerError, resp)
		}
	}

	respSignIn.ID = user.ID
	respSignIn.Name = user.Name
	respSignIn.Email = user.Email
	respSignIn.Role = user.RoleName
	respSignIn.Address = user.Address
	respSignIn.Lat = user.Lat
	respSignIn.Lng = user.Lng
	respSignIn.Phone = user.Phone
	respSignIn.AccessToken = user.Token
	resp.Message = "Success"
	resp.Data = respSignIn
	return c.JSON(http.StatusOK, resp)
}

// UpdatePassword godoc
// @Summary Update password (via reset link)
// @Description Update user password using reset token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body request.UpdatePasswordRequest true "Update Password Request"
// @Success 200 {object} response.DefaultResponse "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request"
// @Failure 404 {object} response.DefaultResponse "Not Found"
// @Failure 422 {object} response.DefaultResponse "Validation Error"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /update-password [put]
func (u *userHandler) UpdatePassword(c echo.Context) error {
	var (
		req  = request.UpdatePasswordRequest{}
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	if err := c.Bind(&req); err != nil {
		log.Errorf("[UserHandler - 1] UpdatePassword: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if err := c.Validate(req); err != nil {
		log.Errorf("[UserHandler - 2] UpdatePassword: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusUnprocessableEntity, resp)
	}

	reqEntity := entity.UserEntity{
		Password: req.NewPassword,
		Token:    req.Token,
	}

	err := u.userService.UpdatePassword(ctx, reqEntity)
	if err != nil {
		switch {
		case errors.Is(err, message.ErrTokenNotFound):
			log.Errorf("[UserHandler - 3] UpdatePassword: %v", err)
			resp.Message = "token not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)

		case errors.Is(err, message.ErrUserNotFound):
			log.Errorf("[UserHandler - 4] UpdatePassword: %v", err)
			resp.Message = "user not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)

		default:
			log.Errorf("[UserHandler - 5] UpdatePassword: %v", err)
			resp.Message = "internal server error"
			resp.Data = nil
			return c.JSON(http.StatusInternalServerError, resp)
		}
	}

	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusOK, resp)

}

// GetUserProfile godoc
// @Summary Get user profile
// @Description Get current authenticated user profile
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.DefaultResponse{data=response.ProfileResponse} "Success"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 404 {object} response.DefaultResponse "Not Found"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /auth/profile [get]
func (u *userHandler) GetUserProfile(c echo.Context) error {
	var (
		resp        = response.DefaultResponse{}
		respProfile = response.ProfileResponse{}
		ctx         = c.Request().Context()
	)

	jwtUserData, ok := c.Get("user").(entity.JwtUserData)
	if !ok {
		log.Errorf("[UserHandler - 1] GetUserProfile: invalid user context")
		resp.Message = "invalid user context"
		resp.Data = nil
		return c.JSON(http.StatusUnauthorized, resp)
	}

	userID := jwtUserData.ID
	if userID == 0 {
		log.Errorf("[UserHandler - 2] GetUserProfile: invalid user id")
		resp.Message = "invalid user id"
		resp.Data = nil
		return c.JSON(http.StatusUnauthorized, resp)
	}

	userProfile, err := u.userService.GetUserProfile(ctx, userID)

	if err != nil {
		if errors.Is(err, message.ErrUserNotFound) {
			log.Errorf("[UserHandler - 1] GetUserProfile: %v", err)
			resp.Message = "user not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}
		log.Errorf("[UserHandler - 2] GetUserProfile: %v", err)
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	respProfile.ID = userProfile.ID
	respProfile.Name = userProfile.Name
	respProfile.Email = userProfile.Email
	respProfile.Phone = userProfile.Phone
	respProfile.Address = userProfile.Address
	respProfile.Lat = userProfile.Lat
	respProfile.Lng = userProfile.Lng
	respProfile.Photo = userProfile.Photo
	respProfile.RoleName = userProfile.RoleName

	resp.Message = "Success"
	resp.Data = respProfile

	return c.JSON(http.StatusOK, resp)
}

// UpdateProfilePassword godoc
// @Summary Update profile password
// @Description Update authenticated user's password
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.UpdateProfilePasswordRequest true "Update Profile Password Request"
// @Success 200 {object} response.DefaultResponse "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 404 {object} response.DefaultResponse "Not Found"
// @Failure 422 {object} response.DefaultResponse "Validation Error"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /auth/profile/password [patch]
func (u *userHandler) UpdateProfilePassword(c echo.Context) error {
	var (
		req  = request.UpdateProfilePasswordRequest{}
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	jwtUserData, ok := c.Get("user").(entity.JwtUserData)
	if !ok {
		log.Errorf("[UserHandler - 1] UpdateProfilePassword: invalid user context")
		resp.Message = "invalid user context"
		resp.Data = nil
		return c.JSON(http.StatusUnauthorized, resp)
	}

	userID := jwtUserData.ID
	if userID == 0 {
		log.Errorf("[UserHandler - 2] UpdateProfilePassword: invalid user id")
		resp.Message = "invalid user id"
		resp.Data = nil
		return c.JSON(http.StatusUnauthorized, resp)
	}

	if err := c.Bind(&req); err != nil {
		log.Errorf("[UserHandler - 3] UpdateProfilePassword: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if err := c.Validate(req); err != nil {
		log.Errorf("[UserHandler - 4] UpdateProfilePassword: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusUnprocessableEntity, resp)
	}

	err := u.userService.UpdateProfilePassword(ctx, userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		log.Errorf("[UserHandler -5] UpdateProfilePassword: %v", err)

		if errors.Is(err, message.ErrUserNotFound) {
			resp.Message = err.Error()
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}

		if errors.Is(err, message.ErrWrongPassword) {
			resp.Message = "current password is incorrect"
			resp.Data = nil
			return c.JSON(http.StatusUnprocessableEntity, resp)
		}

		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusOK, resp)

}

// UpdateDataUser godoc
// @Summary Update profile data
// @Description Update authenticated user's profile data
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.UpdateDataUserRequest true "Update Data User Request"
// @Success 200 {object} response.DefaultResponse "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 404 {object} response.DefaultResponse "Not Found"
// @Failure 409 {object} response.DefaultResponse "Conflict"
// @Failure 422 {object} response.DefaultResponse "Validation Error"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /auth/profile [put]
func (u *userHandler) UpdateDataUser(c echo.Context) error {
	var (
		resp = response.DefaultResponse{}
		req  = request.UpdateDataUserRequest{}
		ctx  = c.Request().Context()
	)

	jwtUserData, ok := c.Get("user").(entity.JwtUserData)
	if !ok {
		log.Errorf("[UserHandler - 1] UpdateDataUser: invalid user context")
		resp.Message = "invalid user context"
		resp.Data = nil
		return c.JSON(http.StatusUnauthorized, resp)
	}

	userID := jwtUserData.ID
	if userID == 0 {
		log.Errorf("[UserHandler - 2] UpdateDataUser: invalid user id")
		resp.Message = "invalid user id"
		resp.Data = nil
		return c.JSON(http.StatusUnauthorized, resp)
	}

	if err := c.Bind(&req); err != nil {
		log.Errorf("[UserHandler - 3] UpdateDataUser: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if err := c.Validate(req); err != nil {
		log.Errorf("[UserHandler - 4] UpdateDataUser: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusUnprocessableEntity, resp)
	}

	reqEntity := entity.UserEntity{
		ID:      userID,
		Name:    req.Name,
		Email:   req.Email,
		Address: req.Address,
		Lat:     req.Lat,
		Lng:     req.Lng,
		Phone:   req.Phone,
		Photo:   req.Photo,
	}

	err := u.userService.UpdateDataUser(ctx, reqEntity)
	if err != nil {
		log.Errorf("[UserHandler - 4] UpdateDataUser: %v", err)

		if errors.Is(err, message.ErrUserNotFound) {
			resp.Message = "user not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}

		if errors.Is(err, message.ErrEmailAlreadyExists) {
			resp.Message = "email already exists"
			resp.Data = nil
			return c.JSON(http.StatusConflict, resp)
		}

		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}
	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusOK, resp)

}

// GetAllCustomers godoc
// @Summary Get all customers
// @Description Get paginated list of customers with optional search and sorting
// @Tags customers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param search query string false "Search by name or email"
// @Param order_by query string false "Order by field (default: created_at)"
// @Param order_type query string false "Order type ASC or DESC (default: DESC)"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Success 200 {object} response.DefaultResponseWithPagination{data=[]response.ProfileResponse} "Success"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /admin/customers [get]
func (u *userHandler) GetAllCustomers(c echo.Context) error {
	var (
		resp = response.DefaultResponseWithPagination{}
		ctx  = c.Request().Context()
	)

	jwtUserData, ok := c.Get("user").(entity.JwtUserData)
	if !ok {
		log.Errorf("[UserHandler - 1] GetAllCustomers: invalid user context")
		resp.Message = "invalid user context"
		resp.Data = nil
		return c.JSON(http.StatusUnauthorized, resp)
	}

	userID := jwtUserData.ID
	if userID == 0 {
		log.Errorf("[UserHandler - 2] GetAllCustomers: invalid user id")
		resp.Message = "invalid user id"
		resp.Data = nil
		return c.JSON(http.StatusUnauthorized, resp)
	}

	search := c.QueryParam("search")
	orderBy := c.QueryParam("order_by")
	if orderBy == "" {
		orderBy = "created_at"
	}

	orderType := c.QueryParam("order_type")
	if orderType == "" {
		orderType = "DESC"
	}

	pageStr := c.QueryParam("page")
	var page int64 = 1
	if pageStr != "" {
		page, _ = conv.StringToInt64(pageStr)
		if page <= 0 {
			page = 1
		}
	}

	limitStr := c.QueryParam("limit")
	var limit int64 = 10
	if limitStr != "" {
		limit, _ = conv.StringToInt64(limitStr)
		if limit <= 0 {
			limit = 10
		}
	}

	customerQuery := entity.QueryStringCustomer{
		Search:    search,
		Page:      page,
		Limit:     limit,
		OrderBy:   orderBy,
		OrderType: orderType,
	}

	customers, count, totalPages, err := u.userService.GetAllCustomers(ctx, customerQuery)
	if err != nil {
		log.Errorf("[UserHandler - 3] GetAllCustomers: %v", err)
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	respUser := make([]response.ProfileResponse, 0, len(customers))

	for _, customer := range customers {
		respUser = append(respUser, response.ProfileResponse{
			ID:    customer.ID,
			Name:  customer.Name,
			Email: customer.Email,
			Phone: customer.Phone,
			Photo: customer.Photo,
		})
	}

	resp.Message = "Success"
	resp.Data = respUser
	resp.Pagination = response.PaginationMeta{
		Page:       page,
		TotalCount: count,
		PerPage:    limit,
		TotalPage:  totalPages,
	}

	return c.JSON(http.StatusOK, resp)
}

// GetCustomerByID godoc
// @Summary Get customer by ID
// @Description Get customer detail by ID
// @Tags customers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Customer ID"
// @Success 200 {object} response.DefaultResponse{data=response.CustomerResponse} "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 404 {object} response.DefaultResponse "Not Found"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /admin/customers/{id} [get]
func (u *userHandler) GetCustomerByID(c echo.Context) error {
	var (
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	jwtUserData, ok := c.Get("user").(entity.JwtUserData)
	if !ok {
		log.Errorf("[UserHandler - 1] GetCustomerByID: invalid user context")
		resp.Message = "invalid user context"
		resp.Data = nil
		return c.JSON(http.StatusUnauthorized, resp)
	}

	userID := jwtUserData.ID
	if userID == 0 {
		log.Errorf("[UserHandler - 2] GetCustomerByID: invalid user id")
		resp.Message = "invalid user id"
		resp.Data = nil
		return c.JSON(http.StatusUnauthorized, resp)
	}

	userIDParam := c.Param("id")
	userID, err := conv.StringToInt64(userIDParam)
	if err != nil || userID <= 0 {
		log.Errorf("[UserHandler - 3] GetCustomerByID: invalid user id: '%s'", userIDParam)
		resp.Message = "invalid user id"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	customer, err := u.userService.GetCustomerByID(ctx, userID)
	if err != nil {
		if errors.Is(err, message.ErrUserNotFound) {
			resp.Message = "customer not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}

		log.Errorf("[UserHandler - 4] GetCustomerByID: %v", err)
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	respUser := response.CustomerResponse{
		ID:       customer.ID,
		Name:     customer.Name,
		Email:    customer.Email,
		Phone:    customer.Phone,
		Photo:    customer.Photo,
		RoleName: customer.RoleName,
		RoleID:   customer.RoleID,
		Address:  customer.Address,
		Lat:      customer.Lat,
		Lng:      customer.Lng,
	}

	resp.Message = "Success"
	resp.Data = respUser
	return c.JSON(http.StatusOK, resp)

}

// CreateCustomer godoc
// @Summary Create customer
// @Description Create a new customer account
// @Tags customers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.CustomerRequest true "Create Customer Request"
// @Success 201 {object} response.DefaultResponse "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 409 {object} response.DefaultResponse "Conflict"
// @Failure 422 {object} response.DefaultResponse "Validation Error"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /admin/customers [post]
func (u *userHandler) CreateCustomer(c echo.Context) error {
	var (
		resp = response.DefaultResponse{}
		req  = request.CustomerRequest{}
		ctx  = c.Request().Context()
	)

	jwtUserData, ok := c.Get("user").(entity.JwtUserData)
	if !ok {
		log.Errorf("[UserHandler - 1] CreateCustomer: invalid user context")
		resp.Message = "invalid user context"
		resp.Data = nil
		return c.JSON(http.StatusUnauthorized, resp)
	}

	userID := jwtUserData.ID
	if userID == 0 {
		log.Errorf("[UserHandler - 2] CreateCustomer: invalid user id")
		resp.Message = "invalid user id"
		resp.Data = nil
		return c.JSON(http.StatusUnauthorized, resp)
	}

	if err := c.Bind(&req); err != nil {
		log.Errorf("[UserHandler - 3] CreateCustomer: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if err := c.Validate(req); err != nil {
		log.Errorf("[UserHandler - 4] CreateCustomer: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusUnprocessableEntity, resp)
	}

	userEntity := entity.UserEntity{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Address:  req.Address,
		Lat:      req.Lat,
		Lng:      req.Lng,
		Phone:    req.Phone,
		Photo:    req.Photo,
	}

	if err := u.userService.CreateCustomer(ctx, userEntity); err != nil {
		log.Errorf("[UserHandler - 5] CreateCustomer: %v", err)

		if errors.Is(err, message.ErrEmailAlreadyExists) {
			resp.Message = "email already exists"
			resp.Data = nil
			return c.JSON(http.StatusConflict, resp)
		}

		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, err)
	}

	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusCreated, resp)
}

// UpdateCustomer godoc
// @Summary Update customer data
// @Description Update customer information by ID
// @Tags customers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Customer ID"
// @Param request body request.UpdateCustomerRequest true "Updated Customer Data"
// @Success 200 {object} response.DefaultResponse "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 404 {object} response.DefaultResponse "Not Found"
// @Failure 409 {object} response.DefaultResponse "Conflict"
// @Failure 422 {object} response.DefaultResponse "Validation Error"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /admin/customers/{id} [put]
func (u *userHandler) UpdateCustomer(c echo.Context) error {
	var (
		resp = response.DefaultResponse{}
		req  = request.UpdateCustomerRequest{}
		ctx  = c.Request().Context()
	)

	jwtUserData, ok := c.Get("user").(entity.JwtUserData)
	if !ok {
		log.Errorf("[UserHandler - 1] UpdateCustomer: invalid user context")
		resp.Message = "invalid user context"
		resp.Data = nil
		return c.JSON(http.StatusUnauthorized, resp)
	}

	userID := jwtUserData.ID
	if userID == 0 {
		log.Errorf("[UserHandler - 2] UpdateCustomer: invalid user id")
		resp.Message = "invalid user id"
		resp.Data = nil
		return c.JSON(http.StatusUnauthorized, resp)
	}

	idParam := c.Param("id")
	customerID, err := conv.StringToInt64(idParam)
	if err != nil {
		resp.Message = "invalid customer id"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if err = c.Bind(&req); err != nil {
		log.Errorf("[UserHandler - 3] UpdateCustomer: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if err = c.Validate(req); err != nil {
		log.Errorf("[UserHandler - 4] UpdateCustomer: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusUnprocessableEntity, resp)
	}

	userEntity := entity.UserEntity{
		ID:      customerID,
		Name:    req.Name,
		Email:   req.Email,
		Address: req.Address,
		Lat:     req.Lat,
		Lng:     req.Lng,
		Phone:   req.Phone,
		Photo:   req.Photo,
	}

	err = u.userService.UpdateCustomer(ctx, userEntity)
	if err != nil {
		log.Errorf("[UserHandler - 5] UpdateCustomer: %v", err)

		if errors.Is(err, message.ErrCustomerNotFound) {
			resp.Message = err.Error()
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}

		if errors.Is(err, message.ErrEmailAlreadyExists) {
			resp.Message = "email already exists"
			resp.Data = nil
			return c.JSON(http.StatusConflict, resp)
		}

		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusOK, resp)
}

// DeleteCustomer godoc
// @Summary Delete customer
// @Description Delete customer by ID
// @Tags customers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Customer ID"
// @Success 200 {object} response.DefaultResponse "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 404 {object} response.DefaultResponse "Not Found"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /admin/customers/{id} [delete]
func (u *userHandler) DeleteCustomer(c echo.Context) error {
	var (
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	jwtUserData, ok := c.Get("user").(entity.JwtUserData)
	if !ok {
		log.Errorf("[UserHandler - 1] DeleteCustomer: invalid user context")
		resp.Message = "invalid user context"
		resp.Data = nil
		return c.JSON(http.StatusUnauthorized, resp)
	}

	userID := jwtUserData.ID
	if userID == 0 {
		log.Errorf("[UserHandler - 2] DeleteCustomer: invalid user id")
		resp.Message = "invalid user id"
		resp.Data = nil
		return c.JSON(http.StatusUnauthorized, resp)
	}

	idParam := c.Param("id")
	customerID, err := conv.StringToInt64(idParam)
	if err != nil {
		resp.Message = "invalid customer id"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	err = u.userService.DeleteCustomer(ctx, customerID)
	if err != nil {
		if errors.Is(err, message.ErrCustomerNotFound) {
			log.Errorf("[UserHandler - 3] DeleteCustomer: %v", err)
			resp.Message = err.Error()
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}

		log.Errorf("[UserHandler - 4] DeleteCustomer: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusOK, resp)
}
