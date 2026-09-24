/**
 * A claim on an id that succeeds once per tab. What an assistant turn
 * announces can reach a tab more than once — a reply rejoined after a reload
 * replays from its start, and a conversation followed by a second reader sees
 * every event again — so anything that acts on an announcement claims it
 * first, and only the first claim acts.
 *
 * Kept in memory as well as in session storage, so a tab whose storage is
 * blocked still acts once.
 */
export function createOnceClaimer(storageKey: string, maxRemembered: number) {
  const claimedThisPage = new Set<string>();

  const readClaimed = (): string[] => {
    try {
      const raw = sessionStorage.getItem(storageKey);
      const parsed: unknown = raw ? JSON.parse(raw) : [];
      return Array.isArray(parsed)
        ? parsed.filter((entry): entry is string => typeof entry === "string")
        : [];
    } catch {
      return [];
    }
  };

  return (id: string): boolean => {
    if (claimedThisPage.has(id)) {
      return false;
    }
    claimedThisPage.add(id);

    const claimed = readClaimed();
    if (claimed.includes(id)) {
      return false;
    }
    try {
      sessionStorage.setItem(storageKey, JSON.stringify([...claimed, id].slice(-maxRemembered)));
    } catch {
      // Blocked storage leaves the in-page record, which is enough for this tab.
    }

    return true;
  };
}
