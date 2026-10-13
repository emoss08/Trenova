import { describe, expect, it } from "vitest";
import { emlFileName, messageEml, type EmlMessage } from "../eml";

const message: EmlMessage = {
  fromName: "Luis Ortega",
  fromAddress: "luis@drivers.example",
  toAddresses: ["pod@trenova.example"],
  ccAddresses: ["dispatch@trenova.example", "ops@trenova.example"],
  subject: "POD S-104233",
  receivedAt: Date.UTC(2026, 9, 7, 13, 39) / 1000,
  textBody: "Delivered.\nSigned by receiving.",
};

describe("messageEml", () => {
  it("writes the stored headers and words as a mail file with CRLF lines", () => {
    expect(messageEml(message)).toBe(
      [
        "From: Luis Ortega <luis@drivers.example>",
        "To: pod@trenova.example",
        "Cc: dispatch@trenova.example, ops@trenova.example",
        "Subject: POD S-104233",
        "Date: Wed, 07 Oct 2026 13:39:00 +0000",
        "MIME-Version: 1.0",
        "Content-Type: text/plain; charset=utf-8",
        "Content-Transfer-Encoding: 8bit",
        "",
        "Delivered.",
        "Signed by receiving.",
      ].join("\r\n"),
    );
  });

  it("leaves out an empty Cc and a missing name, and encodes words that are not ASCII", () => {
    const eml = messageEml({
      ...message,
      fromName: "",
      ccAddresses: [],
      subject: "Détention — S-1",
    });
    expect(eml).toContain("From: luis@drivers.example\r\n");
    expect(eml).not.toContain("Cc:");
    expect(eml).toContain("Subject: =?UTF-8?B?RMOpdGVudGlvbiDigJQgUy0x?=\r\n");
  });

  it("cannot be made to carry a header through a line break in a stored value", () => {
    const eml = messageEml({ ...message, subject: "Hi\r\nBcc: x@evil.example" });
    expect(eml.split("\r\n").filter((line) => line.startsWith("Bcc:"))).toEqual([]);
    expect(eml).toContain("Subject: Hi Bcc: x@evil.example\r\n");
  });
});

describe("emlFileName", () => {
  it("names the file after the subject, safe for any disk", () => {
    expect(emlFileName(message)).toBe("POD S-104233.eml");
    expect(emlFileName({ ...message, subject: 'a/b:c*?"<>|' })).toBe("a b c.eml");
    expect(emlFileName({ ...message, subject: "  " })).toBe("message.eml");
  });
});
