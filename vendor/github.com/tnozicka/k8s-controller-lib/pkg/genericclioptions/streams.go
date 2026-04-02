package genericclioptions

import (
	"io"
)

// IOStreams is a structure containing all standard streams.
type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}
