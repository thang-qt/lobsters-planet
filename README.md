# Lobsters Planet

A small static planet for personal sites discovered from the [Lobste.rs](https://lobste.rs) users page.

It crawls Lobste.rs user profiles, finds personal homepages, discovers RSS/Atom feeds, refreshes recent posts, and generates a static site in `public/`.

Live site: <https://lobsters-planet.thangqt.com>

## What it generates

- `/` — latest posts and user preview
- `/posts/` — paginated recent posts
- `/users/` — paginated Lobste.rs user explorer
- `/users/<username>/` — user detail pages when profile data is available
- `/analytics/` — simple dataset analytics
- `/feed.xml` — combined RSS feed
- `/latest.json` and `/data/*.json` — public static datasets

## Local development

This repo includes a Nix flake and `justfile`.

```sh
direnv allow
just test
just build
just serve
```

Then open <http://localhost:8080>.

Useful commands:

```sh
just discover  # update Lobste.rs users/profile state
just feeds     # discover feeds from personal sites
just refresh   # refresh discovered feeds
just build     # generate public/
```

## Deployment

`public/` is committed and can be deployed directly by Cloudflare Pages.

Cloudflare Pages settings:

- Production branch: `main`
- Build command: empty
- Build output directory: `public`

GitHub Actions periodically refreshes crawler state and rebuilds `public/`; Cloudflare Pages deploys the committed output.

## Notes

Analytics and rankings are based only on crawled profiles and discovered feeds, not the full Lobste.rs database.

Feed summaries are converted to plain text before rendering. The site is static HTML/CSS with no frontend JavaScript.
