# docs

Design specs and implementation plans, kept for the record rather than as
reference material — the README and the package documentation are what to read.

These files are **not** part of the published site. The repository has a
`.nojekyll` marker at its root because GitHub's branch-based Pages build runs
Jekyll over every Markdown file it finds, and Liquid reads a Go map literal like
`{{"id": "a"}` inside a fenced code block as an unterminated template. The build
fails on a plan document nobody was trying to publish.

The site is built by `.github/workflows/pages.yml` from `scripts/build-pages.sh`,
which publishes only `dist/pages`. For that workflow to run at all, the
repository's **Settings → Pages → Source** must be **GitHub Actions**; on
"Deploy from a branch" GitHub ignores the workflow and serves the repository
through Jekyll instead.
