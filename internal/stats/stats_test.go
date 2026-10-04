package stats

import (
	"path/filepath"
	"testing"
)

func TestRecordAndSummary(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "logs"), nil)

	s.Record("1.2.3.4")
	s.Record("1.2.3.4")
	s.Record("1.2.3.5")

	sum := s.Summary(7)
	if sum.Today != 3 {
		t.Fatalf("Today = %d, want 3", sum.Today)
	}
	if sum.TodayIP != 2 {
		t.Fatalf("TodayIP = %d, want 2", sum.TodayIP)
	}
	if sum.Total != 3 {
		t.Fatalf("Total = %d, want 3", sum.Total)
	}
	if sum.TotalIP != 2 {
		t.Fatalf("TotalIP = %d, want 2", sum.TotalIP)
	}
	if sum.Month != 3 {
		t.Fatalf("Month = %d, want 3", sum.Month)
	}
	if len(sum.Labels) != 7 || len(sum.PV) != 7 || len(sum.IP) != 7 {
		t.Fatalf("chart length = %d/%d/%d, want 7/7/7", len(sum.Labels), len(sum.PV), len(sum.IP))
	}
	if sum.Labels[6] != "今天" {
		t.Fatalf("last label = %q, want 今天", sum.Labels[6])
	}
	if sum.PV[6] != 3 || sum.IP[6] != 2 {
		t.Fatalf("today chart = %d/%d, want 3/2", sum.PV[6], sum.IP[6])
	}
	// 缺失的日期应记为 0
	if sum.PV[0] != 0 || sum.IP[0] != 0 {
		t.Fatalf("oldest chart = %d/%d, want 0/0", sum.PV[0], sum.IP[0])
	}
}

func TestRecordEmptyIPIgnored(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "logs"), nil)
	s.Record("")
	if got := s.Summary(7).Today; got != 0 {
		t.Fatalf("Today = %d, want 0", got)
	}
}
