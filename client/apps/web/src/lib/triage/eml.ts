/**
 * A message as a mail file a person can keep or open in a mail client: the
 * headers and the plain text the server stores for it. The provider's own
 * copy is kept in object storage and no route serves it, so this is what
 * "Download as .eml" hands over.
 */
export type EmlMessage = {
  fromName: string;
  fromAddress: string;
  toAddresses: readonly string[];
  ccAddresses: readonly string[];
  subject: string;
  receivedAt: number;
  textBody: string;
};

const CRLF = "\r\n";
const LINE_BREAKS = /[\r\n]+/g;
const NOT_ASCII = /[^\x20-\x7e]/;
const UNSAFE_FILE_CHARACTERS = /[\\/:*?"<>|]+/g;
const FIRST_PRINTABLE = 0x20;

/** The text without control characters, which no file name may hold. */
function withoutControls(value: string): string {
  let clean = "";
  for (let index = 0; index < value.length; index += 1) {
    clean += value.charCodeAt(index) < FIRST_PRINTABLE ? " " : value[index];
  }

  return clean;
}

/** A header value on one line: a stored value can never start a header of its own. */
function oneLine(value: string): string {
  return value.replace(LINE_BREAKS, " ").trim();
}

/** RFC 2047 encoded-word for anything outside printable ASCII. */
function encodeWord(value: string): string {
  if (!NOT_ASCII.test(value)) {
    return value;
  }
  const bytes = new TextEncoder().encode(value);
  let binary = "";
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }

  return `=?UTF-8?B?${btoa(binary)}?=`;
}

function mailbox(name: string, address: string): string {
  const cleanName = oneLine(name);
  const cleanAddress = oneLine(address);

  return cleanName === "" ? cleanAddress : `${encodeWord(cleanName)} <${cleanAddress}>`;
}

function rfc2822Date(seconds: number): string {
  return new Date(seconds * 1000).toUTCString().replace(/GMT$/, "+0000");
}

/** One header line: a protocol name and its value, never interface text. */
function header(name: string, value: string): string {
  // i18n-ignore: an RFC 5322 header line is mail protocol, not interface text
  return `${name}: ${value}`;
}

export function messageEml(message: EmlMessage): string {
  const headers = [
    header("From", mailbox(message.fromName, message.fromAddress)),
    header("To", message.toAddresses.map(oneLine).join(", ")),
    ...(message.ccAddresses.length === 0
      ? []
      : [header("Cc", message.ccAddresses.map(oneLine).join(", "))]),
    header("Subject", encodeWord(oneLine(message.subject))),
    header("Date", rfc2822Date(message.receivedAt)),
    header("MIME-Version", "1.0"),
    header("Content-Type", "text/plain; charset=utf-8"),
    header("Content-Transfer-Encoding", "8bit"),
  ];
  const body = message.textBody.replace(/\r\n?/g, "\n").split("\n").join(CRLF);

  return [...headers, "", body].join(CRLF);
}

/** The file's name: the subject, with what no disk accepts taken out. */
export function emlFileName(message: Pick<EmlMessage, "subject">): string {
  const base = withoutControls(message.subject)
    .replace(UNSAFE_FILE_CHARACTERS, " ")
    .replace(/\s+/g, " ")
    .trim();

  return `${base === "" ? "message" : base}.eml`;
}

/** Saves the message as a .eml file in the browser. */
export function downloadEml(message: EmlMessage): void {
  const blob = new Blob([messageEml(message)], { type: "message/rfc822" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = emlFileName(message);
  link.rel = "noopener";
  link.click();
  setTimeout(() => URL.revokeObjectURL(url), 0);
}
