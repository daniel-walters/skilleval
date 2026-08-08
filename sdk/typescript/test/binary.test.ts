import assert from "node:assert/strict";
import path from "node:path";
import { afterEach, describe, it } from "node:test";

import {
  PLATFORM_PACKAGE_BY_HOST,
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

describe("packagedBinaryPath", () => {
  it("returns undefined for unsupported hosts", () => {
    assert.equal(packagedBinaryPath("aix", "ppc64"), undefined);
  });
});

describe("resolveSkillevalBinary", () => {
  const prev = process.env.SKILLEVAL_BIN;

  afterEach(() => {
    if (prev === undefined) {
      delete process.env.SKILLEVAL_BIN;
    } else {
      process.env.SKILLEVAL_BIN = prev;
    }
  });

  it("prefers SKILLEVAL_BIN when set", () => {
    process.env.SKILLEVAL_BIN = "/custom/skilleval";
    assert.equal(resolveSkillevalBinary(), "/custom/skilleval");
  });

  it("returns packaged path or PATH name when env unset", () => {
    delete process.env.SKILLEVAL_BIN;
    const resolved = resolveSkillevalBinary();
    assert.ok(resolved === "skilleval" || path.isAbsolute(resolved));
  });
});
