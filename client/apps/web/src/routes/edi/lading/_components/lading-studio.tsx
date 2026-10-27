import { useT } from "@trenova/shared/i18n/use-t";
import { lading } from "@/lib/queries/lading";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import type { LadingContext } from "@trenova/shared/edi/lading";
import type { LadingProject } from "@trenova/shared/types/lading";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useQueryState } from "nuqs";
import { LadingCreateProject } from "./lading-create-project";
import { LadingStudioSkeleton } from "./lading-studio-skeleton";
import { LadingWorkbench } from "./lading-workbench";
import { pickVersion } from "./studio-model";

/**
 * The studio for one partner: the start form while the partner has no project, else the
 * version the address names (or the draft, else the active one) open in the workbench.
 */
export default function LadingStudio({
  partnerId,
  partnerName,
}: {
  partnerId: string;
  partnerName: string;
}) {
  const { data: project } = useSuspenseQuery(lading.partnerProject(partnerId));
  const { data: context } = useSuspenseQuery(lading.context());

  if (!project) {
    return <LadingCreateProject partnerId={partnerId} partnerName={partnerName} />;
  }
  return <ProjectStudio project={project} context={context} partnerName={partnerName} />;
}

function ProjectStudio({
  project,
  context,
  partnerName,
}: {
  project: LadingProject;
  context: LadingContext;
  partnerName: string;
}) {
  const t = useT();
  const { data: versions } = useSuspenseQuery(lading.versions(project.id));
  const [requested, setRequested] = useQueryState("version");
  const picked = pickVersion(versions, requested, project.activeVersionId);

  if (!picked) {
    return (
      <EmptySheet
        title={t("This project has no versions")}
        description={t("Every version was removed. Start the project again from the partner list.")}
        sketch={
          <div className="flex flex-col gap-2">
            <GhostLine className="w-2/3" />
            <GhostLine className="w-1/2" />
            <GhostLine className="w-3/4" />
          </div>
        }
      />
    );
  }

  return (
    <LazyComponent fallback={<LadingStudioSkeleton />}>
      <VersionStudio
        key={picked.id}
        project={project}
        versionId={picked.id}
        context={context}
        partnerName={partnerName}
        onPickVersion={(id) => void setRequested(id)}
      />
    </LazyComponent>
  );
}

function VersionStudio({
  project,
  versionId,
  context,
  partnerName,
  onPickVersion,
}: {
  project: LadingProject;
  versionId: string;
  context: LadingContext;
  partnerName: string;
  onPickVersion: (id: string) => void;
}) {
  const { data: versions } = useSuspenseQuery(lading.versions(project.id));
  const { data: detail } = useSuspenseQuery(lading.version(versionId));
  return (
    <LadingWorkbench
      project={project}
      versions={versions}
      detail={detail}
      context={context}
      partnerName={partnerName}
      onPickVersion={onPickVersion}
    />
  );
}
