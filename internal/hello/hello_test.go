package hello_test

import (
	"testing"

	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/hello"
)

func TestSum(t *testing.T) {
	if got := hello.Sum(2, 3); got != 5 {
		t.Fatalf("got %d want 5", got)
	}
}
