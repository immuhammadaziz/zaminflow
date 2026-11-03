import { mkdir, readFile, writeFile } from "node:fs/promises";
import openapiTS, { astToString } from "openapi-typescript";
import {
  contractSnapshot,
  generatedTypes,
  resolveOpenApiSource,
} from "./openapi-helpers.mjs";

const sourcePath = resolveOpenApiSource();
const schema = await readFile(sourcePath, "utf8");
const generated = astToString(await openapiTS(schema));

if (sourcePath !== contractSnapshot) {
  await writeFile(contractSnapshot, schema);
}
await mkdir(new URL("../src/lib/api/", import.meta.url), { recursive: true });
await writeFile(generatedTypes, generated);
console.log(`Generated API types from ${sourcePath}`);
