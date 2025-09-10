package handler

// Response is a generic JSON response wrapper.
// It can be used for single item responses or for errors.
type Response struct {
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

// PaginatedResponse is a generic JSON response for paginated results.
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination holds pagination metadata.
type Pagination struct {
	Total           int64  `json:"total"`
	Page            int    `json:"page,omitempty"`              // Only for traditional pagination
	PageSize        int    `json:"page_size"`                   // Renamed for consistency
	TotalPages      int64  `json:"total_pages,omitempty"`       // Only for traditional pagination
	NextSearchAfter string `json:"next_search_after,omitempty"` // Only for search_after pagination
	HasNext         bool   `json:"has_next"`                    // Indicates if there are more pages
}

// NewPagination creates a new Pagination struct with calculated total pages for traditional pagination.
func NewPagination(page, pageSize int, total int64) Pagination {
	var totalPages int64
	if pageSize > 0 {
		totalPages = (total + int64(pageSize) - 1) / int64(pageSize)
	}
	hasNext := int64(page) < totalPages
	return Pagination{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		HasNext:    hasNext,
	}
}

// NewSearchAfterPagination creates a new Pagination struct for search_after pagination.
func NewSearchAfterPagination(pageSize int, total int64, nextSearchAfter string, hasNext bool) Pagination {
	return Pagination{
		Total:           total,
		PageSize:        pageSize,
		NextSearchAfter: nextSearchAfter,
		HasNext:         hasNext,
	}
}
