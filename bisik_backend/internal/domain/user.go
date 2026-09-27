package domain

import "time"

// Role menentukan apa yang boleh dilakukan seseorang terhadap sebuah sesi.
//
// Nasabah sengaja TIDAK ada di daftar ini: dia tidak pernah memakai aplikasi,
// tidak memegang perangkat, dan tidak mendengar bisikan. Yang dia butuhkan
// adalah persetujuan, bukan akun.
type Role string

const (
	RoleOfficer    Role = "officer"
	RoleSupervisor Role = "supervisor"
)

func (r Role) Valid() bool { return r == RoleOfficer || r == RoleSupervisor }

// User adalah petugas atau supervisor yang masuk ke aplikasi.
type User struct {
	ID           string
	Username     string
	Name         string
	Role         Role
	PasswordHash string
	CreatedAt    time.Time
}

type UserRepository interface {
	FindByUsername(ctx Context, username string) (*User, error)
	FindByID(ctx Context, id string) (*User, error)
	Create(ctx Context, u *User) error
	Count(ctx Context) (int, error)
}
