export function formatElapsedTime(timestamp: number, now = Date.now()) {
  if (timestamp <= 0) return "just now";

  const elapsedSeconds = Math.max(0, Math.floor((now - timestamp) / 1000));
  if (elapsedSeconds < 1) return "just now";
  if (elapsedSeconds < 60) return `${elapsedSeconds}s ago`;

  return `${Math.floor(elapsedSeconds / 60)}m ago`;
}

export function formatTimeAgo(timestamp: number, now = Date.now()): string {
  const elapsedSeconds = Math.max(0, Math.floor((now - timestamp) / 1000));
  if (elapsedSeconds < 60) return "just now";
  if (elapsedSeconds < 3600) return `${Math.floor(elapsedSeconds / 60)}m ago`;
  if (elapsedSeconds < 86400) return `${Math.floor(elapsedSeconds / 3600)}h ago`;
  if (elapsedSeconds < 604800) return `${Math.floor(elapsedSeconds / 86400)}d ago`;
  return new Date(timestamp).toLocaleDateString();
}
