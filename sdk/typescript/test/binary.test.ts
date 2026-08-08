import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { afterEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";

import {
  PLATFORM_PACKAGE_BY_HOST,
  missingBinaryHint,
  packagedBinaryPath,
  platformPackageForHost,
  resolveSkillevalBinary,
} from "../src/binary.js";

describe("platformPackageForHost", () => {
  it("maps known Node hosts to scoped packages", () => {
    assert.equal(
      platformPackageForHost("darwin", "arm64"),
      "@danielwaltersdev/skilleval-darwin-arm64",
    );
    assert.equal(
      platformPackageForHost("linux", "x64"),
      "@danielwaltersdev/skilleval-linux-x64",
    );
    assert.equal(
      platformPackageForHost("win32", "x64"),
      "@danielwaltersdev/skilleval-win32-x64",
    );
  });

  it("returns undefined for unknown hosts", () => {
    assert.equal(platformPackageForHost("aix", "ppc64"), undefined);
  });

  it("covers every entry in PLATFORM_PACKAGE_BY_HOST", () => {
    for (const [host, pkg] of Object.entries(PLATFORM_PACKAGE_BY_HOST)) {
      const [platform, arch] = host.split(" ");
      assert.equal(platformPackageForHost(platform!, arch!), pkg);
    }
  });
});

// Serial: incomplete-package fixture mutates node_modules visible to createRequire.
describe("binary resolution", { concurrency: false }, () => {
  it("packagedBinaryPath returns undefined for unsupported hosts", () => {
    assert.equal(packagedBinaryPath("aix", "ppc64"), undefined);
  });

  it("prefers SKILLEVAL_BIN when set", () => {
    const prev = process.env.SKILLEVAL_BIN;
    process.env.SKILLEVAL_BIN = "/custom/skilleval";
    try {
      assert.equal(resolveSkillevalBinary(), "/custom/skilleval");
    } finally {
      if (prev === undefined) {
        delete process.env.SKILLEVAL_BIN;
      } else {
        process.env.SKILLEVAL_BIN = prev;
      }
    }
  });

  it("returns packaged path or PATH name when env unset", () => {
    const prev = process.env.SKILLEVAL_BIN;
    delete process.env.SKILLEVAL_BIN;
    try {
      const resolved = resolveSkillevalBinary();
      assert.ok(resolved === "skilleval" || path.isAbsolute(resolved));
    } finally {
      if (prev === undefined) {
        delete process.env.SKILLEVAL_BIN;
      } else {
        process.env.SKILLEVAL_BIN = prev;
      }
    }
  });

  it("throws when the platform package is installed without a binary", () => {
    const pkg = platformPackageForHost();
    if (!pkg) {
      return;
    }
    // Test file lives at dist-test/test/; createRequire(binary) walks dist-test/node_modules.
    const distTestRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
    const pkgDir = path.join(distTestRoot, "node_modules", ...pkg.split("/"));
    fs.mkdirSync(path.join(pkgDir, "bin"), { recursive: true });
    fs.writeFileSync(path.join(pkgDir, "package.json"), JSON.stringify({ name: pkg, version: "0.0.0" }));
    const prevBin = process.env.SKILLEVAL_BIN;
    delete process.env.SKILLEVAL_BIN;
    try {
      assert.throws(() => packagedBinaryPath(), /is installed but .* is missing/);
      assert.throws(() => resolveSkillevalBinary(), /is installed but .* is missing/);
    } finally {
      fs.rmSync(pkgDir, { recursive: true, force: true });
      if (prevBin === undefined) {
        delete process.env.SKILLEVAL_BIN;
      } else {
        process.env.SKILLEVAL_BIN = prevBin;
      }
    }
  });
});

describe("missingBinaryHint", () => {
  const prev = process.env.SKILLEVAL_BIN;

  afterEach(() => {
    if (prev === undefined) {
      delete process.env.SKILLEVAL_BIN;
    } else {
      process.env.SKILLEVAL_BIN = prev;
    }
  });

  it("points at SKILLEVAL_BIN when that path was used", () => {
    process.env.SKILLEVAL_BIN = "/no/such/skilleval";
    const hint = missingBinaryHint("/no/such/skilleval");
    assert.match(hint, /SKILLEVAL_BIN=/);
    assert.doesNotMatch(hint, /optional dependency/);
  });

  it("mentions optional dependency when falling back to PATH name", () => {
    delete process.env.SKILLEVAL_BIN;
    const pkg = platformPackageForHost();
    if (!pkg) {
      const hint = missingBinaryHint("skilleval");
      assert.match(hint, /unsupported platform/);
      return;
    }
    const hint = missingBinaryHint("skilleval");
    assert.match(hint, /optional dependency/);
    assert.match(hint, new RegExp(pkg.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")));
  });
});
