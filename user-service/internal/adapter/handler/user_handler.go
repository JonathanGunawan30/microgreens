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
	CreateUserAccount(c echo.Context) error
	ForgotPassword(c echo.Context) error
	VerifyAccount(c echo.Context) error
	UpdatePassword(c echo.Context) error
	GetUserProfile(c echo.Context) error
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

	return userHandler
}

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
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusCreated, resp)
}

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
		if errors.Is(err, message.ErrUserNotFound) {
			log.Errorf("[UserHandler - 4] UpdateDataUser: %v", err)
			resp.Message = "user not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}

		log.Errorf("[UserHandler - 5] UpdateDataUser: %v", err)
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}
	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusOK, resp)

}

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
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, err)
	}

	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusCreated, resp)
}

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
		ID:       customerID,
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Address:  req.Address,
		Lat:      req.Lat,
		Lng:      req.Lng,
		Phone:    req.Phone,
		Photo:    req.Photo,
	}

	err = u.userService.UpdateCustomer(ctx, userEntity)
	if err != nil {
		if errors.Is(err, message.ErrCustomerNotFound) {
			log.Errorf("[UserHandler - 5] UpdateCustomer: %v", err)
			resp.Message = err.Error()
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}

		log.Errorf("[UserHandler - 6] UpdateCustomer: %v", err)
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusOK, resp)
}

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
