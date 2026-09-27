package postgres

import (
	"errors"

	"github.com/bisik/bisik_backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct{ pool *pgxpool.Pool }

func NewUserRepo(p *pgxpool.Pool) *UserRepo { return &UserRepo{pool: p} }

const userColumns = `id, username, name, role, password_hash, created_at`

func (r *UserRepo) FindByUsername(ctx domain.Context, username string) (*domain.User, error) {
	return r.findBy(ctx, `WHERE lower(username) = lower($1)`, username)
}

func (r *UserRepo) FindByID(ctx domain.Context, id string) (*domain.User, error) {
	return r.findBy(ctx, `WHERE id = $1`, id)
}

func (r *UserRepo) findBy(ctx domain.Context, where string, arg any) (*domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users `+where, arg).
		Scan(&u.ID, &u.Username, &u.Name, &u.Role, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) Create(ctx domain.Context, u *domain.User) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO users (id, username, name, role, password_hash)
		VALUES ($1,$2,$3,$4,$5)`,
		u.ID, u.Username, u.Name, u.Role, u.PasswordHash)
	return err
}

func (r *UserRepo) Count(ctx domain.Context) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&n)
	return n, err
}
