# Releasing

CLI binaries (GitHub Releases) and the TypeScript package ([`@danielwaltersdev/skilleval`](https://www.npmjs.com/package/@danielwaltersdev/skilleval)) ship from the **same** `v*` tag. One tag push runs both workflows.

## Happy path

1. Land the release on `main` (merged PR, green CI).
2. Tag and push from `main`:

```bash
git checkout main && git pull
git tag v0.1.2
git push origin v0.1.2
```

3. Confirm both Actions succeed:
   - [Release](../.github/workflows/release.yml) — GoReleaser → GitHub Release archives + checksums
   - [Publish npm](../.github/workflows/publish-npm.yml) — OIDC publish of `sdk/typescript`
4. Spot-check versions match:

```bash
# release binary (after install)
skilleval version

npm view @danielwaltersdev/skilleval version
```

Do not bump `sdk/typescript/package.json` version on `main` for the release itself — the Publish npm workflow sets the npm version from the tag at publish time.

## Prerequisites (one-time)

On npm → `@danielwaltersdev/skilleval` → **Settings** → **Trusted Publisher** → GitHub Actions:

| Field | Value |
| --- | --- |
| Organization or user | `daniel-walters` |
| Repository | `skilleval` |
| Workflow filename | `publish-npm.yml` |
| Environment name | *(empty)* |
| Allowed actions | `npm publish` |

The workflow already has `id-token: write`, Node 24, and no `NODE_AUTH_TOKEN`. No long-lived `NPM_TOKEN` is required.

**First-ever package name:** trusted publishing cannot create a brand-new package. Publish once locally (`npm login` + `npm publish --access public` from `sdk/typescript` at the release tag), then add the trusted publisher above. Later tags use CI.

The GitHub repo is private, so npm will not attach provenance attestations even when OIDC publish succeeds.

## Backfill npm only

If a `v*` tag exists but npm never published (or failed), use **Actions → Publish npm → Run workflow** with the version input (e.g. `0.1.1`, no leading `v`). The job checks out `refs/tags/v<version>` so npm matches that tag’s tree, not whatever branch tip you selected in the UI.

## Failure notes

| Symptom | Likely cause |
| --- | --- |
| Publish `ENEEDAUTH` / 403 | Trusted Publisher missing or mismatched org/repo/workflow name; or an empty registry `_authToken` forcing token auth (do not set `registry-url` on `setup-node` in this workflow) |
| Publish “cannot publish over existing version” | Version already on npm — expected for a re-run; bump the tag for a new release |
| `npm view` 404 while search/UI show the package | Registry half-state after a broken first publish — publish the next patch version from a new tag |
| GoReleaser OK, npm red | Fix npm / backfill that tag; CLI release is already public |

## Versioning

- **Tags:** semver. Patch releases (`v0.1.x`) are for install, docs, and correctness fixes — not new product surface. Larger bumps when product surface changes.
- **Schemas:** Eval YAML and Result JSON stay on `schemaVersion: 1` until a breaking contract change; that bumps the schema number rather than silently changing meaning.
- **Surfaces in sync:** One `v*` tag publishes the CLI (GitHub Release) and `@danielwaltersdev/skilleval` (npm) together.
