package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/config"
)

func TestGenerateManifestDeterministicAndHashesArtifacts(t *testing.T) {
	dir := t.TempDir()
	firstName := "VALDR-Desktop-" + config.Version + "-linux-x64.AppImage"
	secondName := "VALDR-Desktop-" + config.Version + "-windows-x64-setup.exe"

	firstBytes := []byte("valdr-linux-artifact\n")
	secondBytes := []byte("valdr-windows-artifact\n")
	if err := os.WriteFile(filepath.Join(dir, firstName), firstBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, secondName), secondBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	releasedAt := time.Date(2026, 9, 25, 11, 30, 0, 0, time.FixedZone("UTC+3", 3*60*60))
	options := manifestOptions{
		ArtifactsDir: dir,
		Commit:       strings.Repeat("A", 40),
		ReleasedAt:   releasedAt,
		Development:  true,
		Metadata: []artifactMetadata{
			{
				Filename:           secondName,
				OS:                 "windows",
				Arch:               "amd64",
				MinimumOS:          "Windows 10",
				SigningStatus:      "unsigned-development",
				NotarizationStatus: "not_applicable",
			},
			{
				Filename:           firstName,
				OS:                 "linux",
				Arch:               "amd64",
				MinimumOS:          "ci-proven-baseline",
				SigningStatus:      "unsigned-development",
				NotarizationStatus: "not_applicable",
			},
		},
	}

	one, err := generateManifest(options)
	if err != nil {
		t.Fatal(err)
	}
	two, err := generateManifest(options)
	if err != nil {
		t.Fatal(err)
	}

	oneJSON, err := json.Marshal(one)
	if err != nil {
		t.Fatal(err)
	}
	twoJSON, err := json.Marshal(two)
	if err != nil {
		t.Fatal(err)
	}
	if string(oneJSON) != string(twoJSON) {
		t.Fatalf("manifest generation is not deterministic:\n%s\n%s", oneJSON, twoJSON)
	}

	if one.Version != config.Version {
		t.Fatalf("version = %q, want %q", one.Version, config.Version)
	}
	if one.GitCommit != strings.Repeat("a", 40) {
		t.Fatalf("git commit = %q", one.GitCommit)
	}
	if one.Network != config.NetworkTestnetV02 ||
		one.ChainID != "valdr-testnet-1" ||
		one.ProtocolMin != 2 ||
		one.ProtocolMax != 2 {
		t.Fatalf("unexpected Testnet identity: %+v", one)
	}
	if one.ReleaseKind != "development" || !one.Development {
		t.Fatalf("development identity not preserved: %+v", one)
	}
	if one.ProvenanceMethod != "github-sigstore-keyless" || !one.ProvenanceRequired {
		t.Fatalf("provenance policy not preserved: %+v", one)
	}
	if one.ReleasedAt != "2026-09-25T08:30:00Z" {
		t.Fatalf("released_at = %q", one.ReleasedAt)
	}
	if len(one.Artifacts) != 2 {
		t.Fatalf("artifact count = %d", len(one.Artifacts))
	}
	if one.Artifacts[0].Filename != firstName || one.Artifacts[1].Filename != secondName {
		t.Fatalf("artifacts not sorted: %+v", one.Artifacts)
	}

	expectedFirst := sha256.Sum256(firstBytes)
	if one.Artifacts[0].SHA256 != hex.EncodeToString(expectedFirst[:]) {
		t.Fatalf("first sha256 = %q", one.Artifacts[0].SHA256)
	}
	if one.Artifacts[0].SizeBytes != int64(len(firstBytes)) {
		t.Fatalf("first size = %d", one.Artifacts[0].SizeBytes)
	}
}

func TestGenerateManifestRejectsDevelopmentVersionForRelease(t *testing.T) {
	if !isDevelopmentVersion(config.Version) {
		t.Skip("test applies while repository version is a development version")
	}
	_, err := generateManifest(manifestOptions{
		ArtifactsDir: t.TempDir(),
		Commit:       strings.Repeat("a", 40),
		ReleasedAt:   time.Now().UTC(),
		Development:  false,
	})
	if err == nil || !strings.Contains(err.Error(), "cannot use development version") {
		t.Fatalf("expected development-version release rejection, got %v", err)
	}
}

func TestGenerateManifestRejectsUnsafeFilename(t *testing.T) {
	_, err := generateManifest(manifestOptions{
		ArtifactsDir: t.TempDir(),
		Commit:       strings.Repeat("b", 40),
		ReleasedAt:   time.Now().UTC(),
		Development:  true,
		Metadata: []artifactMetadata{
			{
				Filename:           "../VALDR-Desktop-" + config.Version + "-linux-x64.AppImage",
				OS:                 "linux",
				Arch:               "amd64",
				MinimumOS:          "test",
				SigningStatus:      "unsigned-development",
				NotarizationStatus: "not_applicable",
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "unsafe artifact filename") {
		t.Fatalf("expected unsafe filename rejection, got %v", err)
	}
}

func TestGenerateManifestRejectsDuplicateArtifact(t *testing.T) {
	dir := t.TempDir()
	name := "VALDR-Desktop-" + config.Version + "-linux-x64.AppImage"
	if err := os.WriteFile(filepath.Join(dir, name), []byte("artifact"), 0o644); err != nil {
		t.Fatal(err)
	}
	item := artifactMetadata{
		Filename:           name,
		OS:                 "linux",
		Arch:               "amd64",
		MinimumOS:          "test",
		SigningStatus:      "unsigned-development",
		NotarizationStatus: "not_applicable",
	}

	_, err := generateManifest(manifestOptions{
		ArtifactsDir: dir,
		Commit:       strings.Repeat("c", 40),
		ReleasedAt:   time.Now().UTC(),
		Development:  true,
		Metadata:     []artifactMetadata{item, item},
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate artifact filename") {
		t.Fatalf("expected duplicate rejection, got %v", err)
	}
}

func TestGenerateManifestRejectsMissingArtifact(t *testing.T) {
	name := "VALDR-Desktop-" + config.Version + "-macos-arm64.dmg"
	_, err := generateManifest(manifestOptions{
		ArtifactsDir: t.TempDir(),
		Commit:       strings.Repeat("d", 40),
		ReleasedAt:   time.Now().UTC(),
		Development:  true,
		Metadata: []artifactMetadata{
			{
				Filename:           name,
				OS:                 "macos",
				Arch:               "arm64",
				MinimumOS:          "macOS 11.0",
				SigningStatus:      "unsigned-development",
				NotarizationStatus: "notarized-development-no",
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "no such file") {
		t.Fatalf("expected missing artifact rejection, got %v", err)
	}
}

func TestNormalizeCommitRejectsInvalidValue(t *testing.T) {
	for _, value := range []string{
		"",
		"abc",
		strings.Repeat("g", 40),
		strings.Repeat("a", 39),
		strings.Repeat("a", 41),
	} {
		if _, err := normalizeCommit(value); err == nil {
			t.Fatalf("normalizeCommit(%q) unexpectedly succeeded", value)
		}
	}
}


func TestValidateProductionSigningMetadataRejectsDevelopmentClaims(t *testing.T) {
	cases := []artifactMetadata{
		{
			Filename:           "VALDR-Desktop-" + config.Version + "-windows-x64-setup.exe",
			OS:                 "windows",
			Arch:               "amd64",
			MinimumOS:          "Windows 10/11 x64",
			SigningStatus:      "unsigned-development",
			NotarizationStatus: "not_applicable",
		},
		{
			Filename:           "VALDR-Desktop-" + config.Version + "-macos-arm64.dmg",
			OS:                 "macos",
			Arch:               "arm64",
			MinimumOS:          "macOS 11.0+ ARM64",
			SigningStatus:      "adhoc-development",
			NotarizationStatus: "not-notarized-development",
		},
		{
			Filename:           "VALDR-Desktop-" + config.Version + "-linux-x64.AppImage",
			OS:                 "linux",
			Arch:               "amd64",
			MinimumOS:          "Ubuntu 24.04 CI baseline",
			SigningStatus:      "unsigned-development",
			NotarizationStatus: "not_applicable",
		},
	}

	for _, item := range cases {
		if err := validateArtifactMetadata(item, false); err == nil {
			t.Fatalf("production metadata unexpectedly accepted development claim: %+v", item)
		}
		if err := validateArtifactMetadata(item, true); err != nil {
			t.Fatalf("development metadata unexpectedly rejected: %v", err)
		}
	}
}

func TestValidateProductionSigningMetadataAllowsExplicitUnsignedVendorState(t *testing.T) {
	cases := []artifactMetadata{
		{
			Filename:           "VALDR-Desktop-" + config.Version + "-windows-x64-setup.exe",
			OS:                 "windows",
			Arch:               "amd64",
			MinimumOS:          "Windows 10/11 x64",
			SigningStatus:      "unsigned",
			NotarizationStatus: "not_applicable",
		},
		{
			Filename:           "VALDR-Desktop-" + config.Version + "-macos-arm64.dmg",
			OS:                 "macos",
			Arch:               "arm64",
			MinimumOS:          "macOS 11.0+ ARM64",
			SigningStatus:      "adhoc",
			NotarizationStatus: "not-notarized",
		},
		{
			Filename:           "VALDR-Desktop-" + config.Version + "-linux-x64.AppImage",
			OS:                 "linux",
			Arch:               "amd64",
			MinimumOS:          "Linux x86_64",
			SigningStatus:      "unsigned",
			NotarizationStatus: "not_applicable",
		},
	}

	for _, item := range cases {
		if err := validateArtifactMetadata(item, false); err != nil {
			t.Fatalf("explicit unsigned vendor state rejected for %s: %v", item.OS, err)
		}
	}
}

func TestValidateProductionArtifactSetRequiresMandatoryTargets(t *testing.T) {
	version := config.Version
	items := []artifactMetadata{
		{Filename: "VALDR-Desktop-" + version + "-windows-x64-setup.exe", OS: "windows", Arch: "amd64"},
		{Filename: "VALDR-Desktop-" + version + "-windows-x64-portable.zip", OS: "windows", Arch: "amd64"},
		{Filename: "VALDR-Desktop-" + version + "-linux-x64.AppImage", OS: "linux", Arch: "amd64"},
		{Filename: "VALDR-Desktop-" + version + "-linux-amd64.deb", OS: "linux", Arch: "amd64"},
		{Filename: "VALDR-Desktop-" + version + "-macos-arm64.dmg", OS: "macos", Arch: "arm64"},
		{Filename: "VALDR-Desktop-" + version + "-macos-x64.dmg", OS: "macos", Arch: "amd64"},
	}

	if err := validateProductionArtifactSet(items); err != nil {
		t.Fatalf("complete mandatory release set rejected: %v", err)
	}

	incomplete := append([]artifactMetadata(nil), items[:len(items)-1]...)
	err := validateProductionArtifactSet(incomplete)
	if err == nil || !strings.Contains(err.Error(), "macOS Intel AMD64") {
		t.Fatalf("expected missing Intel macOS artifact rejection, got %v", err)
	}
}

func TestValidateProductionArtifactSetAllowsUniversalMac(t *testing.T) {
	version := config.Version
	items := []artifactMetadata{
		{Filename: "VALDR-Desktop-" + version + "-windows-x64-setup.exe", OS: "windows", Arch: "amd64"},
		{Filename: "VALDR-Desktop-" + version + "-windows-x64-portable.zip", OS: "windows", Arch: "amd64"},
		{Filename: "VALDR-Desktop-" + version + "-linux-x64.AppImage", OS: "linux", Arch: "amd64"},
		{Filename: "VALDR-Desktop-" + version + "-linux-amd64.deb", OS: "linux", Arch: "amd64"},
		{Filename: "VALDR-Desktop-" + version + "-macos-universal.dmg", OS: "macos", Arch: "universal"},
	}

	if err := validateProductionArtifactSet(items); err != nil {
		t.Fatalf("universal macOS release set rejected: %v", err)
	}
}
