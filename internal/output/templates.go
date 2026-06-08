package output

const indexTemplate = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{ .Title }}</title>
  <link rel="stylesheet" href="/style.css">
</head>
<body>
  <main>
    <header class="hero">
      <p class="eyebrow">Lobste.rs personal sites</p>
      <h1>{{ .Title }}</h1>
      <p>A static, polite crawl of Lobste.rs users' personal homepages.</p>
    </header>

    <section class="stats" aria-label="stats">
      <div><strong>{{ .Stats.Users }}</strong><span>users</span></div>
      <div><strong>{{ .Stats.Sites }}</strong><span>sites found</span></div>
      <div><strong>{{ .Stats.UserShards }}</strong><span>user shards</span></div>
    </section>

    <section>
      <h2>Discovered sites</h2>
      {{ if .Sites }}
        <ul class="sites">
        {{ range .Sites }}
          <li>
            <a href="{{ .HomepageURL }}">{{ .Username }}</a>
            <small><a href="{{ .ProfileURL }}">lobste.rs profile</a>{{ if .Karma }} · karma {{ .Karma }}{{ end }}</small>
            {{ if .About }}<p>{{ .About }}</p>{{ end }}
          </li>
        {{ end }}
        </ul>
      {{ else }}
        <p>No personal sites discovered yet. Run <code>lobsters-planet discover</code>.</p>
      {{ end }}
    </section>

    <section>
      <h2>Data</h2>
      <ul>
        <li><a href="/data/sites.json">sites.json</a></li>
        <li><a href="/data/stats.json">stats.json</a></li>
        <li><a href="/data/users-index.json">users-index.json</a></li>
      </ul>
    </section>
  </main>
</body>
</html>
`
