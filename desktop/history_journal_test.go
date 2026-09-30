package desktop

import (
	"path/filepath"
	"testing"
)

func TestHistoryJournalTracksConflictedAndReorged(t *testing.T) {
	journal, err := newHistoryJournal(filepath.Join(t.TempDir(), "history.json"))
	if err != nil {
		t.Fatal(err)
	}
	address := "VDR1test"

	pending := TransactionHistoryItem{
		Status:        "pending",
		Direction:     "received",
		Type:          "transfer",
		TransactionID: "tx-pending",
		AmountVal:     10,
		AmountVDR:     "0.00000010",
	}
	got, err := journal.Merge(address, []TransactionHistoryItem{pending})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Status != "pending" {
		t.Fatalf("unexpected pending merge: %+v", got)
	}
	got, err = journal.Merge(address, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Status != "conflicted" {
		t.Fatalf("unexpected conflicted merge: %+v", got)
	}

	confirmed := TransactionHistoryItem{
		Status:        "confirmed",
		Direction:     "sent",
		Type:          "transfer",
		TransactionID: "tx-confirmed",
		BlockHeight:   12,
		BlockHash:     "block",
		Confirmations: 3,
	}
	got, err = journal.Merge(address, []TransactionHistoryItem{confirmed})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("unexpected confirmed merge: %+v", got)
	}
	got, err = journal.Merge(address, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range got {
		if item.TransactionID == "tx-confirmed" {
			found = true
			if item.Status != "reorged" || item.Confirmations != 0 ||
				item.BlockHeight != 0 || item.BlockHash != "" {
				t.Fatalf("unexpected reorged item: %+v", item)
			}
		}
	}
	if !found {
		t.Fatal("reorged transaction missing")
	}
}
