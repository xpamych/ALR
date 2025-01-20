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

package config

import (
	"context"
	"sync"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/types"
)

// Config returns a ALR configuration struct.
// The first time it's called, it'll load the config from a file.
// Subsequent calls will just return the same value.
//
// Deprecated: use struct method
func Config(ctx context.Context) *types.Config {
	return GetInstance(ctx).cfg
}

// =======================
// FOR LEGACY ONLY
// =======================

var (
	alrConfig     *ALRConfig
	alrConfigOnce sync.Once
)

// Deprecated: For legacy only
func GetInstance(ctx context.Context) *ALRConfig {
	alrConfigOnce.Do(func() {
		alrConfig = New()
		alrConfig.Load(ctx)
	})

	return alrConfig
}
