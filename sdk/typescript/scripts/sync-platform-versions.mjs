#!/usr/bin/env node
/**
 * Set version on main + platform packages and align optionalDependencies.
 * Usage: node scripts/sync-platform-versions.mjs <version>
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const sdkRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const version = process.argv[2];
if (!version) {
  console.error("usage: sync-platform-versions.mjs <version>");
  process.exit(1);
}

const manifest = JSON.parse(
  fs.readFileSync(path.join(sdkRoot, "platforms/manifest.json"), "utf8"),
);

const optionalDependencies = {};
for (const p of manifest.packages) {
  const name = `@danielwaltersdev/skilleval-${p.npmSuffix}`;
  optionalDependencies[name] = version;
  const pkgPath = path.join(sdkRoot, "platforms", p.npmSuffix, "package.json");
  const pkg = JSON.parse(fs.readFileSync(pkgPath, "utf8"));
  pkg.version = version;
  fs.writeFileSync(pkgPath, JSON.stringify(pkg, null, 2) + "\n");
}

const mainPath = path.join(sdkRoot, "package.json");
const main = JSON.parse(fs.readFileSync(mainPath, "utf8"));
main.version = version;
main.optionalDependencies = optionalDependencies;
fs.writeFileSync(mainPath, JSON.stringify(main, null, 2) + "\n");

console.log(`synced version ${version} across main + ${manifest.packages.length} platforms`);
