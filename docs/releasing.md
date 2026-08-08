# Releasing

CLI binaries (GitHub Releases) and the TypeScript package ([`@danielwaltersdev/skilleval`](https://www.npmjs.com/package/@danielwaltersdev/skilleval)) — including platform optionalDependencies that ship the Go CLI — publish from the **same** `v*` tag. One tag push runs both workflows.

## Happy path

1. Land the release on `main` (merged PR, green CI).
2. Tag and push from `main`:

```bash
git checkout main && git pull
git tag v0.1.3
git push origin v0.1.3
```

3. Confirm both Actions succeed:
   - [Release](../.github/workflows/release.yml) — GoReleaser → GitHub Release archives + checksums
   - [Publish npm](../.github/workflows/publish-npm.yml) — cross-compiles platform binaries, publishes six `@danielwaltersdev/skilleval-<os>-<arch>` packages, then the main package
4. Spot-check:

```bash
npm view @danielwaltersdev/skilleval version
npx --package=@danielwaltersdev/skilleval skilleval version
```

Do not bump `sdk/typescript/package.json` (or platform package versions) on `main` for the release itself — the Publish npm workflow syncs versions from the tag at publish time.

## Prerequisites (one-time)

Configure a **Trusted Publisher** on **each** npm package (main + every platform package) → **Settings** → **Trusted Publisher** → GitHub Actions:

| Field | Value |
| --- | --- |
| Organization or user | `daniel-walters` |
| Repository | `skilleval` |
| Workflow filename | `publish-npm.yml` |
| Environment name | *(empty)* |
| Allowed actions | `npm publish` |

Packages:

- `@danielwaltersdev/skilleval`
- `@danielwaltersdev/skilleval-darwin-arm64`
- `@danielwaltersdev/skilleval-darwin-x64`
- `@danielwaltersdev/skilleval-linux-arm64`
- `@danielwaltersdev/skilleval-linux-x64`
- `@danielwaltersdev/skilleval-win32-arm64`
- `@danielwaltersdev/skilleval-win32-x64`

The workflow already has `id-token: write`, Node 24, and no `NODE_AUTH_TOKEN`. No long-lived `NPM_TOKEN` is required.

**First-ever package name:** trusted publishing cannot create a brand-new package. For each new name, publish once locally (`npm login` + `npm publish --access public` from that package directory with a real binary in `bin/`), then add the trusted publisher above. Later tags use CI.

The repo is public. npm provenance attestations should attach on Trusted Publisher publishes when OIDC succeeds.

### First platform-binary release (local seed)

Until the six platform packages exist on npm, CI cannot attach Trusted Publishing to them. Seed once from a release tag checkout:

```bash
cd sdk/typescript
node scripts/build-platform-binaries.mjs 0.1.3
node scripts/sync-platform-versions.mjs 0.1.3
# publish each platforms/* then the main package (npm login required)
node scripts/publish-platforms-then-main.mjs
```

Then add Trusted Publishers for each package and use tag-push CI going forward.

## Backfill npm only

If a `v*` tag exists but npm never published (or failed), use **Actions → Publish npm → Run workflow** with the version input (e.g. `0.1.3`, no leading `v`). The job checks out `refs/tags/v<version>`, rebuilds platform binaries, and publishes.

## Failure notes

| Symptom | Likely cause |
| --- | --- |
| Publish `ENEEDAUTH` / 403 | Trusted Publisher missing on that package name, or mismatched org/repo/workflow; or empty registry `_authToken` (do not set `registry-url` on `setup-node`) |
| Publish “cannot publish over existing version” | Version already on npm — expected for a re-run; bump the tag for a new release |
| `npm view` 404 while search/UI show the package | Registry half-state — publish the next patch from a new tag |
| Main package installed but `skilleval` missing | Matching platform optionalDependency failed to install (unsupported OS/arch, or package not published yet); set `SKILLEVAL_BIN` or use a GitHub Release binary |
| GoReleaser OK, npm red | Fix npm / backfill that tag; CLI release is already public |

## Versioning

- **Tags:** semver. Patch releases (`v0.1.x`) are for install, docs, and correctness fixes — not new product surface. Larger bumps when product surface changes.
- **Schemas:** Eval YAML and Result JSON stay on `schemaVersion: 1` until a breaking contract change; that bumps the schema number rather than silently changing meaning.
- **Surfaces in sync:** One `v*` tag publishes the CLI (GitHub Release), platform npm packages, and `@danielwaltersdev/skilleval` together.
