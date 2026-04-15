package workspace

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	ErrUnresolvedVariable = errors.New("unresolved variable")
	varPattern            = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)
)

type EnvVars map[string]string

func LoadEnv(root string) (EnvVars, error) {
	file, err := os.Open(filepath.Join(root, EnvFileName))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	env := EnvVars{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("invalid env line %q", line)
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return nil, fmt.Errorf("invalid env line %q", line)
		}

		env[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return env, nil
}

func ResolvePath(path string, env EnvVars) (string, error) {
	if path == "" {
		return "", nil
	}

	var builder strings.Builder
	last := 0

	matches := varPattern.FindAllStringSubmatchIndex(path, -1)
	for _, match := range matches {
		start, end := match[0], match[1]
		nameStart, nameEnd := match[2], match[3]

		builder.WriteString(normalizeLiteralPathFragment(path[last:start]))

		name := path[nameStart:nameEnd]
		value, ok := lookupEnvValue(name, env)
		if !ok {
			return "", fmt.Errorf("%w: %s", ErrUnresolvedVariable, name)
		}
		builder.WriteString(value)

		last = end
	}

	builder.WriteString(normalizeLiteralPathFragment(path[last:]))

	return filepath.Clean(builder.String()), nil
}

func lookupEnvValue(name string, env EnvVars) (string, bool) {
	if env != nil {
		if value, ok := env[name]; ok {
			return value, true
		}
	}

	value, ok := os.LookupEnv(name)
	if ok {
		return value, ok
	}

	store, err := LoadFavoriteStore()
	if err == nil {
		if favorite, ok := store.Get(name); ok {
			return favorite.Path, true
		}
	}

	return "", false
}

func normalizeLiteralPathFragment(fragment string) string {
	if fragment == "" {
		return ""
	}

	fragment = strings.ReplaceAll(fragment, `\`, string(filepath.Separator))
	fragment = strings.ReplaceAll(fragment, `/`, string(filepath.Separator))
	return fragment
}

func SaveEnv(root string, env EnvVars) error {
	keys := make([]string, 0, len(env))
	for key := range env {
		if strings.TrimSpace(key) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(env[key])
		builder.WriteByte('\n')
	}

	return os.WriteFile(filepath.Join(root, EnvFileName), []byte(builder.String()), 0o644)
}
