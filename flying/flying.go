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

type multic []io.Closer

func (c multic) Close() (err error) {
	for _, c := range c {
		if e := c.Close(); e != nil && err == nil {
			err = e
		}
	}
	return
}

// MultiCloser TODO
func MultiCloser(closers ...io.Closer) io.Closer {
	return multic(closers)
}

type multiwc struct {
	io.Writer
	io.Closer
}

// MultiWriteCloser TODO
func MultiWriteCloser(writers ...io.Writer) io.WriteCloser {
	w := multiwc{Writer: io.MultiWriter(writers...)}
	c := make([]io.Closer, 0, len(writers))
	for _, writer := range writers {
		if closer, ok := writer.(io.Closer); ok {
			c = append(c, closer)
		} else {
			c = append(c, NopCloser(nil))
		}
	}
	w.Closer = MultiCloser(c...)
	return w
}
