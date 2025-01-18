package repos

import (
	"context"

	"plemya-x.ru/alr/internal/config"
	database "plemya-x.ru/alr/internal/db"
	"plemya-x.ru/alr/internal/types"
)

type Config interface {
	GetPaths(ctx context.Context) *config.Paths
	Repos(ctx context.Context) []types.Repo
}

type Repos struct {
	cfg Config
	db  *database.Database
}

func New(
	cfg Config,
	db *database.Database,
) *Repos {
	return &Repos{
		cfg,
		db,
	}
}
