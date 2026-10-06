import { isPasswordProtectedPdf } from "../desk-attachments";
import { describe, expect, it } from "vitest";

function pdf(trailer: string, padding = 0): Blob {
  return new Blob([
    "%PDF-1.7\n1 0 obj\n<< /Type /Catalog >>\nendobj\n",
    "x".repeat(padding),
    `trailer\n${trailer}\n%%EOF\n`,
  ]);
}

describe("isPasswordProtectedPdf", () => {
  it("finds the encryption entry in the trailer", async () => {
    await expect(isPasswordProtectedPdf(pdf("<< /Root 1 0 R /Encrypt 5 0 R >>"))).resolves.toBe(
      true,
    );
  });

  it("finds it at the end of a large file without reading the middle", async () => {
    await expect(
      isPasswordProtectedPdf(pdf("<< /Root 1 0 R /Encrypt 9 0 R >>", 300_000)),
    ).resolves.toBe(true);
  });

  it("leaves an unlocked PDF alone", async () => {
    await expect(isPasswordProtectedPdf(pdf("<< /Root 1 0 R /Size 6 >>"))).resolves.toBe(false);
  });

  it("is not fooled by the word in a page's text", async () => {
    await expect(
      isPasswordProtectedPdf(new Blob(["%PDF-1.7\n(see /Encrypt docs) Tj\ntrailer << >>"])),
    ).resolves.toBe(false);
  });
});
