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

package finddeps

import (
	"context"

	"github.com/goreleaser/nfpm/v2"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/distro"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/types"
)

type ProvReqFinder interface {
	FindProvides(ctx context.Context, pkgInfo *nfpm.Info, dirs types.Directories, skiplist []string) error
	FindRequires(ctx context.Context, pkgInfo *nfpm.Info, dirs types.Directories, skiplist []string) error
}

type ProvReqService struct {
	finder ProvReqFinder
}

func New(info *distro.OSRelease, pkgFormat string) *ProvReqService {
	s := &ProvReqService{
		finder: &EmptyFindProvReq{},
	}
	if pkgFormat == "rpm" {
		switch info.ID {
		case "altlinux":
			s.finder = &ALTLinuxFindProvReq{}
		case "fedora":
			s.finder = &FedoraFindProvReq{}
		}
	}
	return s
}

func (s *ProvReqService) FindProvides(ctx context.Context, pkgInfo *nfpm.Info, dirs types.Directories, skiplist []string) error {
	return s.finder.FindProvides(ctx, pkgInfo, dirs, skiplist)
}

func (s *ProvReqService) FindRequires(ctx context.Context, pkgInfo *nfpm.Info, dirs types.Directories, skiplist []string) error {
	return s.finder.FindRequires(ctx, pkgInfo, dirs, skiplist)
}
