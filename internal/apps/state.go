package apps

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/LynnColeArt/gpm/internal/config"
)

const stateFileName = "apps.json"

type InstallRecord struct {
	RequestedTarget string    `json:"requestedTarget"`
	Target          string    `json:"target"`
	Scope           string    `json:"scope"`
	BinDir          string    `json:"binDir"`
	BinaryName      string    `json:"binaryName"`
	BinaryPath      string    `json:"binaryPath"`
	InstalledAt     time.Time `json:"installedAt"`
}

type State struct {
	Installs []InstallRecord `json:"installs"`
}

func StatePath() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, stateFileName), nil
}

func LoadState() (*State, string, error) {
	path, err := StatePath()
	if err != nil {
		return nil, "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &State{Installs: []InstallRecord{}}, path, nil
		}
		return nil, path, fmt.Errorf("read app state: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, path, fmt.Errorf("parse app state: %w", err)
	}
	normalizeState(&state)
	return &state, path, nil
}

func SaveState(path string, state *State) error {
	if state == nil {
		return errors.New("app state is nil")
	}
	normalizeState(state)

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal app state: %w", err)
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create app state directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write app state: %w", err)
	}
	return nil
}

func BuildInstallRecord(requestedTarget, normalizedTarget, scope, binDir, cwd string, installedAt time.Time) (InstallRecord, error) {
	binaryName, err := binaryNameForTarget(normalizedTarget, cwd)
	if err != nil {
		return InstallRecord{}, err
	}

	binaryPath := filepath.Join(binDir, binaryName)
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	return InstallRecord{
		RequestedTarget: requestedTarget,
		Target:          normalizedTarget,
		Scope:           scope,
		BinDir:          binDir,
		BinaryName:      binaryName,
		BinaryPath:      binaryPath,
		InstalledAt:     installedAt.UTC(),
	}, nil
}

func UpsertInstall(state *State, record InstallRecord) {
	normalizeState(state)
	for i := range state.Installs {
		if sameInstall(state.Installs[i], record) {
			state.Installs[i] = record
			sortInstalls(state.Installs)
			return
		}
	}
	state.Installs = append(state.Installs, record)
	sortInstalls(state.Installs)
}

func List(state *State, scope string) []InstallRecord {
	normalizeState(state)
	if scope == "" || scope == "all" {
		result := append([]InstallRecord(nil), state.Installs...)
		sortInstalls(result)
		return result
	}

	result := make([]InstallRecord, 0, len(state.Installs))
	for _, record := range state.Installs {
		if record.Scope == scope {
			result = append(result, record)
		}
	}
	sortInstalls(result)
	return result
}

func RemoveInstall(state *State, scope, query string) (InstallRecord, bool) {
	normalizeState(state)
	query = strings.TrimSpace(query)
	if query == "" {
		return InstallRecord{}, false
	}

	for i, record := range state.Installs {
		if scope != "" && record.Scope != scope {
			continue
		}
		if matchesInstall(record, query) {
			removed := record
			state.Installs = append(state.Installs[:i], state.Installs[i+1:]...)
			return removed, true
		}
	}

	return InstallRecord{}, false
}

func BinaryExists(record InstallRecord) bool {
	_, err := os.Stat(record.BinaryPath)
	return err == nil
}

func normalizeState(state *State) {
	if state.Installs == nil {
		state.Installs = []InstallRecord{}
	}
	sortInstalls(state.Installs)
}

func sortInstalls(installs []InstallRecord) {
	sort.Slice(installs, func(i, j int) bool {
		if installs[i].Scope != installs[j].Scope {
			return installs[i].Scope < installs[j].Scope
		}
		if installs[i].BinaryName != installs[j].BinaryName {
			return installs[i].BinaryName < installs[j].BinaryName
		}
		return installs[i].BinaryPath < installs[j].BinaryPath
	})
}

func sameInstall(a, b InstallRecord) bool {
	return a.Scope == b.Scope && a.BinaryPath == b.BinaryPath
}

func matchesInstall(record InstallRecord, query string) bool {
	return query == record.BinaryName || query == record.RequestedTarget || query == record.Target
}

func binaryNameForTarget(target, cwd string) (string, error) {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" {
		return "", errors.New("install target is required")
	}

	if isLocalPath(trimmed) {
		resolved := trimmed
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(cwd, resolved)
		}
		resolved = filepath.Clean(resolved)
		base := filepath.Base(resolved)
		if base == "." || base == string(filepath.Separator) || base == "" {
			return "", fmt.Errorf("cannot derive binary name from local target %q", target)
		}
		return base, nil
	}

	modulePath := trimmed
	if at := strings.Index(modulePath, "@"); at >= 0 {
		modulePath = modulePath[:at]
	}
	parts := strings.Split(strings.Trim(modulePath, "/"), "/")
	if len(parts) == 0 {
		return "", fmt.Errorf("cannot derive binary name from target %q", target)
	}
	name := parts[len(parts)-1]
	if name == "" {
		return "", fmt.Errorf("cannot derive binary name from target %q", target)
	}
	return name, nil
}
