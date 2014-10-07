package flying

import "io"

type nopc struct {
	io.Writer
}

func (nopc) Close() error {
	return nil
}

// NopCloser TODO
func NopCloser(writer io.Writer) io.WriteCloser {
	return nopc{writer}
}
