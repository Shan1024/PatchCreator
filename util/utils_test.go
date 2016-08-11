package util

import (
	"testing"

	. "github.com/franela/goblin"
)

func TestIsAZipFile(t *testing.T) {
	g := Goblin(t)
	validPath := "/path/test.zip"
	invalidPath := "/path/test.txt"
	g.Describe("This tests whether given path points to a zip or not", func() {
		g.It(validPath + " should return true", func() {
			g.Assert(HasZipExtension(validPath)).IsTrue()
		})
		g.It(invalidPath + " should return false", func() {
			g.Assert(HasZipExtension(invalidPath)).IsFalse()
		})
	})
}

func TestStringIsInSlice(t *testing.T) {
	a := "a"
	b := "b"
	c := "c"
	d := "d"
	slice := []string{a, b, c}
	g := Goblin(t)
	g.Describe("This tests whether the given string is in a slice or not", func() {
		g.It(a + " is in the slice", func() {
			g.Assert(IsStringIsInSlice(a, slice)).IsTrue()
		})
		g.It("Should return true", func() {
			g.Assert(IsStringIsInSlice(b, slice)).IsTrue()
		})
		g.It("Should return true", func() {
			g.Assert(IsStringIsInSlice(c, slice)).IsTrue()
		})
		g.It("Should return false", func() {
			g.Assert(IsStringIsInSlice(d, slice)).IsFalse()
		})
	})
}