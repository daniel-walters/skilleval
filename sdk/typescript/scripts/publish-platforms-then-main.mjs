#!/usr/bin/env node
/**
 * Publish platform packages then the main package (OIDC / trusted publishing).
 * Usage: node scripts/publish-platforms-then-main.mjs
 * Expects versions already synced and binaries already built.
 */
import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const sdkRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const manifest = JSON.parse(
  fs.readFileSync(path.join(sdkRoot, "platforms/manifest.json"), "utf8"),
);

function npmPublish(cwd) {
  const r = spawnSync("npm", ["publish", "--access", "public"], {
    cwd,
    stdio: "inherit",
    env: process.env,
  });
  if (r.status !== 0) {
    process.exit(r.status ?? 1);
  }
}

for (const p of manifest.packages) {
  const dir = path.join(sdkRoot, "platforms", p.npmSuffix);
  const binName = p.npmOs === "win32" ? "skilleval.exe" : "skilleval";
  const binPath = path.join(dir, "bin", binName);
  if (!fs.existsSync(binPath)) {
    console.error(`missing binary for publish: ${binPath}`);
    process.exit(1);
  }
  console.log(`publishing @danielwaltersdev/skilleval-${p.npmSuffix}`);
  npmPublish(dir);
}

console.log("publishing @danielwaltersdev/skilleval");
npmPublish(sdkRoot);
