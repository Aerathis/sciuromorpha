package sciuromorpha

import (
	"testing"
)

var se = sparseEntries([]string{"first", "second", "third"})

func TestSparseEntriesDoesContain(t *testing.T) {
	if !se.contains("second") {
		t.Fail()
	}
}

func TestSparseEntriesDoesNotContain(t *testing.T) {
	if se.contains("fourth") {
		t.Fail()
	}
}

func TestIsHidden(t *testing.T) {
	if !isHidden(".hiddenDir") {
		t.Fail()
	}
}

func TestIsNotHidden(t *testing.T) {
	if isHidden("nothiddendir") {
		t.Fail()
	}
}
