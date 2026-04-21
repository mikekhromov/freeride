package mtproxy

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

type Manager struct {
	ConfigPath string
}

func (m *Manager) GenerateSecret() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "dd" + strings.ToUpper(hex.EncodeToString(b)), nil
}

func (m *Manager) AddSecret(secret string) error {
	if secret == "" || m.ConfigPath == "" {
		return nil
	}
	f, err := os.OpenFile(m.ConfigPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("mtproxy config open: %w", err)
	}
	defer f.Close()
	if _, err := fmt.Fprintf(f, "%s\n", secret); err != nil {
		return err
	}
	return nil
}

// RemoveSecret удаляет строку с секретом из файла (один секрет на строку).
func (m *Manager) RemoveSecret(secret string) error {
	if secret == "" || m.ConfigPath == "" {
		return nil
	}
	data, err := os.ReadFile(m.ConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("mtproxy read config: %w", err)
	}
	var lines []string
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.EqualFold(line, secret) {
			continue
		}
		lines = append(lines, line)
	}
	if err := sc.Err(); err != nil {
		return err
	}
	out := strings.Join(lines, "\n")
	if len(lines) > 0 {
		out += "\n"
	}
	if err := os.WriteFile(m.ConfigPath, []byte(out), 0o644); err != nil {
		return fmt.Errorf("mtproxy write config: %w", err)
	}
	return nil
}
