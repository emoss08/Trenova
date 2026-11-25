import { QueryClient, QueryObserver } from "@tanstack/react-query";
import { afterEach, describe, expect, it, vi } from "vitest";
import { reportExpiredPhoto, resetExpiredPhotos } from "../photo-expiry";
import { holdsLink, registerPhotoRefresh } from "../photo-refresh";

afterEach(() => {
  resetExpiredPhotos();
});

const LINK = "https://files.example/wrk_1/thumb.jpg?sig=a";

describe("holdsLink", () => {
  it("finds a link anywhere inside a read: a page of rows, a nested record", () => {
    const page = { pages: [{ rows: [{ worker: { photo: { url: "x", thumbnailUrl: LINK } } }] }] };
    expect(holdsLink(page, LINK)).toBe(true);
    expect(holdsLink({ rows: [{ photo: null }] }, LINK)).toBe(false);
    expect(holdsLink(undefined, LINK)).toBe(false);
  });
});

describe("registerPhotoRefresh", () => {
  it("reads again the queries on screen that carried an expired link, and no others", async () => {
    const client = new QueryClient();
    const holding = vi.fn(async () => ({ photo: { url: "u", thumbnailUrl: LINK } }));
    const other = vi.fn(async () => ({ photo: null }));
    const observers = [
      new QueryObserver(client, { queryKey: ["holding"], queryFn: holding }),
      new QueryObserver(client, { queryKey: ["other"], queryFn: other }),
    ];
    const unsubscribe = observers.map((observer) => observer.subscribe(() => undefined));
    await vi.waitFor(() => expect(holding).toHaveBeenCalledTimes(1));
    await vi.waitFor(() => expect(other).toHaveBeenCalledTimes(1));
    const remove = registerPhotoRefresh(client);

    reportExpiredPhoto(LINK);
    reportExpiredPhoto(LINK);

    await vi.waitFor(() => expect(holding).toHaveBeenCalledTimes(2));
    expect(other).toHaveBeenCalledTimes(1);
    remove();
    unsubscribe.forEach((stop) => stop());
  });
});
