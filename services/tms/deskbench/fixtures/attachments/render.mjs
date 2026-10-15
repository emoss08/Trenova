// Renders every attachment fixture from its HTML template and its variants through
// Gotenberg's Chromium route: node render.mjs [gotenberg url]
//   rate-confirmation.html + variants.json   the rate confirmations
//   edi-guide.html + edi-guides.json         a trading partner's EDI implementation guide
import { readFile, writeFile } from "node:fs/promises";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const gotenberg = process.argv[2] ?? "http://localhost:3009";
const families = [
  ["rate-confirmation.html", "variants.json"],
  ["edi-guide.html", "edi-guides.json"],
];

for (const [templateFile, variantsFile] of families) {
  const template = await readFile(join(here, templateFile), "utf8");
  const variants = JSON.parse(await readFile(join(here, variantsFile), "utf8"));

  const valuesOf = (name) => {
    const { extends: parent, ...own } = variants[name];
    return parent ? { ...valuesOf(parent), ...own } : own;
  };

  for (const name of Object.keys(variants)) {
    const values = valuesOf(name);
    const html = template.replace(/\{\{([A-Z0-9_]+)\}\}/g, (_, key) => {
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
}
