package flying

import "io"

func nonil(err ...error) error {
	for _, err := range err {
		if err != nil {
			return err
		}
	}
	return nil
}

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
