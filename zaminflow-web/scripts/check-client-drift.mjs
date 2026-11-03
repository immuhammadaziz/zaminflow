import { readFile } from "node:fs/promises";
import openapiTS, { astToString } from "openapi-typescript";
import {
  contractSnapshot,
  generatedTypes,
  resolveOpenApiSource,
} from "./openapi-helpers.mjs";

const sourcePath = resolveOpenApiSource();
const source = await readFile(sourcePath, "utf8");
const snapshot = await readFile(contractSnapshot, "utf8");

if (source !== snapshot) {
  throw new Error(
    "The backend OpenAPI file changed. Run pnpm sync-openapi, then commit the snapshot and generated types.",
  );
}

const expected = astToString(await openapiTS(snapshot));
const actual = await readFile(generatedTypes, "utf8");
if (expected !== actual) {
  throw new Error(
    "Generated API types are out of date. Run pnpm generate-client and commit the result.",
  );
}

console.log("OpenAPI snapshot and generated TypeScript types are in sync.");
