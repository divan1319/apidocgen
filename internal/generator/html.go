package generator

import (
	"fmt"
	"html/template"
	"os"
	"strings"

	"github.com/divan1319/apidocgen/pkg/models"
)

type HTMLGenerator struct{}

func New() *HTMLGenerator { return &HTMLGenerator{} }

func (g *HTMLGenerator) Generate(docs []models.EndpointDoc, title, outputPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer f.Close()

	tmpl, err := template.New("docs").Funcs(template.FuncMap{
		"methodColor": methodColor,
		"statusColor": statusColor,
		"lower":       strings.ToLower,
	}).Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	return tmpl.Execute(f, map[string]any{
		"Title": title,
		"Docs":  docs,
	})
}

func methodColor(method string) string {
	switch strings.ToUpper(method) {
	case "GET":
		return "method-get"
	case "POST":
		return "method-post"
	case "PUT", "PATCH":
		return "method-put"
	case "DELETE":
		return "method-delete"
	default:
		return "method-default"
	}
}

func statusColor(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "status-success"
	case code >= 400 && code < 500:
		return "status-client-error"
	case code >= 500:
		return "status-server-error"
	default:
		return "status-default"
	}
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{.Title}}</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link href="https://fonts.googleapis.com/css2?family=DM+Mono:ital,wght@0,300;0,400;0,500;1,400&family=Syne:wght@400;500;600;700;800&display=swap" rel="stylesheet">
<style>
  :root {
    --bg: #0a0a0f;
    --surface: #111118;
    --surface-2: #1a1a24;
    --border: #2a2a3a;
    --text: #e8e8f0;
    --text-muted: #6b6b80;
    --accent: #7c6af7;
    --accent-dim: rgba(124,106,247,0.12);

    --get:    #22d3a5;
    --get-bg: rgba(34,211,165,0.08);
    --post:   #f97316;
    --post-bg:rgba(249,115,22,0.08);
    --put:    #eab308;
    --put-bg: rgba(234,179,8,0.08);
    --delete: #f43f5e;
    --del-bg: rgba(244,63,94,0.08);

    --success: #22d3a5;
    --warn:    #f97316;
    --danger:  #f43f5e;

    --radius: 8px;
    --mono: 'DM Mono', monospace;
    --sans: 'Syne', sans-serif;
  }

  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

  html { scroll-behavior: smooth; }

  body {
    background: var(--bg);
    color: var(--text);
    font-family: var(--sans);
    display: grid;
    grid-template-columns: 260px 1fr;
    min-height: 100vh;
  }

  /* ── SIDEBAR ── */
  nav {
    position: sticky;
    top: 0;
    height: 100vh;
    overflow-y: auto;
    background: var(--surface);
    border-right: 1px solid var(--border);
    padding: 32px 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .nav-title {
    font-size: 11px;
    font-weight: 700;
    letter-spacing: .12em;
    text-transform: uppercase;
    color: var(--text-muted);
    padding: 0 24px 16px;
    border-bottom: 1px solid var(--border);
    margin-bottom: 8px;
  }

  .nav-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 24px;
    text-decoration: none;
    color: var(--text-muted);
    font-size: 13px;
    font-family: var(--mono);
    transition: color .15s, background .15s;
    border-left: 2px solid transparent;
  }

  .nav-item:hover {
    color: var(--text);
    background: var(--surface-2);
    border-left-color: var(--accent);
  }

  .nav-badge {
    font-size: 10px;
    font-weight: 500;
    padding: 2px 6px;
    border-radius: 4px;
    flex-shrink: 0;
  }

  /* ── MAIN ── */
  main {
    padding: 64px 80px;
    max-width: 960px;
  }

  .page-header {
    margin-bottom: 64px;
  }

  .page-header h1 {
    font-size: 42px;
    font-weight: 800;
    letter-spacing: -.02em;
    background: linear-gradient(135deg, var(--text) 0%, var(--accent) 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
    margin-bottom: 8px;
  }

  .page-header p {
    color: var(--text-muted);
    font-size: 15px;
  }

  /* ── ENDPOINT CARD ── */
  .endpoint {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 12px;
    margin-bottom: 24px;
    overflow: hidden;
    transition: border-color .2s;
  }

  .endpoint:hover { border-color: var(--accent); }

  .endpoint-header {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 20px 24px;
    cursor: pointer;
    user-select: none;
  }

  .method-badge {
    font-family: var(--mono);
    font-size: 12px;
    font-weight: 500;
    padding: 4px 10px;
    border-radius: 6px;
    letter-spacing: .06em;
    flex-shrink: 0;
  }

  .method-get    { color: var(--get);    background: var(--get-bg); }
  .method-post   { color: var(--post);   background: var(--post-bg);}
  .method-put    { color: var(--put);    background: var(--put-bg); }
  .method-delete { color: var(--delete); background: var(--del-bg); }

  .endpoint-uri {
    font-family: var(--mono);
    font-size: 15px;
    color: var(--text);
    flex: 1;
  }

  .endpoint-summary {
    color: var(--text-muted);
    font-size: 13px;
  }

  .endpoint-body {
    border-top: 1px solid var(--border);
    padding: 24px;
    display: none;
  }

  .endpoint.open .endpoint-body { display: block; }

  .endpoint-description {
    color: var(--text-muted);
    font-size: 14px;
    line-height: 1.7;
    margin-bottom: 28px;
  }

  /* ── SECTION ── */
  .section {
    margin-bottom: 28px;
  }

  .section-title {
    font-size: 11px;
    font-weight: 700;
    letter-spacing: .1em;
    text-transform: uppercase;
    color: var(--accent);
    margin-bottom: 12px;
  }

  /* ── PARAMS TABLE ── */
  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
  }

  th {
    text-align: left;
    padding: 8px 12px;
    color: var(--text-muted);
    font-weight: 500;
    font-size: 11px;
    letter-spacing: .08em;
    text-transform: uppercase;
    border-bottom: 1px solid var(--border);
  }

  td {
    padding: 10px 12px;
    border-bottom: 1px solid var(--border);
    vertical-align: top;
    line-height: 1.5;
  }

  tr:last-child td { border-bottom: none; }

  .param-name { font-family: var(--mono); color: var(--text); }
  .param-type { font-family: var(--mono); color: var(--accent); font-size: 12px; }
  .param-desc { color: var(--text-muted); }

  .badge-required {
    background: rgba(244,63,94,.12);
    color: var(--danger);
    font-size: 10px;
    padding: 2px 6px;
    border-radius: 4px;
    font-weight: 500;
  }

  .badge-optional {
    background: var(--surface-2);
    color: var(--text-muted);
    font-size: 10px;
    padding: 2px 6px;
    border-radius: 4px;
    font-weight: 500;
  }

  /* ── RESPONSES ── */
  .response-item {
    background: var(--surface-2);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    margin-bottom: 10px;
    overflow: hidden;
  }

  .response-header {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 16px;
  }

  .status-badge {
    font-family: var(--mono);
    font-size: 13px;
    font-weight: 500;
  }

  .status-success     { color: var(--success); }
  .status-client-error{ color: var(--warn); }
  .status-server-error{ color: var(--danger); }
  .status-default     { color: var(--text-muted); }

  .response-desc { color: var(--text-muted); font-size: 13px; }

  /* ── CODE BLOCK ── */
  pre {
    background: #070710;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 16px;
    overflow-x: auto;
    font-family: var(--mono);
    font-size: 12.5px;
    line-height: 1.7;
    color: #a5b4fc;
  }

  .middleware-tags { display: flex; gap: 6px; flex-wrap: wrap; margin-top: 4px; }

  .mw-tag {
    background: var(--accent-dim);
    color: var(--accent);
    font-family: var(--mono);
    font-size: 11px;
    padding: 2px 8px;
    border-radius: 4px;
  }

  /* ── SCROLLBAR ── */
  ::-webkit-scrollbar { width: 6px; }
  ::-webkit-scrollbar-track { background: transparent; }
  ::-webkit-scrollbar-thumb { background: var(--border); border-radius: 3px; }
</style>
</head>
<body>

<nav>
  <div class="nav-title">{{.Title}}</div>
  {{range .Docs}}
  <a class="nav-item" href="#ep-{{lower .Endpoint.Method}}-{{.Endpoint.URI | printf "%s" | urlsafe}}">
    <span class="nav-badge method-badge {{methodColor .Endpoint.Method}}">{{.Endpoint.Method}}</span>
    <span style="overflow:hidden;text-overflow:ellipsis;white-space:nowrap;">{{.Endpoint.URI}}</span>
  </a>
  {{end}}
</nav>

<main>
  <div class="page-header">
    <h1>{{.Title}}</h1>
    <p>{{len .Docs}} endpoints documented · Generated by apidocgen</p>
  </div>

  {{range .Docs}}
  <div class="endpoint" id="ep-{{lower .Endpoint.Method}}-{{.Endpoint.URI}}" onclick="this.classList.toggle('open')">
    <div class="endpoint-header">
      <span class="method-badge {{methodColor .Endpoint.Method}}">{{.Endpoint.Method}}</span>
      <span class="endpoint-uri">{{.Endpoint.URI}}</span>
      <span class="endpoint-summary">{{.Summary}}</span>
    </div>

    <div class="endpoint-body">
      {{if .Description}}
      <p class="endpoint-description">{{.Description}}</p>
      {{end}}

      {{if .Endpoint.Middleware}}
      <div class="section">
        <div class="section-title">Middleware</div>
        <div class="middleware-tags">
          {{range .Endpoint.Middleware}}<span class="mw-tag">{{.}}</span>{{end}}
        </div>
      </div>
      {{end}}

      {{if .Parameters}}
      <div class="section">
        <div class="section-title">Parameters</div>
        <table>
          <thead>
            <tr>
              <th>Name</th>
              <th>Type</th>
              <th>Required</th>
              <th>Description</th>
            </tr>
          </thead>
          <tbody>
            {{range .Parameters}}
            <tr>
              <td class="param-name">{{.Name}}</td>
              <td class="param-type">{{.Type}}</td>
              <td>
                {{if .Required}}
                <span class="badge-required">required</span>
                {{else}}
                <span class="badge-optional">optional</span>
                {{end}}
              </td>
              <td class="param-desc">{{.Description}}</td>
            </tr>
            {{end}}
          </tbody>
        </table>
      </div>
      {{end}}

      {{if .Responses}}
      <div class="section">
        <div class="section-title">Responses</div>
        {{range .Responses}}
        <div class="response-item">
          <div class="response-header">
            <span class="status-badge {{statusColor .Code}}">{{.Code}}</span>
            <span class="response-desc">{{.Description}}</span>
          </div>
          {{if .Body}}
          <pre>{{.Body}}</pre>
          {{end}}
        </div>
        {{end}}
      </div>
      {{end}}

      {{if .Example.Body}}
      <div class="section">
        <div class="section-title">Request Example</div>
        <pre>{{.Example.Body}}</pre>
      </div>
      {{end}}
    </div>
  </div>
  {{end}}
</main>

<script>
  // Open first endpoint by default
  const first = document.querySelector('.endpoint');
  if (first) first.classList.add('open');
</script>
</body>
</html>`
