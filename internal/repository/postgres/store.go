package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"sugary/internal/config"
	reposqlc "sugary/internal/repository/postgres/sqlc"
)

type Store struct {
	Pool    *pgxpool.Pool
	Queries *reposqlc.Queries
}

func NewStore(ctx context.Context, cfg config.PostgresConfig) (*Store, error) {
	pool, err := pgxpool.New(ctx, cfg.DSN())
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return &Store{
		Pool:    pool,
		Queries: reposqlc.New(pool),
	}, nil
}

func (s *Store) Close() {
	if s == nil || s.Pool == nil {
		return
	}

	s.Pool.Close()
}
