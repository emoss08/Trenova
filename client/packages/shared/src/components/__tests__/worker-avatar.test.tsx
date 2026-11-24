import { cleanup, render } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { WorkerAvatar, personPhotoSource } from "../worker-avatar";

afterEach(cleanup);

const PHOTO = {
  url: "https://files.example/wrk_1/photo.jpg?sig=a",
  thumbnailUrl: "https://files.example/wrk_1/thumb.jpg?sig=a",
};

describe("personPhotoSource", () => {
  it("takes the thumbnail for a small face and the full picture for a large one", () => {
    expect(personPhotoSource(PHOTO)).toBe(PHOTO.thumbnailUrl);
    expect(personPhotoSource(PHOTO, "full")).toBe(PHOTO.url);
  });

  it("falls back to the other link when one is empty, and to nothing without a photo", () => {
    expect(personPhotoSource({ url: PHOTO.url, thumbnailUrl: "" })).toBe(PHOTO.url);
    expect(personPhotoSource({ url: "", thumbnailUrl: PHOTO.thumbnailUrl }, "full")).toBe(
      PHOTO.thumbnailUrl,
    );
    expect(personPhotoSource(null)).toBeUndefined();
    expect(personPhotoSource({ url: "", thumbnailUrl: "" })).toBeUndefined();
  });
});

describe("WorkerAvatar", () => {
  it("draws the worker's initials under a lazy photo", () => {
    const { container } = render(
      <WorkerAvatar firstName="Maria" lastName="Alvarez" photo={PHOTO} />,
    );

    expect(container.textContent).toBe("MA");
    expect(container.querySelector("img")?.getAttribute("src")).toBe(PHOTO.thumbnailUrl);
    expect(container.querySelector("img")?.getAttribute("loading")).toBe("lazy");
  });

  it("takes the initials from the whole name when the read carries no first and last name", () => {
    const { container } = render(<WorkerAvatar name="Dana Ruiz" photo={null} />);

    expect(container.textContent).toBe("DR");
  });

  it("draws the initials alone for a worker with no photo", () => {
    const { container } = render(
      <WorkerAvatar firstName="Maria" lastName="Alvarez" photo={null} />,
    );

    expect(container.querySelector("img")).toBeNull();
    expect(container.textContent).toBe("MA");
  });
});
