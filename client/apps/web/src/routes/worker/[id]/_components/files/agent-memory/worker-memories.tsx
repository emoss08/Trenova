import { ConfirmDialog } from "@/components/confirm-dialog";
import { isOffered, savedLine, scopeLabel } from "@/components/desk-chat/memory/memory-format";
import {
  memoryKindWord,
  recordMemoriesQueryKey,
  useRecordMemoriesAvailable,
} from "@/components/record-memories/record-memories";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  createDeskMemory,
  fetchDeskMemories,
  reviseDeskMemory,
  setDeskMemoryStatus,
  type DeskMemory,
  type DeskMemorySubject,
} from "@/lib/graphql/desk-memories";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { BotIcon, Edit02Icon, Trash01Icon, User01Icon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { useT } from "@trenova/shared/i18n/use-t";
import { resolveUserTimezone } from "@trenova/shared/lib/date";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useState } from "react";
import { FilesSkeleton } from "../files-skeleton";

/** The most notes about one worker the view reads, as the shared memories panel does. */
const PAGE_SIZE = 50;

type WrittenKind = "Fact" | "Instruction";
type WrittenFor = "Organization" | "User";

/**
 * What agents remember about the worker, on the same reads and writes as the shared record
 * memories panel: each note with where it came from and when, editing the ones the reader may
 * change, removing one behind a confirmation, and writing a new one.
 */
export default function WorkerMemories({
  workerId,
  workerName,
}: {
  workerId: string;
  workerName: string;
}) {
  const t = useT();
  const available = useRecordMemoriesAvailable();
  const queryClient = useQueryClient();
  const subject: DeskMemorySubject = { type: "Worker", id: workerId };
  const [adding, setAdding] = useState(false);
  const [forgetting, setForgetting] = useState<DeskMemory | null>(null);

  const memoriesQuery = useQuery({
    queryKey: recordMemoriesQueryKey(subject),
    queryFn: ({ signal }) =>
      fetchDeskMemories({ scope: null, roleId: null, query: "", subject }, null, PAGE_SIZE, signal),
    enabled: available,
  });
  const refresh = () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: recordMemoriesQueryKey(subject) }),
      queryClient.invalidateQueries({ queryKey: ["worker"] }),
    ]);

  const forget = useApiMutation({
    scope: { id: `worker-memory:${workerId}` },
    resourceName: "Memory",
    mutationFn: (memory: DeskMemory) => setDeskMemoryStatus(memory.id, "Retired"),
    onSuccess: () => void refresh(),
  });

  if (!available) return null;
  if (memoriesQuery.isPending) return <FilesSkeleton view="agent-memory" />;

  const memories = memoriesQuery.data?.items ?? [];

  return (
    <div className="flex min-w-0 flex-col gap-4 text-sm">
      <SectionPanel
        title={t("What agents remember about {0}", workerName)}
        hint={
          memories.length > 0
            ? t("{0, plural, one {# note} other {# notes}}", memories.length)
            : undefined
        }
        action={
          adding ? null : (
            <Button size="xs" variant="quiet" onClick={() => setAdding(true)}>
              {t("Add a note")}
            </Button>
          )
        }
      >
        {adding ? (
          <AddNote
            subject={subject}
            workerName={workerName}
            onDone={() => setAdding(false)}
            onSaved={refresh}
          />
        ) : null}
        {memoriesQuery.isError ? (
          <ErrorState
            error={memoriesQuery.error}
            layout="compact"
            title={t("The notes could not be loaded")}
            onRetry={() => void memoriesQuery.refetch()}
          />
        ) : memories.length === 0 ? (
          <SectionPanelQuiet>
            {t(
              "Agents remember nothing about {0} yet. Notes appear here when an agent or a person saves one.",
              workerName,
            )}
          </SectionPanelQuiet>
        ) : (
          <ScrollArea viewportClassName="max-h-[32rem]" maskVariant="card">
            <ul className="flex flex-col divide-y divide-border-subtle px-1.5 py-1">
              {memories.map((memory) => (
                <MemoryRow
                  key={memory.id}
                  memory={memory}
                  removing={forget.isPending && forget.variables?.id === memory.id}
                  onSaved={refresh}
                  onForget={() => setForgetting(memory)}
                />
              ))}
            </ul>
          </ScrollArea>
        )}
        <p className="border-t border-border-subtle px-3 py-2.5 text-xs text-muted-foreground">
          {t(
            "Agents read these before they plan a load or message {0}. Removing a note stops them using it.",
            workerName,
          )}
        </p>
      </SectionPanel>
      <ConfirmDialog
        open={forgetting !== null}
        onOpenChange={(next) => !next && setForgetting(null)}
        title={t("Remove this note?")}
        description={t("Agents stop using it from their next run. This cannot be undone.")}
        confirmLabel={t("Remove note")}
        pending={forget.isPending}
        onConfirm={() => (forgetting ? forget.mutateAsync(forgetting) : undefined)}
      />
    </div>
  );
}

function MemoryRow({
  memory,
  removing,
  onSaved,
  onForget,
}: {
  memory: DeskMemory;
  removing: boolean;
  onSaved: () => Promise<unknown>;
  onForget: () => void;
}) {
  const t = useT();
  const timezone = resolveUserTimezone(useAuthStore((state) => state.user?.timezone));
  const [draft, setDraft] = useState<string | null>(null);
  const Icon = memory.source === "User" ? User01Icon : BotIcon;

  const revise = useApiMutation({
    scope: { id: `worker-memory-revise:${memory.id}` },
    resourceName: "Memory",
    mutationFn: (content: string) =>
      reviseDeskMemory(memory.id, { content, version: memory.version }),
    onSuccess: async () => {
      setDraft(null);
      await onSaved();
    },
  });

  return (
    <li
      data-testid={`memory-${memory.id}`}
      className="group/memory grid grid-cols-[0.875rem_minmax(0,1fr)_auto] items-start gap-2.5 px-2 py-2.5"
    >
      <Icon className="mt-0.5 size-3.5 text-muted-foreground" />
      <div className="min-w-0">
        {draft === null ? (
          <p className="text-sm whitespace-pre-wrap">{memory.content}</p>
        ) : (
          <div className="flex flex-col gap-2">
            <Textarea
              value={draft}
              onChange={(event) => setDraft(event.target.value)}
              rows={3}
              className="text-sm"
              aria-label={t("What agents should remember")}
            />
            <div className="flex justify-end gap-2">
              <Button size="xs" variant="outline" onClick={() => setDraft(null)}>
                {t("Cancel")}
              </Button>
              <Button
                size="xs"
                isLoading={revise.isPending}
                disabled={draft.trim() === "" || draft.trim() === memory.content}
                onClick={() => void revise.mutateAsync(draft.trim()).catch(() => undefined)}
              >
                {t("Save")}
              </Button>
            </div>
          </div>
        )}
        <p className="mt-0.5 flex flex-wrap items-center gap-x-1.5 gap-y-1 text-xs text-muted-foreground tabular-nums">
          <span>
            {t(
              "{0} · {1} · {2}",
              savedLine(memory, timezone, t),
              memoryKindWord(memory.kind, t),
              scopeLabel(memory, t),
            )}
          </span>
          {isOffered(memory) ? (
            <Badge variant="warning" dot>
              {t("Waiting for an administrator")}
            </Badge>
          ) : null}
        </p>
      </div>
      {memory.editable && draft === null ? (
        <span className="flex items-center">
          <Button
            size="icon-xs"
            variant="ghost"
            aria-label={t("Edit this note")}
            onClick={() => setDraft(memory.content)}
          >
            <Edit02Icon className="size-3.5" />
          </Button>
          <Button
            size="icon-xs"
            variant="ghost"
            aria-label={t("Remove this note")}
            isLoading={removing}
            onClick={onForget}
          >
            <Trash01Icon className="size-3.5" />
          </Button>
        </span>
      ) : (
        <span />
      )}
    </li>
  );
}

function AddNote({
  subject,
  workerName,
  onDone,
  onSaved,
}: {
  subject: DeskMemorySubject;
  workerName: string;
  onDone: () => void;
  onSaved: () => Promise<unknown>;
}) {
  const t = useT();
  const [content, setContent] = useState("");
  const [kind, setKind] = useState<WrittenKind>("Fact");
  const [keptFor, setKeptFor] = useState<WrittenFor>("Organization");

  const add = useApiMutation({
    scope: { id: `worker-memory-add:${subject.id}` },
    resourceName: "Memory",
    mutationFn: () =>
      createDeskMemory(content.trim(), { scope: keptFor, roleId: null }, { kind, subject }),
    onSuccess: async () => {
      setContent("");
      await onSaved();
      onDone();
    },
  });

  return (
    <div className="flex flex-col gap-2 border-b border-border-subtle p-3 motion-safe:animate-expand-in">
      <Textarea
        value={content}
        onChange={(event) => setContent(event.target.value)}
        rows={2}
        autoFocus
        className="text-sm"
        aria-label={t("Something agents should remember about {0}", workerName)}
        placeholder={t("e.g. Will not take loads into New York City.")}
      />
      <div className="flex flex-wrap items-center gap-2">
        <SegmentedControl<WrittenKind>
          aria-label={t("What it is")}
          value={kind}
          onValueChange={setKind}
          items={[
            { value: "Fact", label: t("Fact") },
            { value: "Instruction", label: t("Rule") },
          ]}
        />
        <SegmentedControl<WrittenFor>
          aria-label={t("Who it is for")}
          value={keptFor}
          onValueChange={setKeptFor}
          items={[
            { value: "Organization", label: t("Everyone") },
            { value: "User", label: t("Just me") },
          ]}
        />
        <span className="flex-1" />
        <Button size="sm" variant="outline" onClick={onDone}>
          {t("Cancel")}
        </Button>
        <Button
          size="sm"
          isLoading={add.isPending}
          disabled={content.trim() === ""}
          onClick={() => void add.mutateAsync().catch(() => undefined)}
        >
          {t("Remember")}
        </Button>
      </div>
    </div>
  );
}
