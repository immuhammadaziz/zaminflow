import { copyFile, mkdir } from "node:fs/promises";
import { dirname } from "node:path";
import {
  backendContract,
  contractSnapshot,
  generatedTypes,
} from "./openapi-helpers.mjs";

const source = process.env.API_OPENAPI_PATH ?? backendContract;
await mkdir(dirname(contractSnapshot), { recursive: true });
await copyFile(source, contractSnapshot);
console.log(`Synchronized backend contract: ${source}`);
await import("./generate-client.mjs");
if (!generatedTypes) {
  throw new Error("Generated API types were not created.");
}
