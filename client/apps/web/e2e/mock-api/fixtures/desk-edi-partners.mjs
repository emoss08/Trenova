// The EDI trading partners the Desk cards thread names, as the DeskEdiPartnerCard
// query answers for them: the same fixtures the card's unit tests read, moved
// so their moments fall against the mock's now. Tyson trades by AS2 and is
// healthy; Acme's SFTP pickups are failing with a dead letter, late
// acknowledgments and a file set aside; Kroger is still being set up, in test;
// Lone Star is another Trenova organization traded with directly.
import acme from "../../../src/routes/desk/_components/artifacts/cards/edi-partner/__tests__/acme-trouble.json" with { type: "json" };
import kroger from "../../../src/routes/desk/_components/artifacts/cards/edi-partner/__tests__/kroger-setup.json" with { type: "json" };
import lonestar from "../../../src/routes/desk/_components/artifacts/cards/edi-partner/__tests__/lonestar-internal.json" with { type: "json" };
import tysonMessages from "../../../src/routes/desk/_components/artifacts/cards/edi-partner/__tests__/tyson-messages.json" with { type: "json" };
import tyson from "../../../src/routes/desk/_components/artifacts/cards/edi-partner/__tests__/tyson-trading.json" with { type: "json" };

/** Every Unix-second instant in a fixture moved by `delta`; the rest copied as it is. */
function moved(value, delta) {
  if (Array.isArray(value)) return value.map((entry) => moved(entry, delta));
  if (value === null || typeof value !== "object") return value;
  return Object.fromEntries(
    Object.entries(value).map(([key, entry]) => [
      key,
      /(At|Start|since|Since|notBefore|notAfter)$/.test(key) &&
      typeof entry === "number" &&
      entry > 0
        ? entry + delta
        : moved(entry, delta),
    ]),
  );
}

/** Each partner the cards thread names, with the messages its card lists newest first. */
export function deskEdiPartners(anchor) {
  const at = (fixture) => moved(fixture.card, anchor - fixture.now);
  const acmeCard = at(acme);
  return {
    edip_tyson: {
      card: at(tyson),
      messages: moved(tysonMessages.messages, anchor - tysonMessages.now),
    },
    edip_acme: {
      card: acmeCard,
      messages: [...acmeCard.attention.overdueAcks, ...acmeCard.attention.deadLettered].sort(
        (a, b) => b.generatedAt - a.generatedAt,
      ),
    },
    edip_kroger: { card: at(kroger), messages: [] },
    edip_lonestar_link: { card: at(lonestar), messages: [] },
  };
}

/** The cards thread's EDI partner conversations: each slug's partner, its title, and what was asked and said. */
export const DESK_EDI_PARTNER_CONVERSATIONS = {
  "partner-tyson": {
    recordId: "edip_tyson",
    title: "EDI partner Tyson Foods",
    question: "How is Tyson's EDI holding up?",
    reply:
      "Fine: 99.8% of what we sent them this week was delivered and their 997s come back in about six minutes. [Tyson Foods](artifact:art_partner-tyson) has the traffic and the line.",
  },
  "partner-acme": {
    recordId: "edip_acme",
    title: "EDI partner Acme Foods",
    question: "Why haven't we had a tender from Acme since this morning?",
    reply:
      "We can't sign in to their SFTP server, so their tenders are sitting there. There's also a dead-lettered invoice and three status updates waiting on a 997. [Acme Foods](artifact:art_partner-acme) lists each.",
  },
  "partner-kroger": {
    recordId: "edip_kroger",
    title: "EDI partner Kroger",
    question: "Where are we with Kroger's EDI setup?",
    reply:
      "Three of six checks pass: there's no transport, no mappings and no passing test case yet, and both sets are still in test. [Kroger](artifact:art_partner-kroger) shows what's left.",
  },
  "partner-lonestar": {
    recordId: "edip_lonestar_link",
    title: "EDI partner Lone Star Logistics",
    question: "Are we still connected to Lone Star?",
    reply:
      "Yes, the connection has been active since March and covers tenders both ways, status and invoices. [Lone Star Logistics](artifact:art_partner-lonestar) has it.",
  },
  "partner-gone": {
    recordId: "edip_globex",
    title: "EDI partner Globex",
    question: "Pull up Globex's EDI partner.",
    reply:
      "That partner has been removed since I looked it up. [Globex](artifact:art_partner-gone) is what I found then.",
  },
};
