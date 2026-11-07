import { useT } from "@trenova/shared/i18n/use-t";

export function ReachMapSkeleton() {
  const t = useT();
  return (
    <div
      aria-busy
      aria-label={t("Loading the map")}
      className="aspect-[2/1] w-full ui-shimmer rounded-lg"
    />
  );
}
