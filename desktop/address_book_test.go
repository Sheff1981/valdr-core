package desktop

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Sheff1981/valdr-core/wallet"
)

func TestAddressBookStoreCRUDValidationAndPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "address-book.json")
	store := NewAddressBookStore(path)

	first, err := wallet.New("contact-first")
	if err != nil {
		t.Fatal(err)
	}
	second, err := wallet.New("contact-second")
	if err != nil {
		t.Fatal(err)
	}

	contacts, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(contacts) != 0 {
		t.Fatalf("initial contacts=%d want=0", len(contacts))
	}

	created, err := store.Create("Alice", first.Address)
	if err != nil {
		t.Fatal(err)
	}
	if created.Label != "Alice" || created.Address != first.Address {
		t.Fatalf("created=%+v", created)
	}
	if _, err := store.Create("Duplicate", first.Address); !errors.Is(err, ErrAddressBookDuplicate) {
		t.Fatalf("duplicate error=%v", err)
	}
	if _, err := store.Create("", second.Address); !errors.Is(err, ErrAddressBookInvalid) {
		t.Fatalf("empty label error=%v", err)
	}
	if _, err := store.Create("Bad", "VDR1-not-an-address"); !errors.Is(err, ErrAddressBookInvalid) {
		t.Fatalf("invalid address error=%v", err)
	}

	updated, err := store.Update(first.Address, "Bob", second.Address)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Label != "Bob" || updated.Address != second.Address {
		t.Fatalf("updated=%+v", updated)
	}

	contacts, err = store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(contacts) != 1 || contacts[0] != updated {
		t.Fatalf("contacts=%+v", contacts)
	}

	if err := store.Delete(second.Address); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(second.Address); !errors.Is(err, ErrAddressBookNotFound) {
		t.Fatalf("second delete error=%v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("address-book permissions=%o want owner-only", info.Mode().Perm())
	}
}
