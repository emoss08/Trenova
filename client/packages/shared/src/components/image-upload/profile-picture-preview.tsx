import { useT } from "@trenova/shared/i18n/use-t";
import { CropResult } from "./crop-result";

/**
 * A person's photo where it is drawn, at each place's own size: at the head of their profile,
 * in a row of people, and beside a mention.
 */
export function ProfilePicturePreview({
  name,
  detail,
  handle,
}: {
  name: string;
  /** What sits under the name on a profile, such as a job title. */
  detail?: string;
  /** How the person is mentioned, without the at sign. */
  handle?: string;
}) {
  const t = useT();

  return (
    <section
      aria-label={t("Preview")}
      className="flex min-w-0 flex-1 flex-col overflow-hidden rounded-lg border bg-sunken"
    >
      <header className="flex h-11 shrink-0 items-center border-b px-3 text-xs font-medium text-foreground-subtle">
        {t("Preview")}
      </header>
      <div aria-hidden className="flex flex-1 flex-col justify-center gap-5 p-5">
        <div className="flex items-center gap-4">
          <CropResult className="size-16 shrink-0 rounded-full object-cover" />
          <div className="min-w-0">
            <div className="truncate text-lg font-medium tracking-tight">{name}</div>
            {detail ? (
              <div className="truncate text-sm text-foreground-subtle">{detail}</div>
            ) : null}
          </div>
        </div>
        <div className="flex items-center gap-3 rounded-md border bg-card px-3 py-2">
          <CropResult className="size-9 shrink-0 rounded-full object-cover" />
          <div className="min-w-0 flex-1">
            <div className="truncate text-sm font-medium">{name}</div>
            <span className="mt-1 block h-2 w-28 rounded-full bg-surface-active" />
          </div>
        </div>
        <div className="flex items-center gap-2 text-sm">
          <CropResult className="size-6 shrink-0 rounded-full object-cover" />
          <span className="font-medium">{handle ? `@${handle}` : name}</span>
          <span className="h-2 w-24 rounded-full bg-surface-active" />
        </div>
      </div>
    </section>
  );
}
