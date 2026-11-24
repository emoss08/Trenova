import { ArrowRightIcon } from "@trenova/shared/components/icons";
import { WorkerAvatar } from "@trenova/shared/components/worker-avatar";
import { useT } from "@trenova/shared/i18n/use-t";
import type { PersonPhoto } from "@trenova/shared/types/person-photo";

const SIZES = {
  sm: { face: "size-12", initials: "text-sm", variant: "thumbnail" },
  lg: { face: "size-20", initials: "text-lg", variant: "full" },
} as const;

/**
 * A photo request as the office decides it: the worker's photo now (or their initials, with none
 * yet) beside the one they sent, at the same size, so the two faces are compared side by side.
 */
export function PhotoComparison({
  firstName,
  lastName,
  current,
  proposed,
  size = "lg",
}: {
  firstName: string;
  lastName: string;
  current: PersonPhoto | null | undefined;
  proposed: PersonPhoto;
  size?: keyof typeof SIZES;
}) {
  const t = useT();
  const { face, initials, variant } = SIZES[size];
  return (
    <div className="flex items-center gap-3">
      <figure className="flex flex-col items-center gap-1.5">
        <WorkerAvatar
          firstName={firstName}
          lastName={lastName}
          photo={current}
          variant={variant}
          alt={t("Current photo")}
          className={face}
          fallbackClassName={initials}
        />
        <figcaption className="text-xs text-muted-foreground">
          {current ? t("Now") : t("No photo yet")}
        </figcaption>
      </figure>
      <ArrowRightIcon aria-hidden className="size-4 shrink-0 text-foreground-subtle" />
      <figure className="flex flex-col items-center gap-1.5">
        <WorkerAvatar
          firstName={firstName}
          lastName={lastName}
          photo={proposed}
          variant={variant}
          alt={t("Proposed photo")}
          className={face}
          fallbackClassName={initials}
        />
        <figcaption className="text-xs font-medium">{t("Proposed")}</figcaption>
      </figure>
    </div>
  );
}
