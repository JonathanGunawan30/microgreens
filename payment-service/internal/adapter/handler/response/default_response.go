package response

type DefaultResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type DefaultResponseWithPagination struct {
	Message    string         `json:"message"`
	Data       interface{}    `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

type PaginationMeta struct {
	Page       int64 `json:"page"`
	TotalCount int64 `json:"total_count"`
	PerPage    int64 `json:"per_page"`
	TotalPage  int64 `json:"total_page"`
}

func Success(message string, data any) DefaultResponse {
	return DefaultResponse{
		Message: message,
		Data:    data,
	}
}

func SuccessWithPagination(message string, data any, page, totalCount, perPage, totalPage int64) DefaultResponseWithPagination {
	return DefaultResponseWithPagination{
		Message: message,
		Data:    data,
		Pagination: PaginationMeta{
			Page:       page,
			TotalCount: totalCount,
			PerPage:    perPage,
			TotalPage:  totalPage,
		},
	}
}

func Error(message string) DefaultResponse {
	return DefaultResponse{
		Message: message,
		Data:    nil,
	}
}
