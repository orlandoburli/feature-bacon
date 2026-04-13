package pagination

import (
	"testing"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
)

func TestParse_Defaults(t *testing.T) {
	page, perPage := Parse(nil)
	if page != 1 || perPage != DefaultPerPage {
		t.Fatalf("expected 1/%d, got %d/%d", DefaultPerPage, page, perPage)
	}
}

func TestParse_Custom(t *testing.T) {
	page, perPage := Parse(&pb.PageRequest{Page: 3, PerPage: 50})
	if page != 3 || perPage != 50 {
		t.Fatalf("expected 3/50, got %d/%d", page, perPage)
	}
}

func TestParse_ClampsMax(t *testing.T) {
	_, perPage := Parse(&pb.PageRequest{Page: 1, PerPage: 999})
	if perPage != MaxPerPage {
		t.Fatalf("expected %d, got %d", MaxPerPage, perPage)
	}
}

func TestInfo(t *testing.T) {
	pi := Info(2, 10, 25)
	if pi.Page != 2 || pi.PerPage != 10 || pi.Total != 25 || pi.TotalPages != 3 {
		t.Fatalf("unexpected: %+v", pi)
	}
}

func TestInfo_Exact(t *testing.T) {
	pi := Info(1, 10, 20)
	if pi.TotalPages != 2 {
		t.Fatalf("expected 2, got %d", pi.TotalPages)
	}
}
