// ALR - Any Linux Repository
// Copyright (C) 2025 The ALR Authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package stats

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type InstallationData struct {
	PackageName string `json:"packageName"`
	Version     string `json:"version,omitempty"`
	InstallType string `json:"installType"` // "install" or "upgrade"
	UserAgent   string `json:"userAgent"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

var (
	apiEndpoints = []string{
		"https://alr.plemya-x.ru/api/packages/track-install",
		"http://localhost:3001/api/packages/track-install",
	}
	userAgent = "ALR-CLI/1.0"
)

func generateFingerprint(packageName string) string {
	hostname, _ := os.Hostname()
	data := fmt.Sprintf("%s_%s_%s", hostname, packageName, time.Now().Format("2006-01-02"))
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// TrackInstallation отправляет статистику установки пакета
func TrackInstallation(ctx context.Context, packageName string, installType string) {
	// Запускаем в отдельной горутине, чтобы не блокировать основной процесс
	go func() {
		data := InstallationData{
			PackageName: packageName,
			InstallType: installType,
			UserAgent:   userAgent,
			Fingerprint: generateFingerprint(packageName),
		}

		jsonData, err := json.Marshal(data)
		if err != nil {
			return // Тихо игнорируем ошибки - статистика не критична
		}

		// Пробуем отправить запрос к разным endpoint-ам
		for _, endpoint := range apiEndpoints {
			if sendRequest(endpoint, jsonData) {
				return // Если хотя бы один запрос прошёл успешно, выходим
			}
		}
	}()
}

func sendRequest(endpoint string, data []byte) bool {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(data))
	if err != nil {
		return false
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// ShouldTrackPackage проверяет, нужно ли отслеживать установку этого пакета
func ShouldTrackPackage(packageName string) bool {
	// Отслеживаем только alr-bin
	return strings.Contains(packageName, "alr-bin")
}