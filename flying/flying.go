package flying

import (
	"io"
	"strings"
)

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
func nopCloser(writer io.Writer) io.WriteCloser {
	return nopc{writer}
}

func uniquekv(kv *[]string) func(string) {
	unique := make(map[string]int)
	return func(s string) {
		// http://blogs.msdn.com/b/oldnewthing/archive/2010/05/06/10008132.aspx
		i := strings.Index(s, "=")
		if i == 0 {
			*kv = append(*kv, s)
			return
		}
		if i != -1 {
			if n, ok := unique[s[:i]]; ok {
				(*kv)[n] = s
			} else {
				unique[s[:i]] = len(*kv)
				*kv = append(*kv, s)
			}
		}
	}
}

func mergenv(base []string, envs ...string) []string {
	merged := make([]string, 0, len(base)+len(envs))
	add := uniquekv(&merged)
	for _, env := range base {
		add(env)
	}
	for _, env := range envs {
		add(env)
	}
	return merged
}
