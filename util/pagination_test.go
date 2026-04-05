package util

import "testing"

func TestNewPaginatedResponse(t *testing.T) {
	data := []string{"a", "b", "c"}
	params := PaginationParams{Page: 0, Limit: 20}
	resp := NewPaginatedResponse(data, 50, params)

	if resp.Total != 50 {
		t.Errorf("expected total=50, got %d", resp.Total)
	}
	if resp.Pages != 3 {
		t.Errorf("expected pages=3, got %d", resp.Pages)
	}
	if resp.Page != 0 {
		t.Errorf("expected page=0, got %d", resp.Page)
	}
}

func TestNewPaginatedResponse_ExactDivision(t *testing.T) {
	params := PaginationParams{Page: 0, Limit: 10}
	resp := NewPaginatedResponse(nil, 100, params)

	if resp.Pages != 10 {
		t.Errorf("expected pages=10, got %d", resp.Pages)
	}
}
