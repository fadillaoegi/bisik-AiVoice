package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bisik/bisik_backend/internal/domain"
)

type userRepoFake struct {
	byUsername map[string]*domain.User
	created    []*domain.User
}

func newUserRepoFake() *userRepoFake {
	return &userRepoFake{byUsername: map[string]*domain.User{}}
}

func (f *userRepoFake) FindByUsername(_ domain.Context, username string) (*domain.User, error) {
	if u, ok := f.byUsername[username]; ok {
		return u, nil
	}
	return nil, domain.ErrNotFound
}
func (f *userRepoFake) FindByID(domain.Context, string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (f *userRepoFake) Create(_ domain.Context, u *domain.User) error {
	f.created = append(f.created, u)
	f.byUsername[u.Username] = u
	return nil
}
func (f *userRepoFake) Count(domain.Context) (int, error) { return len(f.byUsername), nil }

// hasherFake sengaja trivial: yang diuji di sini alur use case,
// bukan kekuatan kriptografinya (itu diuji di infrastructure/auth).
type hasherFake struct{}

func (hasherFake) Hash(password string) (string, error) { return "hash:" + password, nil }
func (hasherFake) Verify(password, encoded string) bool { return "hash:"+password == encoded }

type signerFake struct{ issued int }

func (s *signerFake) Sign(u *domain.User) (string, time.Time, error) {
	s.issued++
	return "token-" + u.ID, time.Now().Add(time.Hour), nil
}

func (s *signerFake) VerifyToken(raw string) (*TokenClaims, error) {
	switch raw {
	case "token-petugas":
		return &TokenClaims{UserID: "petugas", Role: domain.RoleOfficer}, nil
	case "token-petugas-lain":
		return &TokenClaims{UserID: "petugas-lain", Role: domain.RoleOfficer}, nil
	case "token-supervisor":
		return &TokenClaims{UserID: "supervisor", Role: domain.RoleSupervisor}, nil
	}
	return nil, domain.ErrUnauthenticated
}

// Username tidak dikenal dan kata sandi salah HARUS menghasilkan error yang
// sama. Kalau berbeda, penyerang bisa memakai endpoint login untuk memetakan
// akun mana yang benar-benar ada.
func TestLoginTidakMembedakanAkunTidakAdaDanSandiSalah(t *testing.T) {
	users := newUserRepoFake()
	users.byUsername["petugas"] = &domain.User{
		ID: "u-1", Username: "petugas", Role: domain.RoleOfficer,
		PasswordHash: "hash:benar",
	}
	uc := NewAuthUsecase(users, hasherFake{}, &signerFake{})

	_, errTidakAda := uc.Login(context.Background(), "hantu", "apa-saja")
	_, errSandiSalah := uc.Login(context.Background(), "petugas", "salah")

	if errTidakAda == nil || errSandiSalah == nil {
		t.Fatal("login gagal seharusnya menghasilkan error")
	}
	if errTidakAda.Error() != errSandiSalah.Error() {
		t.Fatalf("pesan berbeda membocorkan akun mana yang ada:\n  %v\n  %v",
			errTidakAda, errSandiSalah)
	}
}

func TestLoginBerhasilMenerbitkanToken(t *testing.T) {
	users := newUserRepoFake()
	users.byUsername["petugas"] = &domain.User{
		ID: "u-1", Username: "petugas", Role: domain.RoleOfficer,
		PasswordHash: "hash:benar",
	}
	signer := &signerFake{}
	uc := NewAuthUsecase(users, hasherFake{}, signer)

	result, err := uc.Login(context.Background(), "  petugas  ", "benar")
	if err != nil {
		t.Fatalf("Login(): %v", err)
	}
	if result.Token != "token-u-1" || result.User.Role != domain.RoleOfficer {
		t.Fatalf("hasil = %+v", result)
	}
}

func TestSeedHanyaSaatTabelKosong(t *testing.T) {
	users := newUserRepoFake()
	uc := NewAuthUsecase(users, hasherFake{}, &signerFake{})
	accounts := []SeedAccount{
		{Username: "petugas", Role: domain.RoleOfficer, Password: "a"},
		{Username: "supervisor", Role: domain.RoleSupervisor, Password: "b"},
	}

	created, err := uc.Seed(context.Background(), accounts)
	if err != nil || created != 2 {
		t.Fatalf("seed pertama = %d, %v; mau 2, nil", created, err)
	}

	// Pemanggilan kedua tidak boleh menimpa kata sandi yang sudah dipakai.
	created, err = uc.Seed(context.Background(), accounts)
	if err != nil || created != 0 {
		t.Fatalf("seed kedua = %d, %v; mau 0, nil", created, err)
	}
}

func TestSeedMelewatiAkunTanpaKataSandi(t *testing.T) {
	users := newUserRepoFake()
	uc := NewAuthUsecase(users, hasherFake{}, &signerFake{})
	created, err := uc.Seed(context.Background(), []SeedAccount{
		{Username: "petugas", Role: domain.RoleOfficer, Password: ""},
	})
	if err != nil || created != 0 {
		t.Fatalf("akun tanpa kata sandi tidak boleh dibuat: %d, %v", created, err)
	}
}

func TestCreateUserMenambahAkunKeTabelYangSudahTerisi(t *testing.T) {
	users := newUserRepoFake()
	uc := NewAuthUsecase(users, hasherFake{}, &signerFake{})
	if _, err := uc.Seed(context.Background(), []SeedAccount{
		{Username: "petugas", Role: domain.RoleOfficer, Password: "a"},
	}); err != nil {
		t.Fatalf("Seed(): %v", err)
	}

	user, err := uc.CreateUser(context.Background(), SeedAccount{
		Username: " handoko ", Name: " Handoko ", Role: domain.RoleOfficer, Password: "rahasia-123",
	})
	if err != nil {
		t.Fatalf("CreateUser(): %v", err)
	}
	if user.Username != "handoko" || user.Name != "Handoko" || user.Role != domain.RoleOfficer {
		t.Fatalf("user = %+v, spasi harus dipangkas", user)
	}
	// Harus bisa langsung dipakai login lewat hasher yang sama.
	if _, err := uc.Login(context.Background(), "handoko", "rahasia-123"); err != nil {
		t.Fatalf("akun baru tidak bisa login: %v", err)
	}
}

func TestCreateUserMenolakInputTidakSah(t *testing.T) {
	users := newUserRepoFake()
	uc := NewAuthUsecase(users, hasherFake{}, &signerFake{})
	users.byUsername["handoko"] = &domain.User{Username: "handoko"}

	cases := []struct {
		name    string
		account SeedAccount
		want    error
	}{
		{"username dipakai", SeedAccount{Username: "handoko", Name: "H", Role: domain.RoleOfficer, Password: "12345678"}, domain.ErrConflict},
		{"sandi pendek", SeedAccount{Username: "baru", Name: "B", Role: domain.RoleOfficer, Password: "1234567"}, domain.ErrInvalidInput},
		{"role asing", SeedAccount{Username: "baru", Name: "B", Role: "nasabah", Password: "12345678"}, domain.ErrInvalidInput},
		{"nama kosong", SeedAccount{Username: "baru", Name: "  ", Role: domain.RoleOfficer, Password: "12345678"}, domain.ErrInvalidInput},
	}
	for _, c := range cases {
		if _, err := uc.CreateUser(context.Background(), c.account); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, mau %v", c.name, err, c.want)
		}
	}
	if len(users.created) != 0 {
		t.Fatalf("tidak boleh ada akun tercipta: %d", len(users.created))
	}
}

func TestSocketAuthorizerMenegakkanKepemilikanSesi(t *testing.T) {
	sessions := &fakeSessionRepo{}
	sessions.session = &domain.Session{
		ID: "sesi-1", OfficerID: "petugas", Status: domain.StatusActive,
	}
	authorizer := NewSocketAuthorizer(&signerFake{}, sessions)
	ctx := context.Background()

	role, err := authorizer.AuthorizeSocket(ctx, "token-petugas", "sesi-1")
	if err != nil || role != domain.RoleOfficer {
		t.Fatalf("pemilik sesi = %q, %v; mau officer, nil", role, err)
	}

	// Petugas lain tidak boleh menumpang sesi ini — dulu dia bisa mengaku
	// officer lewat query dan mendorong audio ke percakapan orang lain.
	if _, err := authorizer.AuthorizeSocket(ctx, "token-petugas-lain", "sesi-1"); err == nil {
		t.Fatal("petugas lain diterima sebagai pemilik sesi")
	}

	// Supervisor boleh menyimak sesi mana pun.
	role, err = authorizer.AuthorizeSocket(ctx, "token-supervisor", "sesi-1")
	if err != nil || role != domain.RoleSupervisor {
		t.Fatalf("supervisor = %q, %v; mau supervisor, nil", role, err)
	}

	if _, err := authorizer.AuthorizeSocket(ctx, "token-palsu", "sesi-1"); err == nil {
		t.Fatal("token palsu diterima")
	}
}
