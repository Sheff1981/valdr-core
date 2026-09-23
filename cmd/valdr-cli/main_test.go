package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestWalletCLIFlow(t *testing.T) {
	dir := t.TempDir()

	var out, errOut bytes.Buffer
	if code := run(
		[]string{"wallet", "create", "--dir", dir, "--name", "alice"},
		&out,
		&errOut,
	); code != 0 {
		t.Fatalf("create exit=%d stderr=%s", code, errOut.String())
	}
	if strings.Contains(out.String(), "private_key") {
		t.Fatal("wallet create leaked private key")
	}
	if !strings.Contains(out.String(), "VDR1") {
		t.Fatalf(
			"wallet create output missing VALDR address: %s",
			out.String(),
		)
	}

	out.Reset()
	errOut.Reset()
	if code := run(
		[]string{"wallet", "list", "--dir", dir},
		&out,
		&errOut,
	); code != 0 {
		t.Fatalf("list exit=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "alice") {
		t.Fatalf("wallet list missing alice: %s", out.String())
	}

	out.Reset()
	errOut.Reset()
	if code := run(
		[]string{"wallet", "export", "--dir", dir, "alice"},
		&out,
		&errOut,
	); code != 0 {
		t.Fatalf("export exit=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "private_key") {
		t.Fatal("wallet export did not include private key")
	}
	if !strings.Contains(errOut.String(), "WARNING") {
		t.Fatal("wallet export did not warn about private key exposure")
	}

	out.Reset()
	errOut.Reset()
	if code := run(
		[]string{"wallet", "balance"},
		&out,
		&errOut,
	); code != 2 {
		t.Fatalf(
			"balance exit=%d, want 2 until live node/RPC state is connected",
			code,
		)
	}
	if !strings.Contains(errOut.String(), "RPC") {
		t.Fatalf("balance stderr = %q, want RPC integration message", errOut.String())
	}
}
