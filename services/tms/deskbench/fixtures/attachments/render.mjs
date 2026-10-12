// Renders every rate confirmation fixture from rate-confirmation.html and variants.json
// through Gotenberg's Chromium route: node render.mjs [gotenberg url]
import { readFile, writeFile } from "node:fs/promises";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const gotenberg = process.argv[2] ?? "http://localhost:3009";
const template = await readFile(join(here, "rate-confirmation.html"), "utf8");
const variants = JSON.parse(await readFile(join(here, "variants.json"), "utf8"));

function valuesOf(name) {
  const { extends: parent, ...own } = variants[name];
  return parent ? { ...valuesOf(parent), ...own } : own;
}

for (const name of Object.keys(variants)) {
  const values = valuesOf(name);
  const html = template.replace(/\{\{([A-Z_]+)\}\}/g, (_, key) => {
    if (!(key in values)) {
      throw new Error(`${name} gives no ${key}`);
    }
    return values[key];
  });
  const form = new FormData();
  form.append("files", new Blob([html], { type: "text/html" }), "index.html");
  const response = await fetch(`${gotenberg}/forms/chromium/convert/html`, {
    method: "POST",
    body: form,
  });
  if (!response.ok) {
    throw new Error(`${name}: Gotenberg answered ${response.status}`);
  }
  await writeFile(join(here, name), Buffer.from(await response.arrayBuffer()));
  console.log(`wrote ${name}`);
}
