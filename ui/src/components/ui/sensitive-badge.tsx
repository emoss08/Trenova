
export function SensitiveBadge() {
  return (
    <span
      title="Field is sensitive and has been masked."
      className="ml-2 text-xs px-1.5 py-0.5 bg-amber-100 dark:bg-amber-900/40 text-amber-700 dark:text-amber-400 rounded-sm font-medium select-none"
    >
      Sensitive
    </span>
  );
}
