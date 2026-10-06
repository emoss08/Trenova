import { BADGE_ACCENTS, type BadgeAccent } from "@trenova/shared/types/badge";

/* A person or a company drawn as an avatar takes one of the categorical
   accents, chosen by its id, so the same driver is the same colour on every
   screen and no avatar reaches outside the token set. The classes are written
   out whole so Tailwind can see them. */
const ACCENT_FILL: Record<BadgeAccent, string> = {
  "accent-indigo": "bg-accent-indigo-subtle text-accent-indigo-on-subtle",
  "accent-teal": "bg-accent-teal-subtle text-accent-teal-on-subtle",
  "accent-amber": "bg-accent-amber-subtle text-accent-amber-on-subtle",
  "accent-rose": "bg-accent-rose-subtle text-accent-rose-on-subtle",
  "accent-emerald": "bg-accent-emerald-subtle text-accent-emerald-on-subtle",
  "accent-sky": "bg-accent-sky-subtle text-accent-sky-on-subtle",
  "accent-violet": "bg-accent-violet-subtle text-accent-violet-on-subtle",
  "accent-slate": "bg-accent-slate-subtle text-accent-slate-on-subtle",
};

const FNV_OFFSET = 0x811c9dc5;
const FNV_PRIME = 0x01000193;

function fnv1a(key: string): number {
  let hash = FNV_OFFSET;
  for (let i = 0; i < key.length; i++) {
    hash ^= key.charCodeAt(i);
    hash = Math.imul(hash, FNV_PRIME);
  }
  return hash >>> 0;
}

export function identityAccent(key: string): BadgeAccent {
  return BADGE_ACCENTS[fnv1a(key) % BADGE_ACCENTS.length];
}

export function identityAccentClass(key: string): string {
  return ACCENT_FILL[identityAccent(key)];
}
