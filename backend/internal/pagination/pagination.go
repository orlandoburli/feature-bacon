package pagination

import pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"

const (
	DefaultPerPage int32 = 20
	MaxPerPage     int32 = 100
)

func Parse(pr *pb.PageRequest) (page, perPage int32) {
	page, perPage = 1, DefaultPerPage
	if pr != nil {
		if pr.Page > 0 {
			page = pr.Page
		}
		if pr.PerPage > 0 {
			perPage = pr.PerPage
		}
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}
	return page, perPage
}

func Info(page, perPage, total int32) *pb.PageInfo {
	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}
	return &pb.PageInfo{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}
}
