// ALR - Any Linux Repository
// Copyright (C) 2025 Евгений Храмов
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

package dl_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/config"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/dl"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/dlcache"
)

func TestDownloadFileWithoutCache(t *testing.T) {
	type testCase struct {
		name        string
		expectedErr error
	}

	for _, tc := range []testCase{
		{
			name:        "simple download",
			expectedErr: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.URL.Path == "/file":
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("Hello, World!"))
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			tmpdir, err := os.MkdirTemp("", "test-download")
			assert.NoError(t, err)
			defer os.RemoveAll(tmpdir)

			opts := dl.Options{
				CacheDisabled: true,
				URL:           server.URL + "/file",
				Destination:   tmpdir,
			}

			err = dl.Download(context.Background(), opts)
			assert.ErrorIs(t, err, tc.expectedErr)
			_, err = os.Stat(path.Join(tmpdir, "file"))
			assert.NoError(t, err)
		})
	}
}

type TestALRConfig struct{}

func (c *TestALRConfig) GetPaths(ctx context.Context) *config.Paths {
	return &config.Paths{
		CacheDir: "/tmp",
	}
}

func TestDownloadFileWithCache(t *testing.T) {
	type testCase struct {
		name        string
		expectedErr error
	}

	for _, tc := range []testCase{
		{
			name:        "simple download",
			expectedErr: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := 0

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.URL.Path == "/file":
					called += 1
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("Hello, World!"))
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			tmpdir, err := os.MkdirTemp("", "test-download")
			assert.NoError(t, err)
			defer os.RemoveAll(tmpdir)

			cfg := &TestALRConfig{}

			opts := dl.Options{
				CacheDisabled: false,
				URL:           server.URL + "/file",
				Destination:   tmpdir,
				DlCache:       dlcache.New(cfg),
			}

			outputFile := path.Join(tmpdir, "file")

			err = dl.Download(context.Background(), opts)
			assert.ErrorIs(t, err, tc.expectedErr)
			_, err = os.Stat(outputFile)
			assert.NoError(t, err)

			err = os.Remove(outputFile)
			assert.NoError(t, err)

			err = dl.Download(context.Background(), opts)
			assert.NoError(t, err)
			assert.Equal(t, 1, called)
		})
	}
}
