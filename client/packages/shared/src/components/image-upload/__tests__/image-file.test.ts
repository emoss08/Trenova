import { describe, expect, it } from "vitest";
import { imageRejection, offeredFile } from "../image-file";

function fileOf(name: string, type: string, bytes = 4): File {
  const file = new File(["data"], name, { type });
  Object.defineProperty(file, "size", { value: bytes });
  return file;
}

const TWENTY_MEGABYTES = 20 * 1024 * 1024;

describe("which files can be a picture", () => {
  it.each(["image/png", "image/jpeg", "image/webp", "image/svg+xml", "IMAGE/PNG"])(
    "takes %s",
    (type) => {
      expect(imageRejection(fileOf("mark", type))).toBeNull();
    },
  );

  it.each(["image/gif", "image/avif", "application/pdf", ""])("turns away %s", (type) => {
    expect(imageRejection(fileOf("mark", type))).toBe("type");
  });

  it("takes a picture of exactly 20 MB and turns away one byte more", () => {
    expect(imageRejection(fileOf("photo.jpg", "image/jpeg", TWENTY_MEGABYTES))).toBeNull();
    expect(imageRejection(fileOf("photo.jpg", "image/jpeg", TWENTY_MEGABYTES + 1))).toBe("size");
  });

  it("names the type, not the size, of a file that is wrong on both counts", () => {
    expect(imageRejection(fileOf("scan.pdf", "application/pdf", TWENTY_MEGABYTES + 1))).toBe(
      "type",
    );
  });
});

describe("which of several offered files is used", () => {
  it("is nothing when nothing was offered", () => {
    expect(offeredFile(null)).toBeNull();
    expect(offeredFile([])).toBeNull();
  });

  it("is the first picture, wherever it sits among the files", () => {
    const pdf = fileOf("rates.pdf", "application/pdf");
    const logo = fileOf("logo.png", "image/png");
    const photo = fileOf("photo.jpg", "image/jpeg");

    expect(offeredFile([pdf, logo, photo])).toBe(logo);
  });

  it("is the first file when none is a picture, so the person is told what was wrong", () => {
    const pdf = fileOf("rates.pdf", "application/pdf");
    const sheet = fileOf("rates.csv", "text/csv");

    expect(offeredFile([pdf, sheet])).toBe(pdf);
  });
});
