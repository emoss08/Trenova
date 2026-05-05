export function formatElapsedTime(timestamp: number, now = Date.now()) {
  if (timestamp <= 0) return "just now";

  const elapsedSeconds = Math.max(0, Math.floor((now - timestamp) / 1000));
  if (elapsedSeconds < 1) return "just now";
  if (elapsedSeconds < 60) return `${elapsedSeconds}s ago`;

  return `${Math.floor(elapsedSeconds / 60)}m ago`;
}
