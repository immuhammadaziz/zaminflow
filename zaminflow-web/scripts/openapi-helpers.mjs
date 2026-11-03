import { existsSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";

export const projectRoot = resolve(
  fileURLToPath(new URL("..", import.meta.url)),
);
export const monorepoBackendContract = resolve(
  projectRoot,
  "..",
  "docs",
  "openapi.yaml",
);
export const siblingBackendContract = resolve(
  projectRoot,
  "..",
  "zamin",
  "docs",
  "openapi.yaml",
);
export const contractSnapshot = resolve(
  projectRoot,
  "contracts",
  "openapi.yaml",
);
export const generatedTypes = resolve(
  projectRoot,
  "src",
  "lib",
  "api",
  "schema.d.ts",
);

export function resolveOpenApiSource() {
  const configured = process.env.API_OPENAPI_PATH;
  if (configured) {
    const path = resolve(projectRoot, configured);
    if (!existsSync(path)) {
      throw new Error(`API_OPENAPI_PATH does not exist: ${path}`);
    }
    return path;
  }
  if (existsSync(monorepoBackendContract)) {
    return monorepoBackendContract;
  }
  if (existsSync(siblingBackendContract)) {
    return siblingBackendContract;
  }
  return contractSnapshot;
}