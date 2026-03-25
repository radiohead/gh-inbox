# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in gh-inbox, please report it
responsibly through
[GitHub Security Advisories](https://github.com/radiohead/gh-inbox/security/advisories/new).

**Do not open a public issue for security vulnerabilities.**

You should receive a response within 7 days. If the vulnerability is confirmed,
a fix will be released as soon as possible.

## Scope

gh-inbox is a CLI tool that uses your existing `gh` authentication token to
query the GitHub API. It does not store credentials — it delegates
authentication entirely to the `gh` CLI (`gh auth token`).

Cached data (team membership) is stored locally in `~/.cache/gh-inbox/` with
user-only permissions.
