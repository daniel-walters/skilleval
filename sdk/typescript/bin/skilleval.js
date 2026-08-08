#!/usr/bin/env node
/**
 * npm bin entry: forwards argv to the Go skilleval binary
 * (packaged platform optionalDependency, SKILLEVAL_BIN, or PATH).
 */
import { spawn } from "node:child_process";

import { missingBinaryHint, resolveSkillevalBinary } from "../dist/binary.js";

const bin = resolveSkillevalBinary();
const args = process.argv.slice(2);

const child = spawn(bin, args, {
  stdio: "inherit",
  env: process.env,
  windowsHide: true,
});

child.on("error", (err) => {
  const detail = err instanceof Error ? err.message : String(err);
  console.error(`skilleval: failed to spawn ${bin}: ${detail} (${missingBinaryHint()})`);
  process.exit(1);
});

child.on("close", (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
    return;
  }
  process.exit(code ?? 1);
});
