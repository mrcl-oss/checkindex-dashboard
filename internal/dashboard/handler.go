package dashboard

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const dashboardPage = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <noscript><meta http-equiv="refresh" content="{{.RefreshInterval}}"></noscript>
  <title>Index Usage Dashboard</title>
  <style>
    :root {
      color-scheme: light;
      font-family: system-ui, sans-serif;
      color: #172033;
      background: #f4f6fa;
      --filter-bar-height: 80px;
    }
    html {
      scroll-padding-top: calc(var(--filter-bar-height) + 1rem);
    }
    body { margin: 0; padding: 0; width: 100%; min-height: 100vh; }
    .filter-bar { position: sticky; top: 0; z-index: 100; background: #ffffff; border-bottom: 1px solid #d8deea; box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05); padding: 0.5rem 0.75rem; width: 100%; box-sizing: border-box; }
    .filter-container { width: 100%; display: flex; flex-direction: column; gap: .4rem; }
    .filter-top-row { display: flex; align-items: center; gap: .75rem; }
    .filter-label { font-weight: 650; font-size: .95rem; color: #34415a; white-space: nowrap; }
    .filter-input { flex: 1; padding: .5rem .75rem; font-size: .95rem; border: 1px solid #cbd5e1; border-radius: .375rem; background: #f8fafc; color: #172033; }
    .filter-input:focus { outline: 2px solid #2b6cb0; border-color: #2b6cb0; background: #fff; }
    .filter-bottom-row { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: .75rem; font-size: .85rem; color: #475569; }
    .filter-toggles { display: flex; align-items: center; gap: 1rem; }
    .filter-toggle-label { display: inline-flex; align-items: center; gap: .35rem; cursor: pointer; user-select: none; }
    .refresh-box { display: inline-flex; align-items: center; gap: .5rem; }
    .refresh-select { padding: .25rem .5rem; font-size: .85rem; border: 1px solid #cbd5e1; border-radius: .3rem; background: #fff; color: #172033; }
    .live-pill { display: inline-flex; align-items: center; gap: .35rem; font-size: .75rem; font-weight: 600; color: #166534; background: #dcfce7; padding: .15rem .55rem; border-radius: 9999px; }
    .live-dot { width: 7px; height: 7px; background-color: #16a34a; border-radius: 50%; }
    .btn-config { display: inline-flex; align-items: center; gap: .35rem; padding: .25rem .65rem; font-size: .85rem; font-weight: 600; color: #1e293b; background: #f1f5f9; border: 1px solid #cbd5e1; border-radius: .3rem; cursor: pointer; transition: background .15s ease, border-color .15s ease; }
    .btn-config:hover { background: #e2e8f0; border-color: #94a3b8; }
    .btn-download { display: inline-flex; align-items: center; padding: .25rem .65rem; font-size: .85rem; font-weight: 600; color: #1e293b; background: #fff; border: 1px solid #cbd5e1; border-radius: .3rem; text-decoration: none; }
    .btn-download:hover { background: #f1f5f9; border-color: #94a3b8; }
    .modal-backdrop { position: fixed; inset: 0; background: rgba(15, 23, 42, 0.65); backdrop-filter: blur(2px); z-index: 1000; display: flex; align-items: center; justify-content: center; padding: 1rem; box-sizing: border-box; }
    .modal-backdrop[hidden] { display: none !important; }
    .modal-dialog { background: #ffffff; border-radius: .75rem; box-shadow: 0 20px 25px -5px rgba(0,0,0,0.2), 0 8px 10px -6px rgba(0,0,0,0.1); width: 100%; max-width: 820px; max-height: 90vh; display: flex; flex-direction: column; overflow: hidden; border: 1px solid #cbd5e1; }
    .modal-header { padding: .85rem 1.25rem; border-bottom: 1px solid #e2e8f0; display: flex; align-items: center; justify-content: space-between; background: #f8fafc; }
    .modal-header h2 { margin: 0; font-size: 1.15rem; color: #0f172a; display: flex; align-items: center; gap: .5rem; }
    .modal-close-btn { background: transparent; border: none; font-size: 1.25rem; line-height: 1; color: #64748b; cursor: pointer; padding: .25rem .5rem; border-radius: .25rem; }
    .modal-close-btn:hover { color: #0f172a; background: #e2e8f0; }
    .modal-body { padding: 1.25rem; overflow-y: auto; flex: 1; display: flex; flex-direction: column; gap: 1rem; }
    .modal-footer { padding: .85rem 1.25rem; border-top: 1px solid #e2e8f0; display: flex; align-items: center; justify-content: flex-end; gap: .75rem; background: #f8fafc; }
    .cfg-tabs { display: flex; gap: .5rem; border-bottom: 1px solid #e2e8f0; margin-bottom: .5rem; }
    .cfg-tab-btn { background: transparent; border: none; border-bottom: 2px solid transparent; padding: .5rem .75rem; font-size: .9rem; font-weight: 600; color: #64748b; cursor: pointer; }
    .cfg-tab-btn.active { color: #2563eb; border-bottom-color: #2563eb; }
    .cfg-section { background: #f8fafc; border: 1px solid #e2e8f0; border-radius: .5rem; padding: 1rem; }
    .cfg-section-title { font-size: .95rem; font-weight: 700; color: #1e293b; margin: 0 0 .75rem 0; }
    .cfg-grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: .75rem; }
    .cfg-group { display: flex; flex-direction: column; gap: .25rem; }
    .cfg-label { font-size: .8rem; font-weight: 600; color: #334155; }
    .cfg-input { padding: .45rem .6rem; font-size: .85rem; border: 1px solid #cbd5e1; border-radius: .35rem; background: #fff; color: #0f172a; width: 100%; box-sizing: border-box; }
    .cfg-input:focus { outline: 2px solid #2563eb; border-color: #2563eb; }
    .cfg-row-space { display: flex; align-items: center; justify-content: space-between; gap: .5rem; }
    .cfg-btn-sm { padding: .25rem .5rem; font-size: .8rem; font-weight: 600; border-radius: .3rem; border: 1px solid #cbd5e1; background: #fff; cursor: pointer; color: #334155; }
    .cfg-btn-sm:hover { background: #f1f5f9; }
    .cfg-btn-primary { background: #2563eb; color: #fff; border: 1px solid #1d4ed8; padding: .45rem 1rem; font-size: .85rem; font-weight: 600; border-radius: .35rem; cursor: pointer; }
    .cfg-btn-primary:hover { background: #1d4ed8; }
    .cfg-btn-secondary { background: #f1f5f9; color: #475569; border: 1px solid #cbd5e1; padding: .45rem 1rem; font-size: .85rem; font-weight: 600; border-radius: .35rem; cursor: pointer; }
    .cfg-btn-secondary:hover { background: #e2e8f0; }
    .cfg-btn-danger { background: #fff; color: #dc2626; border: 1px solid #fecaca; }
    .cfg-btn-danger:hover { background: #fee2e2; }
    .cfg-profile-card { background: #fff; border: 1px solid #cbd5e1; border-radius: .5rem; padding: 1rem; margin-bottom: .75rem; box-shadow: 0 1px 2px rgba(0,0,0,0.03); }
    .cfg-profile-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: .75rem; padding-bottom: .5rem; border-bottom: 1px solid #f1f5f9; }
    .cfg-profile-name-title { font-weight: 700; font-size: .95rem; color: #1e293b; display: flex; align-items: center; gap: .5rem; }
    .cfg-badge { display: inline-block; font-size: .7rem; font-weight: 600; padding: .1rem .4rem; border-radius: 9999px; }
    .cfg-badge-success { background: #dcfce7; color: #166534; }
    .cfg-badge-muted { background: #f1f5f9; color: #64748b; }
    .cfg-mappings-list { display: flex; flex-direction: column; gap: .35rem; margin-top: .35rem; }
    .cfg-mapping-row { display: grid; grid-template-columns: 1fr 1fr auto; gap: .5rem; align-items: center; }
    .cfg-json-textarea { width: 100%; height: 360px; font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; font-size: .85rem; line-height: 1.4; padding: .75rem; border: 1px solid #cbd5e1; border-radius: .375rem; background: #0f172a; color: #f8fafc; box-sizing: border-box; tab-size: 2; resize: vertical; }
    .cfg-alert { padding: .75rem 1rem; border-radius: .375rem; font-size: .85rem; font-weight: 500; }
    .cfg-alert-error { background: #fee2e2; color: #991b1b; border: 1px solid #fca5a5; }
    .cfg-alert-success { background: #dcfce7; color: #166534; border: 1px solid #86efac; }
    .cfg-help-text { font-size: .8rem; color: #64748b; margin: 0 0 .5rem 0; }
    .cfg-input-pass-wrap { position: relative; display: flex; align-items: center; }
    .cfg-btn-toggle-pass { position: absolute; right: .4rem; background: none; border: none; font-size: .75rem; color: #64748b; cursor: pointer; padding: .2rem .4rem; }
    header { width: 100%; padding: 0.85rem 0.75rem 0.25rem; box-sizing: border-box; scroll-margin-top: calc(var(--filter-bar-height) + 1rem); }
    h1, h2, h3, h4 { line-height: 1.2; }
    h1 { margin: 0; }
    header p { color: #526078; margin-top: .25rem; }
    .layout-wrap { width: 100%; padding: 0.25rem 0.75rem 2rem; display: flex; gap: 1rem; align-items: flex-start; box-sizing: border-box; }
    .sidebar { width: max-content; min-width: 280px; max-width: min(520px, 45vw); flex-shrink: 0; position: sticky; top: calc(var(--filter-bar-height) + 0.75rem); max-height: calc(100vh - var(--filter-bar-height) - 1.5rem); overflow-y: auto; overflow-x: auto; background: #fff; border: 1px solid #d8deea; border-radius: .6rem; padding: 0.75rem; box-sizing: border-box; resize: horizontal; }
    .sidebar h2 { font-size: 1.05rem; margin-top: 0; margin-bottom: .5rem; color: #1e293b; border-bottom: 1px solid #e2e8f0; padding-bottom: .4rem; }
    .toc-toolbar { display: flex; flex-direction: column; gap: .35rem; margin-bottom: .6rem; padding-bottom: .5rem; border-bottom: 1px solid #e2e8f0; }
    .toc-toolbar .btn-toggle-level { width: 100%; text-align: center; }
    .toc-sub-toolbar { padding: .2rem 0; margin-bottom: .3rem; }
    .toc-sub-toolbar .btn-toggle-level { font-size: .75rem; padding: .15rem .4rem; }
    .sidebar-toc details { margin: .35rem 0; }
    .sidebar-toc summary { cursor: pointer; user-select: none; font-size: .9rem; }
    .sidebar-toc a { text-decoration: none; color: inherit; word-break: break-word; overflow-wrap: anywhere; }
    .sidebar-toc a:hover { text-decoration: underline; }
    .toc-cluster > summary { color: #1d4ed8; font-weight: 600; }
    .toc-cluster-body { padding-left: .5rem; }
    .toc-collection { margin-left: .25rem !important; }
    .toc-collection > summary { color: #6d28d9; font-weight: 550; }
    .toc-index-wrap { padding-left: 1rem; display: flex; flex-direction: column; gap: .25rem; margin-top: .25rem; }
    .toc-index-link { color: #0f766e; font-size: .85rem; padding: .15rem 0; line-height: 1.35; white-space: normal; word-break: break-word; overflow-wrap: anywhere; }
    .toc-empty { color: #64748b; font-size: .85rem; margin: 0; }
    .main-column { flex: 1; min-width: 0; width: calc(100% - 300px); scroll-margin-top: calc(var(--filter-bar-height) + 1rem); }
    section, .index { margin: 1rem 0; }
    .freshness, .empty-state { background: #fff; border: 1px solid #d8deea; border-radius: .6rem; padding: 1rem; scroll-margin-top: calc(var(--filter-bar-height) + 1rem); }
    .freshness h2, .empty-state h2 { margin-top: 0; }
    dl { display: grid; grid-template-columns: max-content 1fr; gap: .5rem 1rem; margin: 0; }
    dt { font-weight: 650; }
    dd { margin: 0; color: #34415a; }
    .banner { border-left: .3rem solid; border-radius: .25rem; margin: .75rem 0; padding: .75rem 1rem; }
    .banner-partial { border-color: #b7791f; background: #fff8e7; }
    .banner-failed { border-color: #c53030; background: #fff0f0; }
    .banner-stale { border-color: #596579; background: #edf0f5; }
    details summary { cursor: pointer; user-select: none; }
    details summary h2, details summary h3, details summary h4 { display: inline; margin: 0 0 0 0.35rem; vertical-align: middle; }
    details[open] > summary { margin-bottom: 0.75rem; }

    /* Level toolbars and buttons */
    .level-toolbar { display: flex; align-items: center; justify-content: space-between; gap: .5rem; padding: .35rem .6rem; margin-bottom: .6rem; background: #f8fafc; border-radius: .375rem; border: 1px solid #e2e8f0; }
    .level-badge { font-size: .75rem; font-weight: 700; text-transform: uppercase; letter-spacing: .04em; padding: .15rem .45rem; border-radius: .25rem; }
    .level-badge-cluster { background: #dbeafe; color: #1e40af; }
    .level-badge-collection { background: #ede9fe; color: #6d28d9; }
    .level-badge-index { background: #ccfbf1; color: #0f766e; }
    .btn-toggle-level { font-size: .8rem; font-weight: 600; color: #334155; background: #ffffff; border: 1px solid #cbd5e1; border-radius: .3rem; padding: .25rem .55rem; cursor: pointer; transition: all .15s ease; }
    .btn-toggle-level:hover { background: #f1f5f9; border-color: #94a3b8; color: #0f172a; }
    .btn-toggle-level:active { background: #e2e8f0; }

    /* Distinct level colors */
    /* Level 1: Clusters */
    details.cluster { background: #fff; border: 1px solid #bfdbfe; border-left: 5px solid #2563eb; border-radius: .6rem; padding: 1rem; margin: 1rem 0; box-shadow: 0 1px 3px rgba(0,0,0,.04); scroll-margin-top: calc(var(--filter-bar-height) + 1rem); }
    details.cluster > summary { color: #1d4ed8; font-weight: 600; }
    details.cluster > summary:hover { color: #1e40af; }
    details.cluster > summary h2 { margin-top: 0; font-size: 1.35rem; }

    /* Level 2: Collections */
    details.collection { background: #faf5ff; border: 1px solid #ddd6fe; border-left: 5px solid #7c3aed; border-radius: .5rem; margin-top: 1rem; padding: .85rem; scroll-margin-top: calc(var(--filter-bar-height) + 1rem); }
    details.collection > summary { color: #6d28d9; font-weight: 600; }
    details.collection > summary:hover { color: #5b21b6; }
    details.collection > summary h3 { margin-top: 0; font-size: 1.15rem; }

    /* Level 3: Indexes */
    details.index { background: #f0fdfa; border: 1px solid #99f6e4; border-left: 5px solid #0d9488; border-radius: .4rem; padding: .75rem; margin: .75rem 0; scroll-margin-top: calc(var(--filter-bar-height) + 1rem); }
    details.index > summary { color: #0f766e; font-weight: 600; }
    details.index > summary:hover { color: #115e59; }
    details.index h4 { margin: 0; font-size: 1rem; }

    .cluster-total { font-weight: 600; }
    .table-wrap { overflow-x: auto; width: 100%; }
    table { border-collapse: collapse; min-width: 28rem; width: 100%; background: #fff; border-radius: .3rem; }
    caption { font-size: .9rem; font-weight: 600; padding: .5rem; text-align: left; }
    th, td { border-bottom: 1px solid #d8deea; padding: .6rem; text-align: left; }
    th { background: #edf1f7; }
    .empty-state { text-align: center; }
    .flash-highlight { outline: 3px solid #2563eb !important; outline-offset: 3px; border-radius: .4rem; transition: outline .3s ease; }
    @media (max-width: 860px) {
      .layout-wrap { flex-direction: column; }
      .sidebar { width: 100%; max-width: 100%; position: static; max-height: 250px; }
      .main-column { width: 100%; }
      dl { grid-template-columns: 1fr; gap: .2rem; }
      dd { margin-bottom: .5rem; }
      table { min-width: 24rem; }
    }
  </style>
</head>
<body>
  <div class="filter-bar" role="search">
    <div class="filter-container">
      <div class="filter-top-row">
        <label for="filter-input" class="filter-label">Filter</label>
        <input type="search" id="filter-input" class="filter-input" placeholder="Search clusters, collections, or indexes..." autocomplete="off" spellcheck="false">
      </div>
      <div class="filter-bottom-row">
        <div class="filter-toggles">
          <label class="filter-toggle-label">
            <input type="checkbox" id="filter-case"> Case sensitive
          </label>
          <label class="filter-toggle-label">
            <input type="checkbox" id="filter-exact"> Exact match
          </label>
        </div>
        <div class="refresh-box">
          <label for="refresh-interval-select">Refresh:</label>
          <select id="refresh-interval-select" class="refresh-select">
            <option value="600000">10 min</option>
            <option value="300000" selected>5 min</option>
            <option value="60000">1 min</option>
            <option value="30000">30s</option>
            <option value="10000">10s</option>
            <option value="0">Never (reload only)</option>
          </select>
          <span class="live-pill" id="live-indicator"><span class="live-dot"></span><span id="live-pill-text">Live</span></span>
          <a href="/download/report.json" class="btn-download" aria-label="Download latest report as JSON">Download JSON</a>
          <button type="button" id="btn-open-config" class="btn-config" aria-label="Open Configuration">⚙️ Configuration</button>
        </div>
      </div>
    </div>
  </div>
  <header>
    <h1>Index Usage Dashboard</h1>
    <p>Index operation counts by observed cluster and node.</p>
  </header>
  <div class="layout-wrap">
    <aside class="sidebar" aria-label="Data structure index">
      <h2>Structure Index</h2>
      <div class="toc-toolbar">
        <button type="button" class="btn-toggle-level" data-action="toggle-toc-clusters">Toggle all clusters</button>
        <button type="button" class="btn-toggle-level" data-action="toggle-toc-collections">Toggle all collections</button>
      </div>
      <nav class="sidebar-toc">
        {{if .Clusters}}
          {{range $ci, $cluster := .Clusters}}
          <details id="toc-cluster-{{$ci}}" class="toc-cluster" data-name="{{$cluster.Name}}" data-path="toc-cluster:{{$cluster.Name}}">
            <summary><a href="#cluster-{{$ci}}" class="toc-link">{{$cluster.Name}}</a></summary>
            <div class="toc-cluster-body">
              <div class="toc-sub-toolbar">
                <button type="button" class="btn-toggle-level" data-action="toggle-single-toc-collections">Toggle collections</button>
              </div>
              {{range $coi, $col := $cluster.Collections}}
              <details id="toc-cluster-{{$ci}}-col-{{$coi}}" class="toc-collection" data-name="{{$col.Name}}" data-path="toc-col:{{$cluster.Name}}/{{$col.Name}}">
                <summary><a href="#cluster-{{$ci}}-col-{{$coi}}" class="toc-link">{{$col.Name}}</a></summary>
                <div class="toc-index-wrap">
                  {{range $ii, $idx := $col.Indexes}}
                  <a id="toc-cluster-{{$ci}}-col-{{$coi}}-idx-{{$ii}}" href="#cluster-{{$ci}}-col-{{$coi}}-idx-{{$ii}}" class="toc-link toc-index-link" data-name="{{$idx.Name}}">{{$idx.Name}}</a>
                  {{end}}
                </div>
              </details>
              {{end}}
            </div>
          </details>
          {{end}}
        {{else}}
          <p class="toc-empty">No clusters available</p>
        {{end}}
      </nav>
    </aside>
    <main class="main-column">
      <section class="freshness" aria-labelledby="freshness-heading">
        <h2 id="freshness-heading">Data freshness</h2>
        <dl>
          <dt>Last attempt</dt>
          <dd>{{with .LastAttempt}}<time datetime="{{.DateTime}}">{{.Display}}</time>{{else}}Not available{{end}}</dd>
          <dt>Last successful collection</dt>
          <dd>{{with .LastSuccess}}<time datetime="{{.DateTime}}">{{.Display}}</time>{{else}}Not available{{end}}</dd>
        </dl>
      </section>
      <div class="collection-status-section">
        {{if or .Partial .Failed .Stale}}
        <section aria-label="Collection status">
          {{if .Partial}}<p class="banner banner-partial" role="status">The report contains partial data.</p>{{end}}
          {{if .Failed}}<p class="banner banner-failed" role="status">The latest collection attempt failed.</p>{{end}}
          {{if .Stale}}<p class="banner banner-stale" role="status">The displayed report may be stale.</p>{{end}}
        </section>
        {{end}}
      </div>
      <p id="filter-no-results" class="banner banner-stale" role="status" hidden>No clusters, collections, or indexes match your filter.</p>
      <div class="clusters-container">
        {{if .Clusters}}
          <div class="level-toolbar clusters-toolbar">
            <span class="level-badge level-badge-cluster">Clusters Level</span>
            <button type="button" class="btn-toggle-level" data-action="toggle-all-clusters">Toggle all clusters</button>
          </div>
          {{range $ci, $cluster := .Clusters}}
          <details id="cluster-{{$ci}}" class="cluster" data-name="{{$cluster.Name}}" data-path="cluster:{{$cluster.Name}}">
            <summary><h2>{{$cluster.Name}}</h2></summary>
            <div class="cluster-body">
              <div class="level-toolbar collections-toolbar">
                <span class="level-badge level-badge-collection">Collections Level</span>
                <button type="button" class="btn-toggle-level" data-action="toggle-cluster-collections">Toggle all collections</button>
              </div>
              {{range $coi, $col := $cluster.Collections}}
              <details id="cluster-{{$ci}}-col-{{$coi}}" class="collection" data-name="{{$col.Name}}" data-path="col:{{$cluster.Name}}/{{$col.Name}}">
                <summary><h3>{{$col.Name}}</h3></summary>
                <div class="collection-body">
                  <div class="level-toolbar indexes-toolbar">
                    <span class="level-badge level-badge-index">Indexes Level</span>
                    <button type="button" class="btn-toggle-level" data-action="toggle-collection-indexes">Toggle all indexes</button>
                  </div>
                  {{range $ii, $idx := $col.Indexes}}
                  <details id="cluster-{{$ci}}-col-{{$coi}}-idx-{{$ii}}" class="index" data-name="{{$idx.Name}}" data-path="idx:{{$cluster.Name}}/{{$col.Name}}/{{$idx.Name}}">
                    <summary><h4>{{$idx.Name}}</h4></summary>
                    <div class="index-body">
                      <p class="cluster-total">Total: <data value="{{.Total}}">{{.Total}}</data></p>
                      {{if .Nodes}}
                      <div class="table-wrap">
                        <table>
                          <caption>Observed nodes</caption>
                          <thead><tr><th scope="col">Node</th><th scope="col">Operations</th><th scope="col">Role</th></tr></thead>
                          <tbody>
                            {{range .Nodes}}
                            <tr><th scope="row">{{.Name}}</th><td>{{.Count}}</td><td>{{if .Primary}}Primary{{else}}Non-primary{{end}}</td></tr>
                            {{end}}
                          </tbody>
                        </table>
                      </div>
                      {{else}}
                      <p>No node observations were returned for this index.</p>
                      {{end}}
                    </div>
                  </details>
                  {{end}}
                </div>
              </details>
              {{end}}
            </div>
          </details>
          {{end}}
        {{else}}
          <section class="empty-state" aria-labelledby="empty-heading">
            <h2 id="empty-heading">No report data</h2>
            <p>No index usage data is available yet.</p>
          </section>
        {{end}}
      </div>
    </main>
  </div>
  <div id="config-modal" class="modal-backdrop" hidden>
    <div class="modal-dialog" role="dialog" aria-modal="true" aria-labelledby="config-modal-title">
      <div class="modal-header">
        <h2 id="config-modal-title">⚙️ checkIndex Configuration</h2>
        <button type="button" class="modal-close-btn" id="btn-close-config" aria-label="Close configuration">✕</button>
      </div>
      <div class="modal-body">
        <div id="config-alert" class="cfg-alert" hidden></div>
        <div class="cfg-tabs">
          <button type="button" class="cfg-tab-btn active" id="tab-btn-form">Settings &amp; Profiles</button>
          <button type="button" class="cfg-tab-btn" id="tab-btn-json">Raw JSON</button>
        </div>
        <div id="cfg-view-form">
          <div class="cfg-section">
            <h3 class="cfg-section-title">Shared Settings</h3>
            <div class="cfg-grid-2">
              <div class="cfg-group">
                <label class="cfg-label" for="cfg-dbname">Database Name (dbName)</label>
                <input type="text" id="cfg-dbname" class="cfg-input" placeholder="e.g. database" required>
              </div>
              <div class="cfg-group">
                <label class="cfg-label" for="cfg-authsource">Auth Source (authSource)</label>
                <input type="text" id="cfg-authsource" class="cfg-input" placeholder="e.g. admin" required>
              </div>
            </div>
            <div class="cfg-group" style="margin-top: 0.75rem;">
              <div class="cfg-row-space">
                <label class="cfg-label">Global Host Mappings (optional)</label>
                <button type="button" class="cfg-btn-sm" id="btn-add-global-mapping">+ Add Mapping</button>
              </div>
              <div id="cfg-global-mappings" class="cfg-mappings-list"></div>
            </div>
          </div>
          <div class="cfg-section" style="margin-top: 1rem;">
            <div class="cfg-row-space" style="margin-bottom: 0.75rem;">
              <h3 class="cfg-section-title" style="margin: 0;">Cluster Profiles</h3>
              <button type="button" class="cfg-btn-primary cfg-btn-sm" id="btn-add-profile">+ Add Profile</button>
            </div>
            <div id="cfg-profiles-container" class="cfg-profiles-list"></div>
          </div>
        </div>
        <div id="cfg-view-json" hidden>
          <p class="cfg-help-text">Directly edit <code>config.json</code>. Form view and JSON view stay synchronized.</p>
          <textarea id="cfg-json-text" class="cfg-json-textarea" spellcheck="false"></textarea>
        </div>
      </div>
      <div class="modal-footer">
        <button type="button" class="cfg-btn-secondary" id="btn-cancel-config">Cancel</button>
        <button type="button" class="cfg-btn-primary" id="btn-save-config">Save Configuration</button>
      </div>
    </div>
  </div>
  <script>
    (function() {
      // The meta refresh lives inside <noscript>, so with JavaScript enabled the
      // page is never fully reloaded; live refresh below updates it in place.

      function updateFilterBarHeight() {
        var bar = document.querySelector('.filter-bar');
        if (bar) {
          var h = bar.offsetHeight;
          document.documentElement.style.setProperty('--filter-bar-height', h + 'px');
        }
      }
      window.addEventListener('resize', updateFilterBarHeight);
      updateFilterBarHeight();

      var filterInput = document.getElementById('filter-input');
      var filterCase = document.getElementById('filter-case');
      var filterExact = document.getElementById('filter-exact');
      var noResults = document.getElementById('filter-no-results');

      function matchesName(name, query, isCase, isExact) {
        if (!query) return true;
        var target = isCase ? name : name.toLowerCase();
        var q = isCase ? query : query.toLowerCase();
        if (isExact) {
          return target === q;
        }
        return target.includes(q);
      }

      function applyFilter() {
        if (!filterInput) return;
        var rawQuery = filterInput.value.trim();
        var isCase = filterCase ? filterCase.checked : false;
        var isExact = filterExact ? filterExact.checked : false;
        var query = rawQuery;

        var clusters = document.querySelectorAll('.clusters-container > details.cluster');
        var tocClusters = document.querySelectorAll('.sidebar-toc > details.toc-cluster');
        var anyClusterVisible = false;

        if (!query) {
          if (noResults) noResults.hidden = true;
          clusters.forEach(function(cluster) {
            cluster.hidden = false;
            var collections = cluster.querySelectorAll('details.collection');
            collections.forEach(function(collection) {
              collection.hidden = false;
              var indexes = collection.querySelectorAll('details.index');
              indexes.forEach(function(index) {
                index.hidden = false;
              });
            });
          });
          tocClusters.forEach(function(tc) {
            tc.hidden = false;
            tc.querySelectorAll('details.toc-collection').forEach(function(tcol) {
              tcol.hidden = false;
            });
            tc.querySelectorAll('.toc-index-link').forEach(function(tidx) {
              tidx.hidden = false;
            });
          });
          try { sessionStorage.removeItem('dashboard_filter'); } catch(e) {}
          return;
        }

        try { sessionStorage.setItem('dashboard_filter', rawQuery); } catch(e) {}

        clusters.forEach(function(cluster, ci) {
          var clusterName = cluster.getAttribute('data-name') || '';
          var clusterMatches = matchesName(clusterName, query, isCase, isExact);
          var clusterHasMatchingChild = false;

          var collections = cluster.querySelectorAll('details.collection');
          collections.forEach(function(collection) {
            var collectionName = collection.getAttribute('data-name') || '';
            var collectionMatches = matchesName(collectionName, query, isCase, isExact);
            var collectionHasMatchingChild = false;

            var indexes = collection.querySelectorAll('details.index');
            indexes.forEach(function(index) {
              var indexName = index.getAttribute('data-name') || '';
              var indexMatches = matchesName(indexName, query, isCase, isExact);

              var showIndex = indexMatches || collectionMatches || clusterMatches;
              index.hidden = !showIndex;
              if (indexMatches) {
                collectionHasMatchingChild = true;
              }
            });

            var showCollection = collectionMatches || collectionHasMatchingChild || clusterMatches;
            collection.hidden = !showCollection;
            if (showCollection) {
              clusterHasMatchingChild = true;
              collection.open = true;
            } else {
              collection.open = false;
            }
          });

          var showCluster = clusterMatches || clusterHasMatchingChild;
          cluster.hidden = !showCluster;
          if (showCluster) {
            cluster.open = true;
            anyClusterVisible = true;
          } else {
            cluster.open = false;
          }

          // Sync matching TOC cluster
          var tocCluster = document.getElementById('toc-cluster-' + ci);
          if (tocCluster) {
            tocCluster.hidden = !showCluster;
            if (showCluster) {
              tocCluster.open = true;
              var tocCols = tocCluster.querySelectorAll('details.toc-collection');
              tocCols.forEach(function(tcol, coi) {
                var mainCol = document.getElementById('cluster-' + ci + '-col-' + coi);
                var isColVis = mainCol && !mainCol.hidden;
                tcol.hidden = !isColVis;
                if (isColVis) {
                  tcol.open = true;
                  tcol.querySelectorAll('.toc-index-link').forEach(function(til, ii) {
                    var mainIdx = document.getElementById('cluster-' + ci + '-col-' + coi + '-idx-' + ii);
                    til.hidden = !(mainIdx && !mainIdx.hidden);
                  });
                } else {
                  tcol.open = false;
                }
              });
            } else {
              tocCluster.open = false;
            }
          }
        });

        if (noResults) {
          noResults.hidden = (clusters.length === 0 || anyClusterVisible);
        }
      }

      if (filterInput) {
        filterInput.addEventListener('input', applyFilter);
      }
      if (filterCase) {
        filterCase.addEventListener('change', function() {
          try { sessionStorage.setItem('dashboard_filter_case', filterCase.checked); } catch(e) {}
          applyFilter();
        });
      }
      if (filterExact) {
        filterExact.addEventListener('change', function() {
          try { sessionStorage.setItem('dashboard_filter_exact', filterExact.checked); } catch(e) {}
          applyFilter();
        });
      }

      // Restore filter options
      try {
        if (filterCase && sessionStorage.getItem('dashboard_filter_case') === 'true') {
          filterCase.checked = true;
        }
        if (filterExact && sessionStorage.getItem('dashboard_filter_exact') === 'true') {
          filterExact.checked = true;
        }
        var savedFilter = sessionStorage.getItem('dashboard_filter');
        if (savedFilter && filterInput) {
          filterInput.value = savedFilter;
          applyFilter();
        }
      } catch(e) {}

      // Level toggle buttons delegation
      document.addEventListener('click', function(e) {
        var btn = e.target.closest('.btn-toggle-level');
        if (!btn) return;
        var action = btn.getAttribute('data-action');
        if (!action) return;

        if (action === 'toggle-all-clusters') {
          var clusters = document.querySelectorAll('.clusters-container > details.cluster');
          var anyClosed = Array.from(clusters).some(function(d) { return !d.open; });
          clusters.forEach(function(d) { d.open = anyClosed; });
        } else if (action === 'toggle-cluster-collections') {
          var clusterEl = btn.closest('details.cluster');
          if (clusterEl) {
            var collections = clusterEl.querySelectorAll('.cluster-body > details.collection');
            var anyClosed = Array.from(collections).some(function(d) { return !d.open; });
            collections.forEach(function(d) { d.open = anyClosed; });
          }
        } else if (action === 'toggle-collection-indexes') {
          var colEl = btn.closest('details.collection');
          if (colEl) {
            var indexes = colEl.querySelectorAll('.collection-body > details.index');
            var anyClosed = Array.from(indexes).some(function(d) { return !d.open; });
            indexes.forEach(function(d) { d.open = anyClosed; });
          }
        } else if (action === 'toggle-toc-clusters') {
          var tocClusters = document.querySelectorAll('.sidebar-toc > details.toc-cluster');
          var anyClosed = Array.from(tocClusters).some(function(d) { return !d.open; });
          tocClusters.forEach(function(d) { d.open = anyClosed; });
        } else if (action === 'toggle-toc-collections') {
          var tocCols = document.querySelectorAll('.sidebar-toc details.toc-collection');
          var anyClosed = Array.from(tocCols).some(function(d) { return !d.open; });
          tocCols.forEach(function(d) {
            d.open = anyClosed;
            if (anyClosed) {
              var parentToc = d.closest('details.toc-cluster');
              if (parentToc) parentToc.open = true;
            }
          });
        } else if (action === 'toggle-single-toc-collections') {
          var tocClusterEl = btn.closest('details.toc-cluster');
          if (tocClusterEl) {
            var cols = tocClusterEl.querySelectorAll('details.toc-collection');
            var anyClosed = Array.from(cols).some(function(d) { return !d.open; });
            cols.forEach(function(d) { d.open = anyClosed; });
          }
        }
      });

      // TOC smooth navigation and auto-expand without header overlap
      document.addEventListener('click', function(e) {
        var link = e.target.closest('a.toc-link');
        if (!link) return;
        var href = link.getAttribute('href');
        if (!href || !href.startsWith('#')) return;

        e.preventDefault();
        var targetId = href.slice(1);
        var targetEl = document.getElementById(targetId);
        if (targetEl) {
          var cur = targetEl;
          while (cur && cur !== document.body) {
            if (cur.tagName === 'DETAILS') {
              cur.open = true;
            }
            cur = cur.parentElement;
          }
          var bar = document.querySelector('.filter-bar');
          var barHeight = bar ? bar.offsetHeight : 80;
          var rect = targetEl.getBoundingClientRect();
          var scrollTop = window.pageYOffset || document.documentElement.scrollTop;
          var targetY = rect.top + scrollTop - barHeight - 16;
          window.scrollTo({ top: Math.max(0, targetY), behavior: 'smooth' });
          targetEl.classList.add('flash-highlight');
          setTimeout(function() {
            targetEl.classList.remove('flash-highlight');
          }, 1500);
        }
      });

      // Real-time live polling without page reload
      var refreshIntervalMs = 300000; // default 5 minutes
      var refreshTimer = null;
      var fastInitialTimer = null;
      var isRefreshing = false;

      async function fetchAndUpdate() {
        if (isRefreshing) return;
        isRefreshing = true;
        var livePill = document.getElementById('live-pill-text');
        if (livePill) livePill.textContent = 'Updating...';

        try {
          var res = await fetch(window.location.pathname + window.location.search, {
            cache: 'no-store',
            headers: { 'Accept': 'text/html' }
          });
          if (!res.ok) throw new Error('Refresh response not ok: ' + res.status);
          var html = await res.text();
          var doc = new DOMParser().parseFromString(html, 'text/html');

          // Capture exact open state of all details elements using data-path (or id)
          var openPaths = new Set();
          document.querySelectorAll('details[open]').forEach(function(d) {
            var path = d.getAttribute('data-path') || d.id;
            if (path) openPaths.add(path);
          });

          // Update freshness
          var newFreshness = doc.querySelector('.freshness');
          var oldFreshness = document.querySelector('.freshness');
          if (newFreshness && oldFreshness) {
            oldFreshness.innerHTML = newFreshness.innerHTML;
          }

          // Update banners
          var newStatus = doc.querySelector('.collection-status-section');
          var oldStatus = document.querySelector('.collection-status-section');
          if (newStatus && oldStatus) {
            oldStatus.innerHTML = newStatus.innerHTML;
          }

          // Update clusters container
          var newClustersWrap = doc.querySelector('.clusters-container');
          var oldClustersWrap = document.querySelector('.clusters-container');
          if (newClustersWrap && oldClustersWrap) {
            oldClustersWrap.innerHTML = newClustersWrap.innerHTML;
          }

          // Update sidebar TOC
          var newToc = doc.querySelector('.sidebar-toc');
          var oldToc = document.querySelector('.sidebar-toc');
          if (newToc && oldToc) {
            oldToc.innerHTML = newToc.innerHTML;
          }

          // Restore exact open state of all details elements
          document.querySelectorAll('details').forEach(function(d) {
            var path = d.getAttribute('data-path') || d.id;
            if (path && openPaths.has(path)) {
              d.open = true;
            } else {
              d.open = false;
            }
          });

          // Re-apply filter without resetting open state
          applyFilter();

          if (livePill) {
            var now = new Date();
            var hh = String(now.getHours()).padStart(2, '0');
            var mm = String(now.getMinutes()).padStart(2, '0');
            var ss = String(now.getSeconds()).padStart(2, '0');
            livePill.textContent = 'Live (' + hh + ':' + mm + ':' + ss + ')';
          }
        } catch(err) {
          console.error('Live refresh error:', err);
          if (livePill) livePill.textContent = 'Retry';
        } finally {
          isRefreshing = false;
        }
      }

      function startRefreshTimer(ms) {
        if (refreshTimer) {
          clearInterval(refreshTimer);
          refreshTimer = null;
        }
        refreshIntervalMs = ms;
        var livePill = document.getElementById('live-indicator');
        var livePillText = document.getElementById('live-pill-text');
        var liveDot = livePill ? livePill.querySelector('.live-dot') : null;

        if (ms <= 0) {
          if (livePillText) livePillText.textContent = 'Never (manual)';
          if (liveDot) liveDot.style.backgroundColor = '#94a3b8';
          return;
        }

        if (liveDot) liveDot.style.backgroundColor = '#16a34a';
        refreshTimer = setInterval(fetchAndUpdate, refreshIntervalMs);
      }

      // Fast initial polling if no report data is rendered yet
      function initDataRefresh() {
        if (refreshIntervalMs <= 0) {
          startRefreshTimer(0);
          return;
        }

        var hasClusters = document.querySelectorAll('.clusters-container details.cluster').length > 0;
        if (hasClusters) {
          startRefreshTimer(refreshIntervalMs);
          return;
        }

        var livePillText = document.getElementById('live-pill-text');
        if (livePillText) livePillText.textContent = 'Awaiting data...';

        // Query immediately shortly after startup
        setTimeout(async function() {
          await fetchAndUpdate();
          if (document.querySelectorAll('.clusters-container details.cluster').length > 0) {
            startRefreshTimer(refreshIntervalMs);
            return;
          }

          // Rapid poll every 1.5s until data arrives
          fastInitialTimer = setInterval(async function() {
            await fetchAndUpdate();
            if (document.querySelectorAll('.clusters-container details.cluster').length > 0) {
              clearInterval(fastInitialTimer);
              fastInitialTimer = null;
              startRefreshTimer(refreshIntervalMs);
            }
          }, 1500);
        }, 500);
      }

      var refreshSelect = document.getElementById('refresh-interval-select');
      if (refreshSelect) {
        try {
          var savedInterval = sessionStorage.getItem('dashboard_refresh_ms');
          if (savedInterval !== null && ['600000', '300000', '60000', '30000', '10000', '0'].includes(savedInterval)) {
            refreshSelect.value = savedInterval;
            refreshIntervalMs = parseInt(savedInterval, 10);
          }
        } catch(e) {}

        initDataRefresh();

        refreshSelect.addEventListener('change', function() {
          var val = parseInt(this.value, 10);
          if (!isNaN(val)) {
            try { sessionStorage.setItem('dashboard_refresh_ms', String(val)); } catch(e) {}
            startRefreshTimer(val);
          }
        });
      }

      // --- Configuration Modal Management ---
      var configModal = document.getElementById('config-modal');
      var btnOpenConfig = document.getElementById('btn-open-config');
      var btnCloseConfig = document.getElementById('btn-close-config');
      var btnCancelConfig = document.getElementById('btn-cancel-config');
      var btnSaveConfig = document.getElementById('btn-save-config');
      var configAlert = document.getElementById('config-alert');
      var tabBtnForm = document.getElementById('tab-btn-form');
      var tabBtnJson = document.getElementById('tab-btn-json');
      var cfgViewForm = document.getElementById('cfg-view-form');
      var cfgViewJson = document.getElementById('cfg-view-json');
      var cfgJsonText = document.getElementById('cfg-json-text');
      var btnAddGlobalMapping = document.getElementById('btn-add-global-mapping');
      var btnAddProfile = document.getElementById('btn-add-profile');

      var currentConfigState = {
        dbName: 'database',
        authSource: 'admin',
        hostMapping: {},
        cluster: {}
      };
      var activeTab = 'form';

      function showConfigAlert(msg, isSuccess) {
        if (!configAlert) return;
        configAlert.textContent = msg;
        configAlert.className = 'cfg-alert ' + (isSuccess ? 'cfg-alert-success' : 'cfg-alert-error');
        configAlert.hidden = false;
      }

      function hideConfigAlert() {
        if (configAlert) {
          configAlert.hidden = true;
          configAlert.textContent = '';
        }
      }

      function createMappingRow(key, val, container) {
        var row = document.createElement('div');
        row.className = 'cfg-mapping-row';
        row.innerHTML = '<input type="text" class="cfg-input cfg-map-k" placeholder="Host alias (e.g. mongo1:27017)" value="' + (key || '') + '">' +
                        '<input type="text" class="cfg-input cfg-map-v" placeholder="Target host (e.g. 127.0.0.1:27017)" value="' + (val || '') + '">' +
                        '<button type="button" class="cfg-btn-sm cfg-btn-danger" aria-label="Remove mapping">✕</button>';
        row.querySelector('button').addEventListener('click', function() {
          row.remove();
        });
        container.appendChild(row);
      }

      function renderProfiles(clusters) {
        var container = document.getElementById('cfg-profiles-container');
        if (!container) return;
        container.innerHTML = '';

        var names = Object.keys(clusters || {});
        if (names.length === 0) {
          names = ['default'];
          clusters = { 'default': { initialUri: '', username: '', hasPassword: false, hostMapping: {} } };
        }

        names.forEach(function(name, idx) {
          var prof = clusters[name] || {};
          var card = document.createElement('div');
          card.className = 'cfg-profile-card';
          card.dataset.index = idx;

          var passwordBadge = prof.hasPassword ? '<span class="cfg-badge cfg-badge-success">Password Configured</span>' : '<span class="cfg-badge cfg-badge-muted">No Password</span>';
          var passwordPlaceholder = prof.hasPassword ? 'Leave blank to keep current password' : 'Enter MongoDB password';

          card.innerHTML = 
            '<div class="cfg-profile-header">' +
              '<div class="cfg-profile-name-title"><span>Cluster Profile:</span> ' +
                '<input type="text" class="cfg-input cfg-prof-name" style="width: auto; min-width: 140px; font-weight: 700;" value="' + name + '" required>' +
                passwordBadge +
              '</div>' +
              (names.length > 1 ? '<button type="button" class="cfg-btn-sm cfg-btn-danger btn-remove-profile">Delete Profile</button>' : '') +
            '</div>' +
            '<div class="cfg-grid-2">' +
              '<div class="cfg-group">' +
                '<label class="cfg-label">Initial URI (initialUri)</label>' +
                '<input type="text" class="cfg-input cfg-prof-uri" placeholder="mongodb://host1:27017,host2:27017" value="' + (prof.initialUri || '') + '" required>' +
              '</div>' +
              '<div class="cfg-group">' +
                '<label class="cfg-label">Username</label>' +
                '<input type="text" class="cfg-input cfg-prof-user" placeholder="appuser" value="' + (prof.username || '') + '" required>' +
              '</div>' +
            '</div>' +
            '<div class="cfg-group" style="margin-top: .75rem;">' +
              '<label class="cfg-label">Password</label>' +
              '<div class="cfg-input-pass-wrap">' +
                '<input type="password" class="cfg-input cfg-prof-pass" placeholder="' + passwordPlaceholder + '">' +
                '<button type="button" class="cfg-btn-toggle-pass">Show</button>' +
              '</div>' +
              '<span class="cfg-help-text" style="margin-top: .25rem;">' + (prof.hasPassword ? 'Password is saved on disk. Only fill this in if you want to change it.' : 'Enter cluster authentication password.') + '</span>' +
            '</div>' +
            '<div class="cfg-group" style="margin-top: .75rem;">' +
              '<div class="cfg-row-space">' +
                '<label class="cfg-label">Profile Host Mappings (optional)</label>' +
                '<button type="button" class="cfg-btn-sm btn-add-prof-map">+ Add Mapping</button>' +
              '</div>' +
              '<div class="cfg-mappings-list prof-map-container"></div>' +
            '</div>';

          var passInput = card.querySelector('.cfg-prof-pass');
          var passToggle = card.querySelector('.cfg-btn-toggle-pass');
          if (passToggle && passInput) {
            passToggle.addEventListener('click', function() {
              if (passInput.type === 'password') {
                passInput.type = 'text';
                passToggle.textContent = 'Hide';
              } else {
                passInput.type = 'password';
                passToggle.textContent = 'Show';
              }
            });
          }

          var removeBtn = card.querySelector('.btn-remove-profile');
          if (removeBtn) {
            removeBtn.addEventListener('click', function() {
              card.remove();
            });
          }

          var mapContainer = card.querySelector('.prof-map-container');
          var addMapBtn = card.querySelector('.btn-add-prof-map');
          if (prof.hostMapping) {
            Object.keys(prof.hostMapping).forEach(function(mk) {
              createMappingRow(mk, prof.hostMapping[mk], mapContainer);
            });
          }
          if (addMapBtn && mapContainer) {
            addMapBtn.addEventListener('click', function() {
              createMappingRow('', '', mapContainer);
            });
          }

          container.appendChild(card);
        });
      }

      function renderConfigForm(cfg) {
        var dbInput = document.getElementById('cfg-dbname');
        var authInput = document.getElementById('cfg-authsource');
        if (dbInput) dbInput.value = cfg.dbName || '';
        if (authInput) authInput.value = cfg.authSource || '';

        var globalMapContainer = document.getElementById('cfg-global-mappings');
        if (globalMapContainer) {
          globalMapContainer.innerHTML = '';
          if (cfg.hostMapping) {
            Object.keys(cfg.hostMapping).forEach(function(k) {
              createMappingRow(k, cfg.hostMapping[k], globalMapContainer);
            });
          }
        }

        renderProfiles(cfg.cluster);
      }

      function collectFormState() {
        var dbInput = document.getElementById('cfg-dbname');
        var authInput = document.getElementById('cfg-authsource');
        currentConfigState.dbName = dbInput ? dbInput.value.trim() : '';
        currentConfigState.authSource = authInput ? authInput.value.trim() : '';

        var globalMapping = {};
        var gRows = document.querySelectorAll('#cfg-global-mappings .cfg-mapping-row');
        gRows.forEach(function(row) {
          var k = row.querySelector('.cfg-map-k').value.trim();
          var v = row.querySelector('.cfg-map-v').value.trim();
          if (k && v) globalMapping[k] = v;
        });
        currentConfigState.hostMapping = globalMapping;

        var clusterObj = {};
        var profileCards = document.querySelectorAll('#cfg-profiles-container .cfg-profile-card');
        profileCards.forEach(function(card) {
          var nameInput = card.querySelector('.cfg-prof-name');
          var uriInput = card.querySelector('.cfg-prof-uri');
          var userInput = card.querySelector('.cfg-prof-user');
          var passInput = card.querySelector('.cfg-prof-pass');

          var name = nameInput ? nameInput.value.trim() : '';
          if (!name) return;

          var profMaps = {};
          var pRows = card.querySelectorAll('.prof-map-container .cfg-mapping-row');
          pRows.forEach(function(row) {
            var k = row.querySelector('.cfg-map-k').value.trim();
            var v = row.querySelector('.cfg-map-v').value.trim();
            if (k && v) profMaps[k] = v;
          });

          var existingProf = (currentConfigState.cluster && currentConfigState.cluster[name]) || {};
          clusterObj[name] = {
            initialUri: uriInput ? uriInput.value.trim() : '',
            username: userInput ? userInput.value.trim() : '',
            password: passInput ? passInput.value : '',
            hasPassword: existingProf.hasPassword || (passInput && passInput.value !== ''),
            hostMapping: profMaps
          };
        });
        currentConfigState.cluster = clusterObj;
      }

      if (btnAddGlobalMapping) {
        btnAddGlobalMapping.addEventListener('click', function() {
          var container = document.getElementById('cfg-global-mappings');
          if (container) createMappingRow('', '', container);
        });
      }

      if (btnAddProfile) {
        btnAddProfile.addEventListener('click', function() {
          collectFormState();
          var newName = 'cluster-' + (Object.keys(currentConfigState.cluster).length + 1);
          currentConfigState.cluster[newName] = {
            initialUri: '',
            username: '',
            hasPassword: false,
            hostMapping: {}
          };
          renderProfiles(currentConfigState.cluster);
        });
      }

      function switchConfigTab(tab) {
        activeTab = tab;
        hideConfigAlert();
        if (tab === 'form') {
          if (tabBtnForm) tabBtnForm.classList.add('active');
          if (tabBtnJson) tabBtnJson.classList.remove('active');
          if (cfgViewForm) cfgViewForm.hidden = false;
          if (cfgViewJson) cfgViewJson.hidden = true;
          if (cfgJsonText) {
            try {
              var parsed = JSON.parse(cfgJsonText.value);
              currentConfigState = parsed;
              renderConfigForm(currentConfigState);
            } catch(e) {}
          }
        } else {
          if (tabBtnJson) tabBtnJson.classList.add('active');
          if (tabBtnForm) tabBtnForm.classList.remove('active');
          if (cfgViewJson) cfgViewJson.hidden = false;
          if (cfgViewForm) cfgViewForm.hidden = true;
          collectFormState();
          if (cfgJsonText) {
            cfgJsonText.value = JSON.stringify(currentConfigState, null, 2);
          }
        }
      }

      if (tabBtnForm) tabBtnForm.addEventListener('click', function() { switchConfigTab('form'); });
      if (tabBtnJson) tabBtnJson.addEventListener('click', function() { switchConfigTab('json'); });

      function openConfigModal() {
        hideConfigAlert();
        switchConfigTab('form');
        if (configModal) configModal.hidden = false;
        fetch('/api/config')
          .then(function(res) {
            if (!res.ok) throw new Error('HTTP ' + res.status);
            return res.json();
          })
          .then(function(data) {
            currentConfigState = data || { dbName: 'database', authSource: 'admin', hostMapping: {}, cluster: {} };
            if (!currentConfigState.cluster || Object.keys(currentConfigState.cluster).length === 0) {
              currentConfigState.cluster = {
                'default': { initialUri: '', username: '', hasPassword: false, hostMapping: {} }
              };
            }
            renderConfigForm(currentConfigState);
            if (cfgJsonText) cfgJsonText.value = JSON.stringify(currentConfigState, null, 2);
          })
          .catch(function(err) {
            showConfigAlert('Failed to load configuration: ' + err.message, false);
          });
      }

      function closeConfigModal() {
        if (configModal) configModal.hidden = true;
        hideConfigAlert();
      }

      if (btnOpenConfig) btnOpenConfig.addEventListener('click', openConfigModal);
      if (btnCloseConfig) btnCloseConfig.addEventListener('click', closeConfigModal);
      if (btnCancelConfig) btnCancelConfig.addEventListener('click', closeConfigModal);

      if (configModal) {
        configModal.addEventListener('click', function(e) {
          if (e.target === configModal) closeConfigModal();
        });
      }
      window.addEventListener('keydown', function(e) {
        if (e.key === 'Escape' && configModal && !configModal.hidden) {
          closeConfigModal();
        }
      });

      if (btnSaveConfig) {
        btnSaveConfig.addEventListener('click', async function() {
          hideConfigAlert();
          if (activeTab === 'json') {
            try {
              currentConfigState = JSON.parse(cfgJsonText.value);
            } catch(e) {
              showConfigAlert('Invalid JSON: ' + e.message, false);
              return;
            }
          } else {
            collectFormState();
          }

          if (!currentConfigState.dbName) {
            showConfigAlert('Database name (dbName) is required.', false);
            return;
          }
          if (!currentConfigState.authSource) {
            showConfigAlert('Auth source (authSource) is required.', false);
            return;
          }
          var profKeys = Object.keys(currentConfigState.cluster || {});
          if (profKeys.length === 0) {
            showConfigAlert('At least one cluster profile is required.', false);
            return;
          }
          for (var i = 0; i < profKeys.length; i++) {
            var pk = profKeys[i];
            var p = currentConfigState.cluster[pk];
            if (!p.initialUri) {
              showConfigAlert('Profile "' + pk + '" requires an initialUri.', false);
              return;
            }
            if (!p.username) {
              showConfigAlert('Profile "' + pk + '" requires a username.', false);
              return;
            }
          }

          btnSaveConfig.disabled = true;
          btnSaveConfig.textContent = 'Saving...';

          try {
            var res = await fetch('/api/config', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify(currentConfigState)
            });
            var data = await res.json();
            if (!res.ok) {
              throw new Error(data.error || ('Server returned HTTP ' + res.status));
            }

            showConfigAlert('Configuration saved successfully! Refreshing dashboard...', true);
            setTimeout(function() {
              closeConfigModal();
              if (typeof fetchAndUpdate === 'function') {
                fetchAndUpdate();
              }
            }, 800);
          } catch(err) {
            showConfigAlert(err.message, false);
          } finally {
            btnSaveConfig.disabled = false;
            btnSaveConfig.textContent = 'Save Configuration';
          }
        });
      }
    })();
  </script>
</body>
</html>`

var dashboardTemplate = template.Must(template.New("dashboard").Parse(dashboardPage))

// NewHandler creates a GET-only server-rendered dashboard handler.
func NewHandler(service *Service) http.Handler {
	return NewHandlerWithConfig(service, DefaultRefreshInterval, "")
}

// NewHandlerWithRefreshInterval creates a dashboard handler that refreshes the page at interval.
func NewHandlerWithRefreshInterval(service *Service, interval time.Duration) http.Handler {
	return NewHandlerWithConfig(service, interval, "")
}

// NewHandlerWithConfig creates a dashboard handler with a custom refresh interval and config path.
func NewHandlerWithConfig(service *Service, interval time.Duration, configPath string) http.Handler {
	if interval <= 0 {
		interval = DefaultRefreshInterval
	}
	refreshInterval := strconv.FormatFloat(interval.Seconds(), 'f', -1, 64)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/config" {
			handleConfigAPI(w, r, service, configPath)
			return
		}

		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if r.URL.Path == "/download/report.json" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Content-Disposition", `attachment; filename="checkindex-report.json"`)
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			_ = json.NewEncoder(w).Encode(service.Snapshot().Report)
			return
		}

		pageData := struct {
			ViewModel
			RefreshInterval string
		}{
			ViewModel:       BuildViewModel(service.Snapshot()),
			RefreshInterval: refreshInterval,
		}

		if r.URL.Query().Get("format") == "json" || strings.Contains(r.Header.Get("Accept"), "application/json") {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			_ = json.NewEncoder(w).Encode(pageData.ViewModel)
			return
		}

		var body bytes.Buffer
		if err := dashboardTemplate.Execute(&body, pageData); err != nil {
			http.Error(w, "Unable to render dashboard.", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = w.Write(body.Bytes())
	})
}

func handleConfigAPI(w http.ResponseWriter, r *http.Request, service *Service, configPath string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	switch r.Method {
	case http.MethodGet:
		dto, err := LoadConfigDTO(configPath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(dto)
	case http.MethodPost:
		var dto ConfigDTO
		if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON payload: " + err.Error()})
			return
		}
		if err := SaveConfigDTO(configPath, dto); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		if service != nil {
			go func() {
				_ = service.Refresh(context.Background())
			}()
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	default:
		w.Header().Set("Allow", "GET, POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
	}
}
