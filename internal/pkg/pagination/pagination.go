package pagination

const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 200
)

type Params struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

type Result[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

func Normalize(page, pageSize int) Params {
	if page <= 0 {
		page = DefaultPage
	}
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return Params{Page: page, PageSize: pageSize}
}

func (p Params) Offset() int {
	p = Normalize(p.Page, p.PageSize)
	return (p.Page - 1) * p.PageSize
}
