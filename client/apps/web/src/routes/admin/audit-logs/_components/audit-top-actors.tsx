import { HoverGlideGroup } from "@/components/hover-glide";
import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import type { AuditActorStat } from "@/lib/graphql/audit-overview";
import { BotIcon, ClockIcon, Key01Icon } from "@trenova/shared/components/icons";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { actorKindOfPrincipal, type AuditActorKind } from "./audit-entry-sentence";
import { ACTIVITY_SERIES, ACTOR_KIND_LABELS } from "./audit-overview-model";

const KIND_MARK: Record<
  Exclude<AuditActorKind, "people">,
  { icon: typeof BotIcon; className: string }
> = {
  agents: { icon: BotIcon, className: "bg-accent-violet-subtle text-accent-violet-on-subtle" },
  integrations: { icon: Key01Icon, className: "bg-brand-subtle text-brand-subtle-foreground" },
  jobs: { icon: ClockIcon, className: "bg-muted text-muted-foreground" },
};

/** A person's picture or initials; an agent, a key or a job by its mark. */
export function ActorMark({
  kind,
  name,
  user,
  size = "md",
}: {
  kind: AuditActorKind;
  name: string;
  user?: { id: string; profilePicUrl?: string | null; thumbnailUrl?: string | null } | null;
  size?: "md" | "lg";
}) {
  const box = size === "lg" ? "size-8" : "size-6";
  if (kind === "people") {
    return (
      <ResolvedUserAvatar
        userId={user?.id}
        name={name}
        profilePicUrl={user?.profilePicUrl ?? undefined}
        thumbnailUrl={user?.thumbnailUrl ?? undefined}
        className={cn("bg-muted shrink-0 rounded-md", box)}
        imageClassName="rounded-md bg-muted"
        fallbackClassName="text-xs"
        alt={name}
      />
    );
  }
  const mark = KIND_MARK[kind];
  const Icon = mark.icon;
  return (
    <span
      aria-hidden
      className={cn("grid shrink-0 place-items-center rounded-md", box, mark.className)}
    >
      <Icon className="size-3.5" />
    </span>
  );
}

/**
 * Who wrote the most entries in the window, each with how many were sensitive. Picking one
 * narrows the trail to theirs; picking it again lets it go.
 */
export function AuditTopActors({
  actors,
  otherActors,
  otherEntries,
  selectedId,
  onSelect,
}: {
  actors: readonly AuditActorStat[];
  otherActors: number;
  otherEntries: number;
  selectedId: string | null;
  onSelect: (actor: AuditActorStat | null) => void;
}) {
  const t = useT();
  const most = Math.max(...actors.map((actor) => actor.entries), 1);

  return (
    <section
      aria-label={t("Who made the changes")}
      className="bg-card flex min-w-0 flex-col rounded-md border"
    >
      <header className="px-3 pt-2.5">
        <h3 className="text-sm font-semibold">{t("Who made the changes")}</h3>
        <p className="text-muted-foreground text-xs">{t("Pick one to see only theirs")}</p>
      </header>
      {actors.length === 0 ? (
        <p className="text-muted-foreground px-3 py-4 text-xs">
          {t("Nobody wrote an entry you can read in this window.")}
        </p>
      ) : (
        <HoverGlideGroup className="bg-card px-1 py-1">
          <ul className="flex flex-col gap-px">
            {actors.map((actor) => {
              const kind = actorKindOfPrincipal(actor.principalType);
              const series = ACTIVITY_SERIES.find((entry) => entry.key === kind);
              const selected = actor.principalId === selectedId;
              const name = actor.name || actor.user?.name || t("Unknown");
              return (
                <li key={actor.principalId}>
                  <button
                    type="button"
                    data-glide-target
                    aria-pressed={selected}
                    onClick={() => onSelect(selected ? null : actor)}
                    className={cn(
                      "ui-focus-ring relative grid w-full grid-cols-[1.5rem_minmax(0,1fr)_auto] items-center gap-2 rounded-md px-2 py-1.5 text-left",
                      selected && "bg-brand-subtle",
                    )}
                  >
                    <ActorMark kind={kind} name={name} user={actor.user} />
                    <span className="min-w-0">
                      <span className="flex min-w-0 items-baseline gap-1.5">
                        <span className="truncate text-sm font-medium">{name}</span>
                        <span className="text-foreground-subtle shrink-0 text-xs">
                          {ACTOR_KIND_LABELS[kind]}
                        </span>
                      </span>
                      <span
                        aria-hidden
                        className="bg-muted mt-1 block h-0.75 overflow-hidden rounded-full"
                      >
                        <span
                          className={cn("block h-full rounded-full", series?.swatch)}
                          style={{ width: `${(actor.entries / most) * 100}%` }}
                        />
                      </span>
                    </span>
                    <span className="text-muted-foreground text-right text-xs tabular-nums">
                      {formatNumber(actor.entries)}
                      {actor.sensitive > 0 ? (
                        <span className="text-warning-foreground block">
                          {t("{0, plural, one {# sensitive} other {# sensitive}}", actor.sensitive)}
                        </span>
                      ) : null}
                    </span>
                  </button>
                </li>
              );
            })}
          </ul>
        </HoverGlideGroup>
      )}
      {otherActors > 0 ? (
        <p className="text-muted-foreground mx-3 mt-auto border-t py-2 text-xs">
          {t(
            "{0, plural, one {# more person, agent or key} other {# more people, agents and keys}} · {1, plural, one {# change} other {# changes}}",
            otherActors,
            otherEntries,
          )}
        </p>
      ) : null}
    </section>
  );
}
