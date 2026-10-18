package library

import (
	"math"
	"strings"
	"testing"
)

func TestRatingBounds(t *testing.T) {
	for _, score := range []float64{0, 0.5, 10} {
		if err := ValidateRating(score, strings.Repeat("á", 2000)); err != nil {
			t.Fatal("limite válido rejeitado", score, err)
		}
	}
	for _, score := range []float64{-0.5, 10.5, 7.2, math.NaN(), math.Inf(1)} {
		if err := ValidateRating(score, ""); err != ErrInvalid {
			t.Fatal("nota inválida aceita", score)
		}
	}
	if err := ValidateRating(5, strings.Repeat("á", 2001)); err != ErrInvalid {
		t.Fatal("comentário excedente aceito")
	}
}
