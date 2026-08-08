#!/usr/bin/env node
/**
 * npm bin entry: routes TypeScript/JavaScript evals (and discovery) in Node;
 * forwards YAML and other commands to the Go skilleval binary.
 */
import { main } from "../dist/cli.js";

const code = await main(process.argv.slice(2));
process.exit(code);
