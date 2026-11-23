import { Link } from "@trenova/shared/components/link";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { useT } from "@trenova/shared/i18n/use-t";
import type { WorkerHosLink } from "@trenova/graphql/generated/graphql";

/** A faint paper log: four empty lanes under an hour row. */
function LogSketch() {
  return (
    <div className="flex flex-col rounded-md border border-border/70 bg-card text-left">
      <div className="flex justify-between border-b px-3 py-1.5">
        {Array.from({ length: 7 }, (_, index) => (
          <GhostLine key={index} className="h-1.5 w-3" />
        ))}
      </div>
      {Array.from({ length: 4 }, (_, lane) => (
        <div
          key={lane}
          className="flex items-center gap-3 border-b border-dashed border-border/60 px-3 py-2.5 last:border-0"
        >
          <GhostLine className="h-1.5 w-5" />
          <GhostLine className={lane % 2 === 0 ? "w-2/5" : "w-3/5"} />
        </div>
      ))}
    </div>
  );
}

function IntegrationsButton({ label }: { label: string }) {
  return (
    <Button
      size="sm"
      variant="secondary"
      render={<Link to="/admin/integrations" search={{ type: "Samsara" }} />}
    >
      {label}
    </Button>
  );
}

/**
 * Why there are no hours of service to draw, from the stored state the overview read: no
 * provider connected, the driver not matched to one, or nothing reported yet. Null when the
 * plan leaves telematics out.
 */
export function HosLinkState({ link }: { link: Exclude<WorkerHosLink, "Reporting"> | null }) {
  const t = useT();
  if (link === null) {
    return (
      <EmptySheet
        sketch={<LogSketch />}
        title={t("Hours of service are not on your plan")}
        description={t(
          "Live clocks, daily logs and violations come from a connected telematics provider, which your plan does not include.",
        )}
      />
    );
  }
  if (link === "NotConnected") {
    return (
      <EmptySheet
        sketch={<LogSketch />}
        title={t("Samsara is not connected")}
        description={t(
          "Connect your Samsara account to see this driver's clocks, duty status, daily logs and violations here.",
        )}
        action={<IntegrationsButton label={t("Open integrations")} />}
      />
    );
  }
  if (link === "NotLinked") {
    return (
      <EmptySheet
        sketch={<LogSketch />}
        title={t("Not linked to a Samsara driver")}
        description={t(
          "Hours of service appear once this worker is matched to a Samsara driver. Run the worker sync from the Samsara integration to link them.",
        )}
        action={<IntegrationsButton label={t("Open the Samsara integration")} />}
      />
    );
  }
  return (
    <EmptySheet
      sketch={<LogSketch />}
      title={t("No hours of service reported yet")}
      description={t(
        "This driver is linked to Samsara, but Samsara has not reported a duty status for them. Their clocks appear here after their first log.",
      )}
    />
  );
}
