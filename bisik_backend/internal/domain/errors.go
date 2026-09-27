package domain

import "errors"

var (
	ErrNotFound      = errors.New("resource tidak ditemukan")
	ErrSessionEnded  = errors.New("sesi sudah berakhir")
	ErrInvalidInput  = errors.New("input tidak valid")
	ErrUpstreamAudio = errors.New("upstream audio gagal")

	// ErrUnauthenticated: tidak ada kredensial yang sah sama sekali.
	ErrUnauthenticated = errors.New("kredensial tidak sah")
	// ErrForbidden: identitasnya terbukti, tetapi tidak berhak atas sumber
	// daya ini. Dibedakan dari ErrUnauthenticated supaya klien tahu apakah
	// perlu login ulang atau memang tidak punya akses.
	ErrForbidden = errors.New("tidak berhak mengakses sumber daya ini")
)
