package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Sheff1981/valdr-core/config"
)

const manifestSchemaVersion = 1

type artifactMetadata struct {
	Filename           string `json:"filename"`
	OS                 string `json:"os"`
	Arch               string `json:"arch"`
	MinimumOS          string `json:"minimum_os"`
	SigningStatus      string `json:"signing_status"`
	NotarizationStatus string `json:"notarization_status"`
}

type metadataFile struct {
	Artifacts []artifactMetadata `json:"artifacts"`
}

type manifestArtifact struct {
	Filename           string `json:"filename"`
	OS                 string `json:"os"`
	Arch               string `json:"arch"`
	MinimumOS          string `json:"minimum_os"`
	SizeBytes          int64  `json:"size_bytes"`
	SHA256             string `json:"sha256"`
	SigningStatus      string `json:"signing_status"`
	NotarizationStatus string `json:"notarization_status"`
}

type releaseManifest struct {
	SchemaVersion int                `json:"schema_version"`
	Product       string             `json:"product"`
	Version       string             `json:"version"`
	GitCommit     string             `json:"git_commit"`
	ReleaseKind   string             `json:"release_kind"`
	Development   bool               `json:"development"`
	Network       string             `json:"network"`
	ChainID       string             `json:"chain_id"`
	ProtocolMin   uint16             `json:"protocol_min"`
	ProtocolMax   uint16             `json:"protocol_max"`
	ReleasedAt    string             `json:"released_at"`
	Artifacts     []manifestArtifact `json:"artifacts"`
}

type manifestOptions struct {
	ArtifactsDir string
	Metadata     []artifactMetadata
	Commit       string
	ReleasedAt   time.Time
	Development bool
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("valdr-release-manifest", flag.ContinueOnError)
	fs.SetOutput(errOut)

	artifactsDir := fs.String("artifacts-dir", "", "directory containing release artifacts")
	metadataPath := fs.String("metadata", "", "artifact metadata JSON")
	commit := fs.String("commit", "", "exact 40-character git commit SHA")
	releasedAt := fs.String("released-at", "", "release timestamp in RFC3339 format")
	output := fs.String("output", "", "output manifest JSON path")
	development := fs.Bool("development", false, "generate an explicitly non-release development manifest")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 || *artifactsDir == "" || *metadataPath == "" ||
		*commit == "" || *releasedAt == "" || *output == "" {
		fmt.Fprintln(errOut, "usage: valdr-release-manifest --artifacts-dir DIR --metadata FILE --commit SHA --released-at RFC3339 --output FILE [--development]")
		return 2
	}

	metadata, err := readMetadata(*metadataPath)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}

	releaseTime, err := time.Parse(time.RFC3339, *releasedAt)
	if err != nil {
		fmt.Fprintf(errOut, "invalid --released-at: %v\n", err)
		return 2
	}

	manifest, err := generateManifest(manifestOptions{
		ArtifactsDir: *artifactsDir,
		Metadata:     metadata.Artifacts,
		Commit:       *commit,
		ReleasedAt:   releaseTime,
		Development:  *development,
	})
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}

	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	raw = append(raw, '\n')

	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if err := os.WriteFile(*output, raw, 0o644); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}

	fmt.Fprintf(out, "wrote %s artifacts=%d development=%t\n", *output, len(manifest.Artifacts), manifest.Development)
	return 0
}

func readMetadata(path string) (metadataFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return metadataFile{}, fmt.Errorf("open metadata: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(io.LimitReader(file, 1<<20))
	decoder.DisallowUnknownFields()

	var metadata metadataFile
	if err := decoder.Decode(&metadata); err != nil {
		return metadataFile{}, fmt.Errorf("decode metadata: %w", err)
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return metadataFile{}, errors.New("metadata contains multiple JSON values")
		}
		return metadataFile{}, fmt.Errorf("decode metadata trailing data: %w", err)
	}

	if len(metadata.Artifacts) == 0 {
		return metadataFile{}, errors.New("metadata contains no artifacts")
	}
	return metadata, nil
}

func generateManifest(options manifestOptions) (releaseManifest, error) {
	commit, err := normalizeCommit(options.Commit)
	if err != nil {
		return releaseManifest{}, err
	}
	if strings.TrimSpace(options.ArtifactsDir) == "" {
		return releaseManifest{}, errors.New("artifacts directory is required")
	}
	if options.ReleasedAt.IsZero() {
		return releaseManifest{}, errors.New("release timestamp is required")
	}
	if !options.Development && isDevelopmentVersion(config.Version) {
		return releaseManifest{}, fmt.Errorf(
			"production Testnet manifest cannot use development version %q",
			config.Version,
		)
	}

	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV02)
	if err != nil {
		return releaseManifest{}, fmt.Errorf("resolve Testnet profile: %w", err)
	}

	items := append([]artifactMetadata(nil), options.Metadata...)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Filename < items[j].Filename
	})

	seen := make(map[string]struct{}, len(items))
	artifacts := make([]manifestArtifact, 0, len(items))
	for _, item := range items {
		if err := validateArtifactMetadata(item); err != nil {
			return releaseManifest{}, err
		}
		if _, exists := seen[item.Filename]; exists {
			return releaseManifest{}, fmt.Errorf("duplicate artifact filename %q", item.Filename)
		}
		seen[item.Filename] = struct{}{}

		path := filepath.Join(options.ArtifactsDir, item.Filename)
		info, err := os.Lstat(path)
		if err != nil {
			return releaseManifest{}, fmt.Errorf("artifact %q: %w", item.Filename, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return releaseManifest{}, fmt.Errorf("artifact %q must not be a symlink", item.Filename)
		}
		if !info.Mode().IsRegular() {
			return releaseManifest{}, fmt.Errorf("artifact %q is not a regular file", item.Filename)
		}

		sum, err := sha256File(path)
		if err != nil {
			return releaseManifest{}, fmt.Errorf("artifact %q: %w", item.Filename, err)
		}

		artifacts = append(artifacts, manifestArtifact{
			Filename:           item.Filename,
			OS:                 item.OS,
			Arch:               item.Arch,
			MinimumOS:          item.MinimumOS,
			SizeBytes:          info.Size(),
			SHA256:             sum,
			SigningStatus:      item.SigningStatus,
			NotarizationStatus: item.NotarizationStatus,
		})
	}

	releaseKind := "testnet"
	if options.Development {
		releaseKind = "development"
	}

	return releaseManifest{
		SchemaVersion: manifestSchemaVersion,
		Product:       "VALDR Desktop",
		Version:       config.Version,
		GitCommit:     commit,
		ReleaseKind:   releaseKind,
		Development:   options.Development,
		Network:       profile.Name,
		ChainID:       profile.ChainID,
		ProtocolMin:   profile.ProtocolMin,
		ProtocolMax:   profile.ProtocolMax,
		ReleasedAt:    options.ReleasedAt.UTC().Format(time.RFC3339),
		Artifacts:     artifacts,
	}, nil
}

func validateArtifactMetadata(item artifactMetadata) error {
	if item.Filename == "" || item.Filename == "." || item.Filename == ".." ||
		strings.ContainsAny(item.Filename, "/\\") ||
		filepath.Base(item.Filename) != item.Filename {
		return fmt.Errorf("unsafe artifact filename %q", item.Filename)
	}

	expectedPrefix := "VALDR-Desktop-" + config.Version + "-"
	if !strings.HasPrefix(item.Filename, expectedPrefix) {
		return fmt.Errorf(
			"artifact %q does not match application version %q",
			item.Filename,
			config.Version,
		)
	}

	switch item.OS {
	case "windows", "macos", "linux":
	default:
		return fmt.Errorf("artifact %q has unsupported os %q", item.Filename, item.OS)
	}
	switch item.Arch {
	case "amd64", "arm64":
	default:
		return fmt.Errorf("artifact %q has unsupported arch %q", item.Filename, item.Arch)
	}

	if strings.TrimSpace(item.MinimumOS) == "" {
		return fmt.Errorf("artifact %q minimum_os is required", item.Filename)
	}
	if strings.TrimSpace(item.SigningStatus) == "" {
		return fmt.Errorf("artifact %q signing_status is required", item.Filename)
	}
	if strings.TrimSpace(item.NotarizationStatus) == "" {
		return fmt.Errorf("artifact %q notarization_status is required", item.Filename)
	}
	return nil
}

func normalizeCommit(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if len(value) != 40 {
		return "", errors.New("git commit must be exactly 40 hexadecimal characters")
	}
	if _, err := hex.DecodeString(value); err != nil {
		return "", errors.New("git commit must be exactly 40 hexadecimal characters")
	}
	return value, nil
}

func isDevelopmentVersion(version string) bool {
	version = strings.ToLower(strings.TrimSpace(version))
	return strings.HasSuffix(version, "-dev") ||
		strings.Contains(version, "-dev.") ||
		strings.Contains(version, "+dev")
}

func sha256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
