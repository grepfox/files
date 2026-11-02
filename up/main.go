package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var secret, host string

var client = &http.Client{Transport: &http.Transport{Proxy: http.ProxyFromEnvironment, TLSNextProto: map[string]func(string, *tls.Conn) http.RoundTripper{}, WriteBufferSize: 4 << 20, ReadBufferSize: 64 << 10}}

type prog struct {
	r    io.Reader
	n, d int64
}

func (p *prog) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.d += int64(n)
	fmt.Fprintf(os.Stderr, "\r%5.1f%%", float64(p.d)*100/float64(max(p.n, 1)))
	return n, err
}

func fail(v ...any) {
	fmt.Fprintln(os.Stderr, v...)
	os.Exit(1)
}

func send(local, remote string) {
	f, err := os.Open(local)
	if err != nil {
		fail(err)
	}
	defer f.Close()
	i, _ := f.Stat()
	fmt.Fprintln(os.Stderr, remote)
	s := strings.Split(remote, "/")
	for k := range s {
		s[k] = url.PathEscape(s[k])
	}
	req, _ := http.NewRequest("PUT", strings.TrimRight(host, "/")+"/u/"+strings.Join(s, "/"), &prog{r: f, n: i.Size()})
	req.ContentLength = i.Size()
	req.Header.Set("X-Secret", secret)
	res, err := client.Do(req)
	if err != nil {
		fail("\n", err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	fmt.Fprint(os.Stderr, "\n")
	if res.StatusCode != 200 {
		fail("failed:", res.Status)
	}
	fmt.Print(string(b))
}

func main() {
	d, args := "", []string{}
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "-d" && i+1 < len(os.Args) {
			i++
			d = os.Args[i]
		} else {
			args = append(args, os.Args[i])
		}
	}
	if len(args) == 0 {
		fail("usage: upload <file or folder>... [-d remote/folder]")
	}
	for _, a := range args {
		a = filepath.Clean(a)
		root := filepath.Dir(a)
		err := filepath.WalkDir(a, func(p string, e fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if p != a && strings.HasPrefix(e.Name(), ".") {
				if e.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if e.Type().IsRegular() {
				r, _ := filepath.Rel(root, p)
				send(p, path.Join(d, filepath.ToSlash(r)))
			}
			return nil
		})
		if err != nil {
			fail(err)
		}
	}
}
