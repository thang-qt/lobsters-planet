package output

const pageHead = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{ .Title }}</title>
  <link rel="stylesheet" href="/style.css">
  <link rel="alternate" type="application/rss+xml" title="{{ .Title }}" href="/feed.xml">
</head>
<body>
  <header class="site-nav">
    <a class="brand" href="/">{{ .Title }}</a>
    <nav>
      <a href="/posts/">Posts</a>
      <a href="/users/">Users</a>
      <a href="/analytics/">Analytics</a>
      <a href="/data/sites.json">Data</a>
      <a class="pill" href="/feed.xml">RSS</a>
    </nav>
  </header>
`

const pageFoot = `
  <footer class="site-footer">
    <p>Made on a rainy night by <a href="https://thangqt.com">ThangQT</a>. <a href="https://github.com/thang-qt/lobsters-planet">Source</a>.</p>
  </footer>
</body>
</html>
`

const homeTemplate = pageHead + `
  <main>
    <section class="page-title">
      <h1>Latest posts</h1>
      <p>{{ .Stats.Entries }} recent entries from {{ .Stats.Sites }} discovered personal sites. Last built {{ datetime .Stats.GeneratedAt }}. <a href="/feed.xml">RSS</a> · <a href="/posts/">All posts</a> · <a href="/users/">Users</a></p>
    </section>


    <section class="section-head">
      <h2>Recent posts</h2>
      <a href="/posts/">All posts</a>
    </section>
    <div class="entries compact-feed">
      {{ range .Entries }}
        <article class="entry">
          <p class="meta">{{ entryDate . }} · <a href="{{ userPath .OwnerUsername }}">{{ .OwnerUsername }}</a>{{ if .FeedTitle }} · {{ .FeedTitle }}{{ end }}</p>
          <h3><a href="{{ .URL }}">{{ .Title }}</a></h3>
          {{ if .Summary }}<p>{{ .Summary }}</p>{{ end }}
        </article>
      {{ else }}
        <p>No feed entries yet. Run <code>lobsters-planet refresh</code>.</p>
      {{ end }}
    </div>

    <section class="section-head">
      <h2>Users</h2>
      <a href="/users/">Explore all users</a>
    </section>
    <div class="user-list home-users">
      {{ range .Users }}
        <article class="user-card">
          <h3>{{ if .HasDetailPage }}<a href="{{ userPath .Username }}">{{ .Username }}</a>{{ else }}<a href="{{ .ProfileURL }}">{{ .Username }}</a>{{ end }}</h3>
          <p class="meta">{{ if .JoinedAt }}joined {{ date .JoinedAt }}{{ end }}{{ if .Karma }} · karma {{ .Karma }}{{ end }}{{ if .HomepageURL }} · <a href="{{ .HomepageURL }}">site</a>{{ end }}</p>
        </article>
      {{ end }}
    </div>
  </main>
` + pageFoot

const postsTemplate = pageHead + `
  <main>
    <section class="page-title">
      <p class="eyebrow">Feed</p>
      <h1>All posts</h1>
      <p>{{ .Stats.Entries }} recent entries. Last built {{ datetime .Stats.GeneratedAt }}. <a href="/feed.xml">RSS</a></p>
    </section>
    <div class="entries">
      {{ range .Entries }}
        <article class="entry">
          <p class="meta">{{ entryDate . }} · <a href="{{ userPath .OwnerUsername }}">{{ .OwnerUsername }}</a>{{ if .FeedTitle }} · {{ .FeedTitle }}{{ end }}</p>
          <h2><a href="{{ .URL }}">{{ .Title }}</a></h2>
          {{ if .Summary }}<p>{{ .Summary }}</p>{{ end }}
        </article>
      {{ end }}
    </div>
    <nav class="pager">
      {{ if gt .Page 1 }}<a href="{{ prevPostPath .Page }}">Previous</a>{{ end }}
      <span>Page {{ .Page }} of {{ .Pages }}</span>
      {{ if lt .Page .Pages }}<a href="{{ nextPostPath .Page }}">Next</a>{{ end }}
    </nav>
  </main>
` + pageFoot

const analyticsTemplate = pageHead + `
  <main>
    <section class="page-title">
      <p class="eyebrow">Analytics</p>
      <h1>Dataset notes</h1>
      <p>Based on crawled profiles and discovered feeds, not the full Lobste.rs database. Last built {{ datetime .Analytics.GeneratedAt }}. <a href="/data/analytics.json">JSON</a></p>
    </section>

    <section class="metrics" aria-label="analytics overview">
      <div><strong>{{ .Analytics.Overview.Users }}</strong><span>users</span></div>
      <div><strong>{{ .Analytics.Overview.CrawledProfiles }}</strong><span>crawled profiles</span></div>
      <div><strong>{{ .Analytics.Overview.Homepages }}</strong><span>homepages</span></div>
      <div><strong>{{ .Analytics.Overview.UsersWithFeeds }}</strong><span>users with feeds</span></div>
      <div><strong>{{ .Analytics.Overview.Feeds }}</strong><span>feeds with posts</span></div>
      <div><strong>{{ .Analytics.Overview.Entries }}</strong><span>entries</span></div>
    </section>

    <section class="analytics-section">
      <h2>Oldest known users</h2>
      <table>
        <thead><tr><th>User</th><th>Joined</th><th>Karma</th><th>Site</th></tr></thead>
        <tbody>{{ range .Analytics.OldestUsers }}<tr><td><a href="{{ userPath .Username }}">{{ .Username }}</a></td><td>{{ date .JoinedAt }}</td><td>{{ if .Karma }}{{ .Karma }}{{ end }}</td><td>{{ if .HomepageURL }}<a href="{{ .HomepageURL }}">site</a>{{ end }}</td></tr>{{ end }}</tbody>
      </table>
    </section>

    <section class="analytics-grid">
      <section class="analytics-section">
        <h2>Highest karma</h2>
        <ol class="rank-list">{{ range .Analytics.TopKarma }}<li><a href="{{ userPath .Username }}">{{ .Username }}</a> <span>{{ .Karma }}</span></li>{{ end }}</ol>
      </section>
      <section class="analytics-section">
        <h2>Most stories</h2>
        <ol class="rank-list">{{ range .Analytics.TopStories }}<li><a href="{{ userPath .Username }}">{{ .Username }}</a> <span>{{ .StoriesSubmitted }}</span></li>{{ end }}</ol>
      </section>
      <section class="analytics-section">
        <h2>Most comments</h2>
        <ol class="rank-list">{{ range .Analytics.TopComments }}<li><a href="{{ userPath .Username }}">{{ .Username }}</a> <span>{{ .CommentsPosted }}</span></li>{{ end }}</ol>
      </section>
    </section>

    <section class="analytics-section">
      <h2>Most active discovered feeds</h2>
      <table>
        <thead><tr><th>User</th><th>Entries</th><th>Latest</th><th>Feed</th></tr></thead>
        <tbody>{{ range .Analytics.ActiveFeeds }}<tr><td><a href="{{ userPath .OwnerUsername }}">{{ .OwnerUsername }}</a></td><td>{{ .Entries }}</td><td>{{ date .LatestAt }}</td><td><a href="{{ .FeedURL }}">{{ if .FeedTitle }}{{ .FeedTitle }}{{ else }}feed{{ end }}</a></td></tr>{{ end }}</tbody>
      </table>
    </section>

    <section class="analytics-section">
      <h2>Recently active personal sites</h2>
      <div class="entries">
        {{ range .Analytics.RecentSites }}
          <article class="entry">
            <p class="meta">{{ date .LatestAt }} · <a href="{{ userPath .Username }}">{{ .Username }}</a>{{ if .FeedTitle }} · {{ .FeedTitle }}{{ end }}</p>
            <h3><a href="{{ .URL }}">{{ .Title }}</a></h3>
          </article>
        {{ end }}
      </div>
    </section>
  </main>
` + pageFoot

const usersTemplate = pageHead + `
  <main>
    <section class="page-title">
      <p class="eyebrow">Explore</p>
      <h1>Lobste.rs users</h1>
      <p>Users are ordered by their appearance in the Lobste.rs invitation tree. Profile pages are generated for users with crawled details. Last built {{ datetime .Stats.GeneratedAt }}.</p>
    </section>
    <div class="user-list">
      {{ range .Users }}
        <article class="user-card">
          <h2>{{ if .HasDetailPage }}<a href="{{ userPath .Username }}">{{ .Username }}</a>{{ else }}<a href="{{ .ProfileURL }}">{{ .Username }}</a>{{ end }}</h2>
          <p class="meta">{{ if .JoinedAt }}joined {{ date .JoinedAt }}{{ end }}{{ if .Karma }} · karma {{ .Karma }}{{ end }}{{ if .HomepageURL }} · <a href="{{ .HomepageURL }}">site</a>{{ end }} · <a href="{{ .ProfileURL }}">profile</a></p>
          {{ if .About }}<p>{{ .About }}</p>{{ end }}
        </article>
      {{ end }}
    </div>
    <nav class="pager">
      {{ if gt .Page 1 }}<a href="{{ prevPath .Page }}">Previous</a>{{ end }}
      <span>Page {{ .Page }} of {{ .Pages }}</span>
      {{ if lt .Page .Pages }}<a href="{{ nextPath .Page }}">Next</a>{{ end }}
    </nav>
  </main>
` + pageFoot

const userTemplate = pageHead + `
  <main>
    <section class="profile">
      <p class="eyebrow">User</p>
      <h1>{{ .User.Username }}</h1>
      <div class="profile-links">
        <a href="{{ .User.ProfileURL }}">Lobste.rs profile</a>
        {{ if .User.HomepageURL }}<a href="{{ .User.HomepageURL }}">Personal site</a>{{ end }}
      </div>
      <dl class="facts">
        {{ if .User.JoinedAt }}<div><dt>Joined</dt><dd>{{ date .User.JoinedAt }}</dd></div>{{ end }}
        {{ if .User.Karma }}<div><dt>Karma</dt><dd>{{ .User.Karma }}</dd></div>{{ end }}
        {{ if .User.StoriesSubmitted }}<div><dt>Stories</dt><dd>{{ .User.StoriesSubmitted }}</dd></div>{{ end }}
        {{ if .User.CommentsPosted }}<div><dt>Comments</dt><dd>{{ .User.CommentsPosted }}</dd></div>{{ end }}
        {{ if .User.ProfileCheckedAt }}<div><dt>Profile checked</dt><dd>{{ date .User.ProfileCheckedAt }}</dd></div>{{ end }}
        {{ if .User.FeedCheckedAt }}<div><dt>Feed refreshed</dt><dd>{{ date .User.FeedCheckedAt }}</dd></div>{{ end }}
      </dl>
      {{ if .User.About }}<p class="about">{{ .User.About }}</p>{{ end }}
    </section>

    <section class="section-head">
      <div>
        <p class="eyebrow">Recent</p>
        <h2>Posts from discovered feeds</h2>
      </div>
    </section>
    <div class="entries">
      {{ range .Entries }}
        <article class="entry">
          <p class="meta">{{ entryDate . }}{{ if .FeedTitle }} · {{ .FeedTitle }}{{ end }}</p>
          <h3><a href="{{ .URL }}">{{ .Title }}</a></h3>
          {{ if .Summary }}<p>{{ .Summary }}</p>{{ end }}
        </article>
      {{ else }}
        <p>No recent posts found for this user yet.</p>
      {{ end }}
    </div>
  </main>
` + pageFoot
