import { copyFile, mkdir } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const clientRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const source = resolve(clientRoot, "src/graphql/generated/persisted-documents.json");
const destination = resolve(
  clientRoot,
  "../services/tms/internal/api/graphql/persisted-documents.json",
);

await mkdir(dirname(destination), { recursive: true });
await copyFile(source, destination);
