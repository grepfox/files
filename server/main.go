package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var secret, pass string

const dir = "/data"
const cf = dir + "/.counts.json"

var (
	mu     sync.Mutex
	counts = map[string]int{}
)

type entry struct {
	Name, Href, Path string
	Dir              bool
	Size             int64
	Count            int
	Time             time.Time
}

var page = template.Must(template.New("").Funcs(template.FuncMap{"size": size}).Parse(`<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>files</title>
<link rel="preconnect" href="https://fonts.googleapis.com"><link rel="preconnect" href="https://fonts.gstatic.com" crossorigin><link href="https://fonts.googleapis.com/css2?family=Google+Sans+Flex:wght@400;500;700&display=swap" rel="stylesheet"><style>
:root{--bg:#0a0a0a;--fg:#eee;--mu:#a3a3a3;--bd:#1f1f1f}
*{box-sizing:border-box;margin:0}
body{background:var(--bg);color:var(--fg);font:15px/1.6 "Google Sans Flex",system-ui,sans-serif;padding:64px 24px;-webkit-text-size-adjust:100%;-webkit-font-smoothing:antialiased;-moz-osx-font-smoothing:grayscale;text-rendering:optimizeLegibility}
main{max-width:960px;margin:auto;border:1px solid var(--bd)}
header{padding:24px 32px;border-bottom:1px solid var(--bd);display:flex;justify-content:space-between;gap:16px}
h1{font-size:15px;font-weight:500;color:var(--mu);word-break:break-all}
table{width:100%;border-collapse:collapse}
th{font-weight:500;font-size:14px;text-align:left;color:var(--mu)}
th,td{padding:12px 32px 12px 0;border-bottom:1px solid var(--bd);white-space:nowrap;color:var(--mu)}
td:first-child{width:100%;white-space:normal;word-break:break-all}
th:first-child,td:first-child{padding-left:32px}
a{color:var(--fg);text-decoration:none}
a:hover{text-decoration:underline}
button{background:none;border:0;padding:0;color:var(--mu);font:inherit;cursor:pointer}
button:hover{color:var(--fg)}
.d:hover{color:#e5484d}
.c{opacity:0;margin-left:10px;vertical-align:-2px;line-height:0}
tr:hover .c,.c:focus{opacity:1}
.c.ok{color:#46a758;opacity:1}
@media(hover:none){.c{opacity:.7}}
.l{display:flex;gap:16px;padding:32px}
input{flex:1;background:none;border:1px solid var(--bd);color:var(--fg);font:inherit;padding:8px 12px;outline:0}
input:focus{border-color:var(--mu)}
footer{padding:16px 32px;color:var(--mu);display:flex;justify-content:space-between}
.t{padding:32px}
.t h2{font-size:15px;font-weight:500;margin-top:28px}
.t p,.t li{color:var(--mu)}
.t ul{padding-left:20px}
.t a{text-decoration:underline}
@media(max-width:640px){body{padding:0;font-size:14px}main{border:0}header{padding:20px 16px}th,td{padding:14px 16px 14px 0}th:first-child,td:first-child{padding-left:16px}.m{display:none}.l{padding:16px}.t{padding:16px}footer{padding:16px}}
</style></head><body><main>
<header><h1><a href="/">files</a>{{range .Crumbs}} / <a href="{{.Href}}">{{.Name}}</a>{{end}}</h1>{{if .Admin}}<a href="/_/logout">logout</a>{{end}}</header>
{{if .Privacy}}<div class="t"><p>Last updated 2 November 2026</p>
<h2>Overview</h2><p>This is a public file download site. Visitors do not need an account, and we try to collect as little as possible.</p>
<h2>What we collect</h2><ul><li>No analytics, advertising or tracking.</li><li>Each file has a total download count. It is a single number and is not linked to you.</li><li>The application does not store IP addresses and does not keep access logs.</li></ul>
<h2>Cookies</h2><p>Visitors receive no cookies. A single strictly necessary login cookie is used only by the site administrator.</p>
<h2>Third parties</h2><ul><li>Hosting: the server is provided by Hetzner Online GmbH, Germany. <a href="https://www.hetzner.com/legal/privacy-policy">Hetzner privacy policy</a>.</li><li>DNS: Cloudflare, Inc. answers lookups for this domain name. Site traffic does not pass through Cloudflare. <a href="https://www.cloudflare.com/privacypolicy/">Cloudflare privacy policy</a>.</li><li>Fonts: text is displayed with Google Fonts, loaded from Google servers, which receive your IP address and browser details. <a href="https://policies.google.com/privacy">Google privacy policy</a>.</li></ul>
<h2>Files</h2><p>Files on this site are public and provided as is. To request removal of content, contact us.</p>
<h2>Your rights</h2><p>Because we do not store personal data about visitors, there is nothing to access, correct or delete. For questions about this policy, contact us.</p></div>
{{else if .Login}}<form class="l" method="post" action="/_/login"><input type="password" name="s" placeholder="password" autofocus><button>login</button></form>{{else}}
<table><tr><th>name</th><th>size</th><th>downloads</th><th class="m">modified</th>{{if .Admin}}<th></th>{{end}}</tr>{{with .Up}}<tr><td><a href="{{.}}">..</a></td><td></td><td></td><td class="m"></td>{{if $.Admin}}<td></td>{{end}}</tr>{{end}}{{range .Files}}<tr><td><a href="{{.Href}}">{{.Name}}{{if .Dir}}/{{end}}</a>{{if not .Dir}}<button class="c" title="copy link" data-c="{{$.Base}}{{.Href}}" onclick="navigator.clipboard.writeText(this.dataset.c);this.classList.add('ok');setTimeout(()=>this.classList.remove('ok'),1200)"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg></button>{{end}}</td><td>{{if not .Dir}}{{size .Size}}{{end}}</td><td>{{if not .Dir}}{{.Count}}{{end}}</td><td class="m">{{.Time.Format "2006-01-02"}}</td>{{if $.Admin}}<td><form method="post" action="/_/rm" onsubmit="return confirm('delete '+this.p.value+'?')"><input type="hidden" name="p" value="{{.Path}}"><button class="d">delete</button></form></td>{{end}}</tr>{{else}}<tr><td>empty</td></tr>{{end}}</table>
<footer><span>{{len .Files}} items</span><a href="/_/privacy">privacy</a></footer>{{end}}
</main></body></html>`))

func size(n int64) string {
	u, f, i := []string{"B", "KB", "MB", "GB", "TB"}, float64(n), 0
	for ; f >= 1024 && i < len(u)-1; i++ {
		f /= 1024
	}
	return fmt.Sprintf("%.1f %s", f, u[i])
}

func clean(p string) (string, bool) {
	p = strings.Trim(path.Clean("/"+p), "/")
	for _, s := range strings.Split(p, "/") {
		if strings.HasPrefix(s, ".") {
			return "", false
		}
	}
	return p, true
}

func esc(p string) string {
	s := strings.Split(p, "/")
	for i := range s {
		s[i] = url.PathEscape(s[i])
	}
	return strings.Join(s, "/")
}

func proto(r *http.Request) string {
	if p := r.Header.Get("X-Forwarded-Proto"); p != "" {
		return p
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func base(r *http.Request) string {
	return proto(r) + "://" + r.Host
}

func token() string {
	h := hmac.New(sha256.New, []byte(secret+pass))
	h.Write([]byte("admin"))
	return hex.EncodeToString(h.Sum(nil))
}

func admin(r *http.Request) bool {
	c, err := r.Cookie("a")
	return pass != "" && err == nil && subtle.ConstantTimeCompare([]byte(c.Value), []byte(token())) == 1
}

func save() {
	b, _ := json.Marshal(counts)
	if os.WriteFile(cf+".tmp", b, 0644) == nil {
		os.Rename(cf+".tmp", cf)
	}
}

func cookie(w http.ResponseWriter, r *http.Request, v string, age int) {
	http.SetCookie(w, &http.Cookie{Name: "a", Value: v, Path: "/", MaxAge: age, HttpOnly: true, Secure: proto(r) == "https", SameSite: http.SameSiteStrictMode})
}

func main() {
	if b, err := os.ReadFile(cf); err == nil {
		json.Unmarshal(b, &counts)
	}
	http.HandleFunc("GET /_/privacy", func(w http.ResponseWriter, r *http.Request) {
		page.Execute(w, map[string]any{"Privacy": true, "Crumbs": []entry{{Name: "privacy", Href: "/_/privacy"}}})
	})
	http.HandleFunc("GET /_/login", func(w http.ResponseWriter, r *http.Request) {
		page.Execute(w, map[string]any{"Login": true})
	})
	http.HandleFunc("POST /_/login", func(w http.ResponseWriter, r *http.Request) {
		if pass != "" && subtle.ConstantTimeCompare([]byte(r.FormValue("s")), []byte(pass)) == 1 {
			cookie(w, r, token(), 30*86400)
			http.Redirect(w, r, "/", 303)
			return
		}
		time.Sleep(2 * time.Second)
		http.Redirect(w, r, "/_/login", 303)
	})
	http.HandleFunc("GET /_/logout", func(w http.ResponseWriter, r *http.Request) {
		cookie(w, r, "", -1)
		http.Redirect(w, r, "/", 303)
	})
	http.HandleFunc("POST /_/rm", func(w http.ResponseWriter, r *http.Request) {
		p, ok := clean(r.FormValue("p"))
		if !admin(r) || !ok || p == "" {
			http.NotFound(w, r)
			return
		}
		if err := os.RemoveAll(filepath.Join(dir, p)); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		mu.Lock()
		for k := range counts {
			if k == p || strings.HasPrefix(k, p+"/") {
				delete(counts, k)
			}
		}
		save()
		mu.Unlock()
		u := "/"
		if d := path.Dir(p); d != "." {
			u = "/" + esc(d) + "/"
		}
		http.Redirect(w, r, u, 303)
	})
	http.HandleFunc("GET /{p...}", func(w http.ResponseWriter, r *http.Request) {
		p, ok := clean(r.PathValue("p"))
		f, err := os.Open(filepath.Join(dir, p))
		if !ok || err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		i, err := f.Stat()
		if err != nil || !(i.IsDir() || i.Mode().IsRegular()) {
			http.NotFound(w, r)
			return
		}
		if !i.IsDir() {
			if rg := r.Header.Get("Range"); r.Method == "GET" && (rg == "" || strings.HasPrefix(rg, "bytes=0-")) {
				mu.Lock()
				counts[p]++
				save()
				mu.Unlock()
			}
			w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(i.Name()))
			http.ServeContent(w, r, i.Name(), i.ModTime(), f)
			return
		}
		es, _ := f.ReadDir(-1)
		l := []entry{}
		mu.Lock()
		for _, e := range es {
			ei, err := e.Info()
			if err != nil || strings.HasPrefix(e.Name(), ".") || !(ei.IsDir() || ei.Mode().IsRegular()) {
				continue
			}
			k := path.Join(p, e.Name())
			h := "/" + esc(k)
			if ei.IsDir() {
				h += "/"
			}
			l = append(l, entry{e.Name(), h, k, ei.IsDir(), ei.Size(), counts[k], ei.ModTime()})
		}
		mu.Unlock()
		sort.Slice(l, func(a, b int) bool {
			if l[a].Dir != l[b].Dir {
				return l[a].Dir
			}
			return l[a].Time.After(l[b].Time)
		})
		c, acc, up := []entry{}, "", ""
		if p != "" {
			for _, s := range strings.Split(p, "/") {
				acc = path.Join(acc, s)
				c = append(c, entry{Name: s, Href: "/" + esc(acc) + "/"})
			}
			up = "/"
			if d := path.Dir(p); d != "." {
				up = "/" + esc(d) + "/"
			}
		}
		page.Execute(w, map[string]any{"Files": l, "Base": base(r), "Crumbs": c, "Up": up, "Admin": admin(r)})
	})
	http.HandleFunc("PUT /u/{p...}", func(w http.ResponseWriter, r *http.Request) {
		p, ok := clean(r.PathValue("p"))
		if secret == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Secret")), []byte(secret)) != 1 || !ok || p == "" {
			http.NotFound(w, r)
			return
		}
		d := filepath.Join(dir, path.Dir(p))
		if err := os.MkdirAll(d, 0755); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		t, err := os.CreateTemp(d, ".up-")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer os.Remove(t.Name())
		_, err = io.Copy(t, r.Body)
		if e := t.Close(); err == nil {
			err = e
		}
		if err == nil {
			err = os.Rename(t.Name(), filepath.Join(dir, p))
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		fmt.Fprintf(w, "%s/%s\n", base(r), esc(p))
	})
	http.ListenAndServe(":8080", nil)
}
