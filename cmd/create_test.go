//todo: add copyright notice

package cmd

import (
	"testing"
	. "github.com/franela/goblin"
)

func TestSetLogLevel(t *testing.T) {
	g := Goblin(t)
	g.Describe("This tests whether the log level is set", func() {
		//g.It("should return levels.DEBUG", func() {
		//	setLogLevel(true, false)
		//	g.Assert(logger.Level()).Equal(levels.DEBUG)
		//})
		//g.It("should return levels.TRACE", func() {
		//	setLogLevel(false, true)
		//	g.Assert(logger.Level()).Equal(levels.TRACE)
		//})
	})
}