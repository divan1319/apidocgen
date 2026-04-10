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

func (g *HTMLGenerator) GenerateSections(sections []models.SectionDoc, title, outputPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer f.Close()

	tmpl, err := template.New("docs").Funcs(template.FuncMap{
		"methodColor": methodColor,
		"statusColor": statusColor,
		"lower":       strings.ToLower,
		"urlsafe": func(s string) string {
			r := strings.NewReplacer("/", "-", "{", "", "}", "")
			return strings.Trim(r.Replace(s), "-")
		},
		"slugify": func(s string) string {
			s = strings.ToLower(s)
			re := strings.NewReplacer(" ", "-", "/", "-", ".", "-")
			return re.Replace(s)
		},
	}).Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	return tmpl.Execute(f, map[string]any{
		"Title":    title,
		"Sections": sections,
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
    --get:    #22d3a5; --get-bg: rgba(34,211,165,0.08);
    --post:   #f97316; --post-bg:rgba(249,115,22,0.08);
    --put:    #eab308; --put-bg: rgba(234,179,8,0.08);
    --delete: #f43f5e; --del-bg: rgba(244,63,94,0.08);
    --success:#22d3a5; --warn:#f97316; --danger:#f43f5e;
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
    grid-template-columns: 280px 1fr;
    min-height: 100vh;
  }

  /* ── SIDEBAR ── */
  nav {
    position: sticky; top: 0; height: 100vh;
    overflow-y: auto;
    background: var(--surface);
    border-right: 1px solid var(--border);
    padding: 24px 0;
    display: flex; flex-direction: column; gap: 2px;
  }
  .nav-header {
    font-size: 13px; font-weight: 700; letter-spacing: .06em;
    color: var(--text); padding: 0 20px 20px;
    border-bottom: 1px solid var(--border); margin-bottom: 8px;
  }
  .nav-header span { color: var(--accent); }

  /* Section group in sidebar */
  .nav-section {
    margin-bottom: 4px;
  }
  .nav-section-header {
    display: flex; align-items: center; gap: 8px;
    padding: 8px 20px;
    cursor: pointer;
    user-select: none;
    color: var(--text);
    font-size: 12px; font-weight: 700;
    letter-spacing: .08em; text-transform: uppercase;
    transition: color .15s;
  }
  .nav-section-header:hover { color: var(--accent); }
  .nav-section-header .version-tag {
    font-size: 9px; padding: 1px 5px;
    background: var(--accent-dim); color: var(--accent);
    border-radius: 3px; font-family: var(--mono);
  }
  .nav-section-header .chevron {
    margin-left: auto; transition: transform .2s;
    color: var(--text-muted); font-size: 10px;
  }
  .nav-section.collapsed .chevron { transform: rotate(-90deg); }
  .nav-section-items { overflow: hidden; }
  .nav-section.collapsed .nav-section-items { display: none; }

  .nav-item {
    display: flex; align-items: center; gap: 8px;
    padding: 6px 20px 6px 28px;
    text-decoration: none; color: var(--text-muted);
    font-size: 12px; font-family: var(--mono);
    transition: color .15s, background .15s;
    border-left: 2px solid transparent;
    cursor: pointer;
  }
  .nav-item:hover { color: var(--text); background: var(--surface-2); border-left-color: var(--accent); }
  .nav-item.active { color: var(--text); border-left-color: var(--accent); background: var(--accent-dim); }
  .nav-badge {
    font-size: 9px; font-weight: 500; padding: 1px 5px;
    border-radius: 3px; flex-shrink: 0;
  }
  .nav-uri { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

  /* ── MAIN ── */
  main { padding: 48px 64px; max-width: 980px; }

  .page-header { margin-bottom: 48px; }
  .page-header h1 {
    font-size: 40px; font-weight: 800; letter-spacing: -.02em;
    background: linear-gradient(135deg, var(--text) 0%, var(--accent) 100%);
    -webkit-background-clip: text; -webkit-text-fill-color: transparent;
    background-clip: text; margin-bottom: 6px;
  }
  .page-header p { color: var(--text-muted); font-size: 14px; }

  /* ── SECTION PAGE ── */
  .section-page { display: none; }
  .section-page.active { display: block; }

  .section-title-bar {
    display: flex; align-items: center; gap: 12px;
    margin-bottom: 32px; padding-bottom: 20px;
    border-bottom: 1px solid var(--border);
  }
  .section-title-bar h2 { font-size: 24px; font-weight: 700; }
  .section-version {
    font-family: var(--mono); font-size: 11px;
    padding: 3px 8px; border-radius: 5px;
    background: var(--accent-dim); color: var(--accent);
  }
  .section-count { color: var(--text-muted); font-size: 13px; margin-left: auto; }

  /* ── ENDPOINT CARD ── */
  .endpoint {
    background: var(--surface); border: 1px solid var(--border);
    border-radius: 12px; margin-bottom: 16px; overflow: hidden;
    transition: border-color .2s;
  }
  .endpoint:hover { border-color: var(--accent); }
  .endpoint-header {
    display: flex; align-items: center; gap: 14px;
    padding: 16px 20px; cursor: pointer; user-select: none;
  }
  .method-badge {
    font-family: var(--mono); font-size: 11px; font-weight: 500;
    padding: 3px 8px; border-radius: 5px; letter-spacing: .06em; flex-shrink: 0;
  }
  .method-get    { color: var(--get);    background: var(--get-bg); }
  .method-post   { color: var(--post);   background: var(--post-bg);}
  .method-put    { color: var(--put);    background: var(--put-bg); }
  .method-delete { color: var(--delete); background: var(--del-bg); }
  .endpoint-uri  { font-family: var(--mono); font-size: 14px; color: var(--text); flex: 1; }
  .endpoint-summary { color: var(--text-muted); font-size: 12px; text-align: right; }
  .endpoint-body { border-top: 1px solid var(--border); padding: 20px; display: none; }
  .endpoint.open .endpoint-body { display: block; }
  .endpoint-description { color: var(--text-muted); font-size: 13px; line-height: 1.7; margin-bottom: 24px; }

  /* ── SECTIONS ── */
  .section { margin-bottom: 24px; }
  .section-label {
    font-size: 10px; font-weight: 700; letter-spacing: .1em;
    text-transform: uppercase; color: var(--accent); margin-bottom: 10px;
  }
  table { width: 100%; border-collapse: collapse; font-size: 13px; }
  th {
    text-align: left; padding: 7px 10px; color: var(--text-muted);
    font-weight: 500; font-size: 10px; letter-spacing: .08em;
    text-transform: uppercase; border-bottom: 1px solid var(--border);
  }
  td { padding: 9px 10px; border-bottom: 1px solid var(--border); vertical-align: top; line-height: 1.5; }
  tr:last-child td { border-bottom: none; }
  .param-name { font-family: var(--mono); color: var(--text); }
  .param-type { font-family: var(--mono); color: var(--accent); font-size: 11px; }
  .param-desc { color: var(--text-muted); }
  .badge-required { background: rgba(244,63,94,.12); color: var(--danger); font-size: 10px; padding: 2px 5px; border-radius: 3px; }
  .badge-optional { background: var(--surface-2); color: var(--text-muted); font-size: 10px; padding: 2px 5px; border-radius: 3px; }
  .response-item { background: var(--surface-2); border: 1px solid var(--border); border-radius: var(--radius); margin-bottom: 8px; overflow: hidden; }
  .response-header { display: flex; align-items: center; gap: 10px; padding: 9px 14px; }
  .status-badge { font-family: var(--mono); font-size: 13px; font-weight: 500; }
  .status-success { color: var(--success); }
  .status-client-error { color: var(--warn); }
  .status-server-error { color: var(--danger); }
  .response-desc { color: var(--text-muted); font-size: 13px; }
  pre { background: #070710; border: 1px solid var(--border); border-radius: var(--radius); padding: 14px; overflow-x: auto; font-family: var(--mono); font-size: 12px; line-height: 1.7; color: #a5b4fc; }
  .mw-tags { display: flex; gap: 6px; flex-wrap: wrap; }
  .mw-tag { background: var(--accent-dim); color: var(--accent); font-family: var(--mono); font-size: 10px; padding: 2px 7px; border-radius: 4px; }
  ::-webkit-scrollbar { width: 5px; } ::-webkit-scrollbar-track { background: transparent; } ::-webkit-scrollbar-thumb { background: var(--border); border-radius: 3px; }
</style>
</head>
<body>

<nav>
  <div class="nav-header">{{.Title}}</div>

  {{range .Sections}}
    {{$sectionName := .Name}}
  <div class="nav-section" id="nav-sec-{{slugify .Name}}">
    <div class="nav-section-header" onclick="toggleNavSection('nav-sec-{{slugify .Name}}')">
      <span>{{.Name}}</span>
      {{if .Version}}<span class="version-tag">{{.Version}}</span>{{end}}
      <span class="chevron">▼</span>
    </div>
    <div class="nav-section-items">
      {{range .Docs}}
      <div class="nav-item" onclick="showEndpoint('{{slugify $sectionName}}-{{lower .Endpoint.Method}}-{{urlsafe .Endpoint.URI}}', 'sec-{{slugify $sectionName}}')">
        <span class="nav-badge method-badge {{methodColor .Endpoint.Method}}">{{.Endpoint.Method}}</span>
        <span class="nav-uri">{{.Endpoint.URI}}</span>
      </div>
      {{end}}
    </div>
  </div>
  {{end}}
</nav>

<main>
  <div class="page-header">
    <h1>{{.Title}}</h1>
    <p>
      {{len .Sections}} section(s) ·
      {{range .Sections}}{{len .Docs}} + {{end}}0 endpoints · Generated by apidocgen
    </p>
  </div>

  {{range .Sections}}
    {{$sectionName := .Name}}
  <div class="section-page" id="sec-{{slugify .Name}}">
    <div class="section-title-bar">
      <h2>{{.Name}}</h2>
      {{if .Version}}<span class="section-version">{{.Version}}</span>{{end}}
      <span class="section-count">{{len .Docs}} endpoints</span>
    </div>

    {{range .Docs}}
    <div class="endpoint" id="{{slugify $sectionName}}-{{lower .Endpoint.Method}}-{{urlsafe .Endpoint.URI}}">
      <div class="endpoint-header" onclick="this.parentElement.classList.toggle('open')">
        <span class="method-badge {{methodColor .Endpoint.Method}}">{{.Endpoint.Method}}</span>
        <span class="endpoint-uri">{{.Endpoint.URI}}</span>
        <span class="endpoint-summary">{{.Summary}}</span>
      </div>
      <div class="endpoint-body">
        {{if .Description}}<p class="endpoint-description">{{.Description}}</p>{{end}}

        {{if .Endpoint.Middleware}}
        <div class="section">
          <div class="section-label">Middleware</div>
          <div class="mw-tags">{{range .Endpoint.Middleware}}<span class="mw-tag">{{.}}</span>{{end}}</div>
        </div>
        {{end}}

        {{if .Parameters}}
        <div class="section">
          <div class="section-label">Parameters</div>
          <table>
            <thead><tr><th>Name</th><th>Type</th><th>Required</th><th>Description</th></tr></thead>
            <tbody>
              {{range .Parameters}}
              <tr>
                <td class="param-name">{{.Name}}</td>
                <td class="param-type">{{.Type}}</td>
                <td>{{if .Required}}<span class="badge-required">required</span>{{else}}<span class="badge-optional">optional</span>{{end}}</td>
                <td class="param-desc">{{.Description}}</td>
              </tr>
              {{end}}
            </tbody>
          </table>
        </div>
        {{end}}

        {{if .Responses}}
        <div class="section">
          <div class="section-label">Responses</div>
          {{range .Responses}}
          <div class="response-item">
            <div class="response-header">
              <span class="status-badge {{statusColor .Code}}">{{.Code}}</span>
              <span class="response-desc">{{.Description}}</span>
            </div>
            {{if .Body}}<pre>{{.Body}}</pre>{{end}}
          </div>
          {{end}}
        </div>
        {{end}}

        {{if .Example.Body}}
        <div class="section">
          <div class="section-label">Request Example</div>
          <pre>{{.Example.Body}}</pre>
        </div>
        {{end}}
      </div>
    </div>
    {{end}}
  </div>
  {{end}}
</main>

<script>
  function slugify(s) {
    return s.toLowerCase().replace(/\s+/g, '-').replace(/[^a-z0-9-]/g, '');
  }

  // Show a specific section
  function showSection(sectionId) {
    document.querySelectorAll('.section-page').forEach(el => el.classList.remove('active'));
    const target = document.getElementById(sectionId);
    if (target) target.classList.add('active');
  }

  // Highlight active nav item and show endpoint
  function showEndpoint(endpointId, sectionId) {
    showSection(sectionId);
    document.querySelectorAll('.nav-item').forEach(el => el.classList.remove('active'));
    event.currentTarget.classList.add('active');
    const ep = document.getElementById(endpointId);
    if (ep) {
      ep.classList.add('open');
      ep.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
  }

  function toggleNavSection(id) {
    document.getElementById(id).classList.toggle('collapsed');
  }

  // Init: show first section
  const firstSection = document.querySelector('.section-page');
  if (firstSection) firstSection.classList.add('active');

  // Open first endpoint in first section
  const firstEndpoint = document.querySelector('.section-page.active .endpoint');
  if (firstEndpoint) firstEndpoint.classList.add('open');
</script>
</body>
</html>`
