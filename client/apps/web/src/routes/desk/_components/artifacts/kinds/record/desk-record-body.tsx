import { DisplayValue } from "@/components/assistant/display-value";
import {
  formatDisplayValue,
  isDetailType,
  statusPhase,
} from "@/components/assistant/readable-values";
import type { AssistantArtifact } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { humanizeToolName } from "@/components/assistant/proposal-state";
import { useMemo } from "react";
import { Link } from "react-router";
import { entityCardFrom } from "../../artifact-payloads";
import { ArtIcon } from "../../desk-art-kinds";
import { recordCardFor, RecordArtifact } from "../../cards/registry";
import { DeskRecordView } from "./desk-record-view";
import { entityName } from "../artifact-body-parts";

const PHASE_PILL: Record<string, string> = {
  failed: "dk-warn",
  attention: "dk-warn",
  awaiting: "dk-blue",
  active: "dk-blue",
  queued: "dk-blue",
  complete: "dk-green",
  closed: "dk-ink",
};

/** A record as its name, its status and its fields, with a way into the record itself. */
export function DeskRecordBody({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const card = useMemo(() => entityCardFrom(artifact), [artifact]);
  const [lead, ...rest] = card.fields;
  const status = rest.find((field) => field.type === "status");
  const fields = rest.filter((field) => field !== status && !isDetailType(field.type));
  const details = rest.filter((field) => isDetailType(field.type));
  const title = lead ? formatDisplayValue(lead.type, lead.value, t) : artifact.title;
  const statusText = status ? formatDisplayValue(status.type, status.value, t) : "";
  const phase = status && typeof status.value === "string" ? statusPhase(status.value) : null;

  const viewed = card.view ? (
    <DeskRecordView title={title} view={card.view} entity={card.entity} path={card.path} />
  ) : null;
  // A record with a card of its own is read live there, and acted on there
  // when its card has actions: a billing item is reviewed from it.
  const resource = card.recordId !== "" ? recordCardFor(card.entity) : null;
  if (resource) {
    return (
      <RecordArtifact
        entry={resource}
        artifact={artifact}
        recordId={card.recordId}
        fallback={viewed ?? <resource.Skeleton />}
      />
    );
  }
  if (viewed) {
    return viewed;
  }

  return (
    <div className="dk-ax-pad dk-ax-rec">
      <div className="dk-ax-rec-h">
        <div>
          <div className="dk-ax-big">{title}</div>
          {card.entity !== "" && <div className="dk-ax-sub">{entityName(card.entity, t)}</div>}
        </div>
        {statusText !== "" && (
          <span className={cn("dk-ax-pill dk-lg", phase ? PHASE_PILL[phase] : "")}>
            <i />
            {statusText}
          </span>
        )}
      </div>
      <dl className="dk-ax-fields">
        {fields.map((field) => (
          <div key={field.key}>
            <dt>{field.label}</dt>
            <dd>
              <DisplayValue type={field.type} value={field.value} label={field.label} inline />
            </dd>
          </div>
        ))}
      </dl>
      {details.map((field) => (
        <div key={field.key} className="dk-ax-note">
          <b>{field.label}</b>{" "}
          <DisplayValue type={field.type} value={field.value} label={field.label} />
        </div>
      ))}
      {card.path !== "" && (
        <Link className="dk-ax-btn" to={card.path}>
          <ArtIcon name="ext" size={13} />
          {t("Open {0}", card.entity ? humanizeToolName(card.entity).toLowerCase() : t("record"))}
        </Link>
      )}
    </div>
  );
}
