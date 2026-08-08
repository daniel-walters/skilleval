import { createRequire } from "node:module";
import fs from "node:fs";
import path from "node:path";

/** npm package name prefix for platform optionalDependencies. */
export const PLATFORM_PACKAGE_SCOPE = "@danielwaltersdev/skilleval";

/**
 * Map Node `process.platform` + `process.arch` to the optionalDependency
 * package that ships the Go binary for that host.
 */
export const PLATFORM_PACKAGE_BY_HOST: Record<string, string> = {
  "darwin arm64": `${PLATFORM_PACKAGE_SCOPE}-darwin-arm64`,
  "darwin x64": `${PLATFORM_PACKAGE_SCOPE}-darwin-x64`,
  "linux arm64": `${PLATFORM_PACKAGE_SCOPE}-linux-arm64`,
  "linux x64": `${PLATFORM_PACKAGE_SCOPE}-linux-x64`,
  "win32 arm64": `${PLATFORM_PACKAGE_SCOPE}-win32-arm64`,
  "win32 x64": `${PLATFORM_PACKAGE_SCOPE}-win32-x64`,
};

export function platformPackageForHost(
  platform: string = process.platform,
  arch: string = process.arch,
): string | undefined {
  return PLATFORM_PACKAGE_BY_HOST[`${platform} ${arch}`];
}

function binaryFileName(platform: string = process.platform): string {
  return platform === "win32" ? "skilleval.exe" : "skilleval";
}

/**
 * Absolute path to the Go binary inside an installed platform package, or
 * undefined if that optionalDependency is missing / incomplete.
 */
export function packagedBinaryPath(
  platform: string = process.platform,
  arch: string = process.arch,
): string | undefined {
  const pkg = platformPackageForHost(platform, arch);
  if (!pkg) {
    return undefined;
  }
  try {
    const require = createRequire(import.meta.url);
    const pkgJson = require.resolve(`${pkg}/package.json`);
    const candidate = path.join(path.dirname(pkgJson), "bin", binaryFileName(platform));
    if (fs.existsSync(candidate)) {
      return candidate;
    }
  } catch {
    // optionalDependency not installed
  }
  return undefined;
}

/**
 * Resolve which `skilleval` binary to run.
 *
 * Order: `SKILLEVAL_BIN` → packaged platform optionalDependency → `skilleval` on PATH.
 */
export function resolveSkillevalBinary(): string {
  const fromEnv = process.env.SKILLEVAL_BIN?.trim();
  if (fromEnv) {
    return fromEnv;
  }
  const packaged = packagedBinaryPath();
  if (packaged) {
    return packaged;
  }
  return "skilleval";
}

/** Human-readable hint when spawn fails and no packaged binary was found. */
export function missingBinaryHint(): string {
  const pkg = platformPackageForHost();
  if (!pkg) {
    return (
      `unsupported platform ${process.platform}/${process.arch}; ` +
      `set SKILLEVAL_BIN to a skilleval binary, or install one from GitHub Releases`
    );
  }
  return (
    `optional dependency ${pkg} is not installed; ` +
    `reinstall @danielwaltersdev/skilleval, set SKILLEVAL_BIN, or put skilleval on PATH`
  );
}
