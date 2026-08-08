#!/usr/bin/env node
/**
 * Cross-compile Go skilleval into each platforms/<name>/bin for npm packaging.
 * Usage: node scripts/build-platform-binaries.mjs <version>
 */
import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const sdkRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const repoRoot = path.resolve(sdkRoot, "../..");
const version = process.argv[2];
if (!version) {
  console.error("usage: build-platform-binaries.mjs <version>");
  process.exit(1);
}

const manifest = JSON.parse(
  fs.readFileSync(path.join(sdkRoot, "platforms/manifest.json"), "utf8"),
);

for (const p of manifest.packages) {
  const outDir = path.join(sdkRoot, "platforms", p.npmSuffix, "bin");
  fs.mkdirSync(outDir, { recursive: true });
  // Drop placeholders / stale binaries.
  for (const name of fs.readdirSync(outDir)) {
    fs.rmSync(path.join(outDir, name), { force: true });
  }
  const outName = p.npmOs === "win32" ? "skilleval.exe" : "skilleval";
  const out = path.join(outDir, outName);
  console.log(`building ${p.goos}/${p.goarch} → ${path.relative(sdkRoot, out)}`);
  const r = spawnSync(
    "go",
    [
      "build",
      "-ldflags",
      `-s -w -X main.version=${version}`,
      "-o",
      out,
      "./cmd/skilleval",
    ],
    {
      cwd: repoRoot,
      env: {
        ...process.env,
        CGO_ENABLED: "0",
        GOOS: p.goos,
        GOARCH: p.goarch,
      },
      stdio: "inherit",
    },
  );
  if (r.status !== 0) {
    process.exit(r.status ?? 1);
  }
  if (!fs.existsSync(out)) {
    console.error(`missing binary: ${out}`);
    process.exit(1);
  }
}

console.log("platform binaries ready");
