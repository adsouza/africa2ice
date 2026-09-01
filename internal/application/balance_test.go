package application

import "testing"

func TestMoistureBalanceGate(t *testing.T) {
	report, err := BuildMoistureBalanceReport()
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateMoistureBalanceReport(report); err != nil {
		t.Fatal(err)
	}
	for index, turn := range []int{0, 100, 200, 300, 400} {
		if report.Samples[index].Turn != turn {
			t.Fatalf("sample %d turn = %d, want %d", index, report.Samples[index].Turn, turn)
		}
	}
}
