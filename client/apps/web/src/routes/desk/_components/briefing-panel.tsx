import type { Briefing, BriefingSection } from "@/lib/graphql/briefing";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { ArrowRightIcon } from "lucide-react";
import { Link } from "react-router";

/**
 * The morning's page, under the dateline.
 *
 * Every figure here was computed before anything was written, and the
 * sentence a person reads is either the model's — checked against those
 * figures — or the computed wording it failed to beat. That is why the
 * numbers are links: each one opens the page that holds the rows it counted,
 * so "four uncovered moves" is one click from the four moves, and a figure
 * nobody can trace does not survive this screen.
 *
 * A briefing that nobody narrated is a complete briefing that reads plainer,
 * so nothing here is conditional on the model having run.
 */
export function BriefingPanel({ briefing }: { briefing: Briefing }) {
  const t = useT();
  const sections = briefing.sections.filter(
    (section) => section.read !== "" || section.items.length > 0,
  );

  if (sections.length === 0) {
    return null;
  }

  return (
    <section className="flex flex-col gap-4">
      <div className="flex items-center gap-2">
        <h2 className="text-muted-foreground text-xs font-medium">{t("Your day")}</h2>
        {briefing.narrated && (
          <span
            className="text-muted-foreground inline-flex items-center gap-1 text-xs"
            title={t("Some of this wording was written by a model and checked against the figures")}
          >
            <AssistMark className="size-3" />
          </span>
        )}
      </div>

      <div className="border-desk-hairline rounded-surface divide-desk-hairline divide-y border">
        {sections.map((section, index) => (
          <SectionRow key={section.key} section={section} index={index} />
        ))}
      </div>
    </section>
  );
}

function SectionRow({ section, index }: { section: BriefingSection; index: number }) {
  return (
    <div
      style={{ animationDelay: `${Math.min(index, 8) * 35}ms` }}
      className="animate-land flex flex-col gap-2 px-4 py-3"
    >
      <div className="flex items-baseline gap-2">
        <h3 className="text-sm font-medium">{section.title}</h3>
        {section.path && (
          <Link
            to={section.path}
            className="ui-focus-ring text-muted-foreground hover:text-foreground ml-auto shrink-0 rounded-md text-xs transition-colors"
          >
            <ArrowRightIcon className="size-3.5" />
          </Link>
        )}
      </div>

      {section.read !== "" && (
        <p className="text-muted-foreground text-sm leading-relaxed">{section.read}</p>
      )}

      {section.items.length > 0 && (
        <dl className="flex flex-wrap gap-x-6 gap-y-1.5 pt-0.5">
          {section.items.map((item) => (
            <div key={item.label} className="flex items-baseline gap-1.5">
              <dt className="text-muted-foreground text-xs">{item.label}</dt>
              <dd className="text-sm tabular-nums">
                {item.path ? (
                  <Link
                    to={item.path}
                    className={cn(
                      "ui-focus-ring rounded-sm underline decoration-transparent underline-offset-2",
                      "hover:decoration-current transition-colors",
                    )}
                  >
                    {item.value}
                  </Link>
                ) : (
                  item.value
                )}
              </dd>
            </div>
          ))}
        </dl>
      )}
    </div>
  );
}
