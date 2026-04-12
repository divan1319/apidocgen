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

type IndexEntry struct {
	Name     string
	Slug     string
	Title    string
	Lang     string
	HtmlFile string
}

func (g *HTMLGenerator) GenerateIndex(entries []IndexEntry, outputPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating index file: %w", err)
	}
	defer f.Close()

	tmpl, err := template.New("index").Parse(indexTemplate)
	if err != nil {
		return fmt.Errorf("parsing index template: %w", err)
	}

	return tmpl.Execute(f, map[string]any{
		"Projects": entries,
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

  /* ── HAMBURGER ── */
  .menu-toggle {
    display: none;
    position: fixed; top: 14px; left: 14px; z-index: 200;
    background: var(--surface-2); border: 1px solid var(--border);
    color: var(--text); border-radius: var(--radius);
    width: 40px; height: 40px;
    align-items: center; justify-content: center;
    cursor: pointer; font-size: 18px;
  }
  .nav-overlay {
    display: none;
    position: fixed; inset: 0; z-index: 150;
    background: rgba(0,0,0,.55);
  }
  .nav-overlay.open { display: block; }

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
  main { padding: 48px 64px; min-width: 0; }

  .page-header {
    display: flex; align-items: flex-end; gap: 32px;
    margin-bottom: 32px;
  }
  .page-header-text { flex: 1; }
  .page-header h1 {
    font-size: 40px; font-weight: 800; letter-spacing: -.02em;
    background: linear-gradient(135deg, var(--text) 0%, var(--accent) 100%);
    -webkit-background-clip: text; -webkit-text-fill-color: transparent;
    background-clip: text; margin-bottom: 6px;
  }
  .page-header p { color: var(--text-muted); font-size: 14px; }

  /* ── SEARCH BAR ── */
  .search-bar {
    position: sticky; top: 0; z-index: 10;
    background: var(--bg);
    border-bottom: 1px solid var(--border);
    padding: 14px 0 16px;
    margin-bottom: 28px;
    display: flex; align-items: center; gap: 12px; flex-wrap: wrap;
  }
  .search-input-wrap {
    flex: 1; min-width: 180px;
    display: flex; align-items: center; gap: 10px;
    background: var(--surface); border: 1px solid var(--border);
    border-radius: var(--radius); padding: 0 14px;
    transition: border-color .15s;
  }
  .search-input-wrap:focus-within { border-color: var(--accent); }
  .search-icon { color: var(--text-muted); font-size: 14px; flex-shrink: 0; }
  .search-input {
    flex: 1; background: transparent; border: none; outline: none;
    color: var(--text); font-family: var(--mono); font-size: 13px;
    padding: 10px 0;
  }
  .search-input::placeholder { color: var(--text-muted); }
  .search-clear {
    background: none; border: none; color: var(--text-muted);
    cursor: pointer; font-size: 16px; padding: 0; line-height: 1;
    display: none;
  }
  .search-clear.visible { display: block; }
  .search-clear:hover { color: var(--text); }
  .method-filters { display: flex; gap: 6px; flex-wrap: wrap; }
  .mf-btn {
    font-family: var(--mono); font-size: 10px; font-weight: 500;
    padding: 5px 10px; border-radius: 5px; letter-spacing: .06em;
    cursor: pointer; border: 1px solid transparent;
    transition: opacity .15s, border-color .15s;
    opacity: .45;
  }
  .mf-btn.active { opacity: 1; border-color: currentColor; }
  .mf-btn.mf-all { color: var(--text); background: var(--surface-2); border-color: var(--border); opacity: 1; }
  .mf-btn.mf-all.active { border-color: var(--accent); color: var(--accent); }
  .mf-btn.mf-get  { color: var(--get);    background: var(--get-bg); }
  .mf-btn.mf-post { color: var(--post);   background: var(--post-bg); }
  .mf-btn.mf-put  { color: var(--put);    background: var(--put-bg); }
  .mf-btn.mf-del  { color: var(--delete); background: var(--del-bg); }
  .no-results {
    display: none; text-align: center; padding: 48px 0;
    color: var(--text-muted); font-size: 14px;
  }
  .no-results.visible { display: block; }

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
  .table-wrap { overflow-x: auto; -webkit-overflow-scrolling: touch; }
  table { width: 100%; border-collapse: collapse; font-size: 13px; min-width: 480px; }
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

  /* ── RESPONSIVE ── */
  @media (max-width: 1024px) {
    body { grid-template-columns: 240px 1fr; }
    main { padding: 40px 32px; }
  }

  @media (max-width: 768px) {
    body { grid-template-columns: 1fr; }
    .menu-toggle { display: flex; }
    nav {
      position: fixed; top: 0; left: 0; z-index: 160;
      width: 280px; height: 100vh;
      transform: translateX(-100%);
      transition: transform .25s ease;
    }
    nav.open { transform: translateX(0); }
    .nav-header { padding-left: 60px; }
    main { padding: 64px 20px 40px; }
    .page-header { flex-direction: column; align-items: flex-start; gap: 16px; }
    .page-header h1 { font-size: 28px; }
    .search-bar { flex-direction: column; align-items: stretch; gap: 10px; }
    .method-filters { justify-content: flex-start; }
    .endpoint-header { flex-wrap: wrap; gap: 8px; }
    .endpoint-summary { text-align: left; width: 100%; }
    .section-title-bar { flex-wrap: wrap; gap: 8px; }
    .section-count { margin-left: 0; }
  }

  @media (max-width: 480px) {
    main { padding: 56px 12px 32px; }
    .page-header h1 { font-size: 22px; }
    .endpoint-uri { font-size: 12px; }
    pre { font-size: 11px; padding: 10px; }
  }
</style>
</head>
<body>

<button class="menu-toggle" onclick="toggleNav()" aria-label="Toggle menu">☰</button>
<div class="nav-overlay" id="nav-overlay" onclick="closeNav()"></div>

<nav id="sidebar">
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
    <div class="page-header-text">
      <h1>{{.Title}}</h1>
      <p>
        {{len .Sections}} section(s) ·
        {{range .Sections}}{{len .Docs}} + {{end}}0 endpoints · Generated by apidocgen
      </p>
    </div>
  </div>

  <div class="search-bar">
    <div class="search-input-wrap">
      <span class="search-icon">⌕</span>
      <input class="search-input" id="search-input" type="search" placeholder="Buscar endpoint…" oninput="filterEndpoints()" autocomplete="off">
      <button class="search-clear" id="search-clear" onclick="clearSearch()">×</button>
    </div>
    <div class="method-filters">
      <button class="mf-btn mf-all active" data-method="ALL" onclick="setMethodFilter(this)">ALL</button>
      <button class="mf-btn mf-get"  data-method="GET"    onclick="setMethodFilter(this)">GET</button>
      <button class="mf-btn mf-post" data-method="POST"   onclick="setMethodFilter(this)">POST</button>
      <button class="mf-btn mf-put"  data-method="PUT"    onclick="setMethodFilter(this)">PUT</button>
      <button class="mf-btn mf-del"  data-method="DELETE" onclick="setMethodFilter(this)">DELETE</button>
    </div>
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
          <div class="table-wrap">
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
  <div class="no-results" id="no-results">
    No se encontraron endpoints que coincidan con la búsqueda.
  </div>
</main>

<script>
  function slugify(s) {
    return s.toLowerCase().replace(/\s+/g, '-').replace(/[^a-z0-9-]/g, '');
  }

  function toggleNav() {
    document.getElementById('sidebar').classList.toggle('open');
    document.getElementById('nav-overlay').classList.toggle('open');
  }
  function closeNav() {
    document.getElementById('sidebar').classList.remove('open');
    document.getElementById('nav-overlay').classList.remove('open');
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
    closeNav();
  }

  function toggleNavSection(id) {
    document.getElementById(id).classList.toggle('collapsed');
  }

  // ── SEARCH & FILTER ──
  let activeMethod = 'ALL';

  function setMethodFilter(btn) {
    activeMethod = btn.dataset.method;
    document.querySelectorAll('.mf-btn').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
    filterEndpoints();
  }

  function clearSearch() {
    document.getElementById('search-input').value = '';
    document.getElementById('search-clear').classList.remove('visible');
    filterEndpoints();
  }

  function filterEndpoints() {
    const query = document.getElementById('search-input').value.trim().toLowerCase();
    const clearBtn = document.getElementById('search-clear');
    clearBtn.classList.toggle('visible', query.length > 0);

    const isSearching = query.length > 0 || activeMethod !== 'ALL';
    let anyVisible = false;

    document.querySelectorAll('.section-page').forEach(section => {
      let sectionHasVisible = false;
      section.querySelectorAll('.endpoint').forEach(ep => {
        const uri     = (ep.querySelector('.endpoint-uri')?.textContent || '').toLowerCase();
        const summary = (ep.querySelector('.endpoint-summary')?.textContent || '').toLowerCase();
        const method  = (ep.querySelector('.method-badge')?.textContent || '').trim().toUpperCase();
        const matchText   = !query || uri.includes(query) || summary.includes(query);
        const matchMethod = activeMethod === 'ALL' || method === activeMethod;
        const visible = matchText && matchMethod;
        ep.style.display = visible ? '' : 'none';
        if (visible) { sectionHasVisible = true; anyVisible = true; }
      });
      section.style.display = (!isSearching || sectionHasVisible) ? '' : 'none';
      if (isSearching && sectionHasVisible) section.classList.add('active');
    });

    if (isSearching) {
      document.querySelectorAll('.section-page').forEach(s => s.classList.remove('active'));
      document.querySelectorAll('.section-page').forEach(s => {
        if (s.style.display !== 'none') s.classList.add('active');
      });
    } else {
      const first = document.querySelector('.section-page');
      document.querySelectorAll('.section-page').forEach(s => {
        s.classList.remove('active');
        s.style.display = '';
        s.querySelectorAll('.endpoint').forEach(ep => ep.style.display = '');
      });
      if (first) first.classList.add('active');
    }

    document.getElementById('no-results').classList.toggle('visible', isSearching && !anyVisible);
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

const indexTemplate = `<!DOCTYPE html>
<html lang="es">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>API Documentation Hub</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link href="https://fonts.googleapis.com/css2?family=DM+Mono:wght@400;500&family=Syne:wght@400;500;600;700;800&display=swap" rel="stylesheet">
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
    --radius: 12px;
    --mono: 'DM Mono', monospace;
    --sans: 'Syne', sans-serif;
  }
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    background: var(--bg);
    color: var(--text);
    font-family: var(--sans);
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 80px 24px;
  }
  .header {
    text-align: center;
    margin-bottom: 56px;
  }
  .header h1 {
    font-size: 48px;
    font-weight: 800;
    letter-spacing: -.03em;
    background: linear-gradient(135deg, var(--text) 0%, var(--accent) 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
    margin-bottom: 12px;
  }
  .header p {
    color: var(--text-muted);
    font-size: 15px;
  }
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
    gap: 20px;
    width: 100%;
    max-width: 960px;
  }
  .card {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 28px;
    text-decoration: none;
    color: var(--text);
    transition: border-color .2s, transform .2s;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .card:hover {
    border-color: var(--accent);
    transform: translateY(-2px);
  }
  .card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  .card-title {
    font-size: 20px;
    font-weight: 700;
  }
  .card-lang {
    font-family: var(--mono);
    font-size: 10px;
    font-weight: 500;
    padding: 3px 10px;
    border-radius: 6px;
    background: var(--accent-dim);
    color: var(--accent);
    letter-spacing: .06em;
    text-transform: uppercase;
  }
  .card-desc {
    color: var(--text-muted);
    font-size: 13px;
    line-height: 1.5;
  }
  .card-slug {
    font-family: var(--mono);
    font-size: 11px;
    color: var(--text-muted);
  }
  .card-arrow {
    color: var(--accent);
    font-size: 14px;
    font-weight: 600;
    align-self: flex-end;
    opacity: 0;
    transition: opacity .2s;
  }
  .card:hover .card-arrow { opacity: 1; }
  .empty {
    text-align: center;
    color: var(--text-muted);
    font-size: 14px;
    padding: 48px 0;
  }
  @media (max-width: 480px) {
    body { padding: 40px 16px; }
    .header h1 { font-size: 32px; }
    .grid { grid-template-columns: 1fr; }
  }
</style>
</head>
<body>
  <div class="header">
    <h1>API Documentation</h1>
    <p>{{len .Projects}} proyecto(s) documentado(s)</p>
  </div>

  {{if .Projects}}
  <div class="grid">
    {{range .Projects}}
    <a class="card" href="{{.HtmlFile}}">
      <div class="card-header">
        <span class="card-title">{{.Name}}</span>
        <span class="card-lang">{{.Lang}}</span>
      </div>
      <span class="card-desc">{{.Title}}</span>
      <span class="card-slug">{{.Slug}}</span>
      <span class="card-arrow">→ Ver documentación</span>
    </a>
    {{end}}
  </div>
  {{else}}
  <div class="empty">
    No hay documentaciones generadas todavía.<br>
    Ejecuta <code>apidocgen generate</code> para crear una.
  </div>
  {{end}}
</body>
</html>`
