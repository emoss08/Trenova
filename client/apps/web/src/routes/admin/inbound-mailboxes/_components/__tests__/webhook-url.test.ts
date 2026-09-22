import { describe, expect, it } from "vitest";
import {
  POSTMARK_PASSWORD_LENGTH,
  generatePostmarkCredentials,
  postmarkSecret,
  webhookUrl,
} from "../webhook-url";

describe("webhookUrl", () => {
  const path = "/api/v1/webhooks/inbound-mail/Ab-_09/";

  it("puts the path on the API's own origin when the API lives elsewhere", () => {
    // Development: the page is on :5173 and the API on :8080. A provider must
    // be pointed at the API, not at the page it was copied from.
    expect(webhookUrl("http://localhost:8080/api/v1", "http://localhost:5173", path)).toBe(
      "http://localhost:8080/api/v1/webhooks/inbound-mail/Ab-_09/",
    );
  });

  it("uses the page's origin when the API is served beside it", () => {
    expect(webhookUrl("/api/v1", "https://tms.acme.com", path)).toBe(
      "https://tms.acme.com/api/v1/webhooks/inbound-mail/Ab-_09/",
    );
  });

  it("embeds Postmark's credentials in the URL, encoded", () => {
    expect(
      webhookUrl("/api/v1", "https://tms.acme.com", path, {
        user: "postmark",
        password: "p@ss:w/rd",
      }),
    ).toBe("https://postmark:p%40ss%3Aw%2Frd@tms.acme.com/api/v1/webhooks/inbound-mail/Ab-_09/");
  });
});

describe("generatePostmarkCredentials", () => {
  it("makes a password long enough for the server to accept, from URL-safe characters", () => {
    const { user, password } = generatePostmarkCredentials();

    expect(user).toBe("postmark");
    expect(password).toHaveLength(POSTMARK_PASSWORD_LENGTH);
    expect(POSTMARK_PASSWORD_LENGTH).toBeGreaterThanOrEqual(16);
    expect(password).toMatch(/^[A-Za-z0-9]+$/);
  });

  it("draws every character from the random source rather than a fixed pattern", () => {
    let calls = 0;
    const counting = (buffer: Uint8Array<ArrayBuffer>) => {
      calls++;
      buffer.fill(0);
      return buffer;
    };

    const { password } = generatePostmarkCredentials(counting);
    expect(calls).toBeGreaterThan(0);
    expect(password).toBe("A".repeat(POSTMARK_PASSWORD_LENGTH));
  });

  it("is written back as the user:password pair the server stores", () => {
    expect(postmarkSecret({ user: "postmark", password: "abc" })).toBe("postmark:abc");
  });
});
