// filter_exec.go
package main

import "io"

// filterCmd implements tea.ExecCommand so we can use tea.Exec to
// suspend the bubbletea program and run a blocking huh form.
type filterCmd struct {
	fn func() error
}

func newFilterCmd(fn func() error) *filterCmd {
	return &filterCmd{fn: fn}
}

func (c *filterCmd) Run() error {
	return c.fn()
}

func (c *filterCmd) SetStdin(_ io.Reader)  {}
func (c *filterCmd) SetStdout(_ io.Writer) {}
func (c *filterCmd) SetStderr(_ io.Writer) {}
