package cmd

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
			g.Assert(isAZipFile(validPath)).IsTrue()
		})
		g.It(invalidPath + " should return false", func() {
			g.Assert(isAZipFile(invalidPath)).IsFalse()
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
			g.Assert(stringIsInSlice(a, slice)).IsTrue()
		})
		g.It("Should return true", func() {
			g.Assert(stringIsInSlice(b, slice)).IsTrue()
		})
		g.It("Should return true", func() {
			g.Assert(stringIsInSlice(c, slice)).IsTrue()
		})
		g.It("Should return false", func() {
			g.Assert(stringIsInSlice(d, slice)).IsFalse()
		})
	})
}