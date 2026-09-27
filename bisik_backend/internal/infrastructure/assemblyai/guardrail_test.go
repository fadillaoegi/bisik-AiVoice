package assemblyai

import (
	"context"
	"testing"
)

func TestPhraseGuardMendeteksiJanjiTerlarang(t *testing.T) {
	g := NewPhraseGuard()

	cases := []struct {
		name      string
		utterance string
		wantFound bool
	}{
		{"janji terlarang apa adanya", "Pokoknya ini dijamin untung pak", true},
		{"beda kapitalisasi", "PASTI CAIR kok besok", true},
		{"kalimat wajar", "Bunganya dua persen per bulan ya pak", false},
		{"kosong", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, found, err := g.Inspect(context.Background(), tc.utterance)
			if err != nil {
				t.Fatalf("tidak mengharapkan error: %v", err)
			}
			if found != tc.wantFound {
				t.Errorf("found = %v, mau %v", found, tc.wantFound)
			}
		})
	}
}
