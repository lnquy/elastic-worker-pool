// This file was originally taken from Docker project:
// https://github.com/moby/moby/blob/e50f791d42/pkg/namesgenerator/names-generator_test.go
//
// Apache License 2.0: https://github.com/moby/moby/blob/e50f791d42/LICENSE

package ewp

import (
	"strings"
	"testing"
)

func TestNameFormat(t *testing.T) {
	name := getRandomName(0)
	if !strings.Contains(name, "_") {
		t.Fatalf("Generated name does not contain an underscore")
	}
	if strings.ContainsAny(name, "0123456789") {
		t.Fatalf("Generated name contains numbers!")
	}
}

func TestNameRetries(t *testing.T) {
	name := getRandomName(1)
	if !strings.Contains(name, "_") {
		t.Fatalf("Generated name does not contain an underscore")
	}
	if !strings.ContainsAny(name, "0123456789") {
		t.Fatalf("Generated name doesn't contain a number")
	}

}
