import { AgentTile } from "@/components/agent-identity/agent-tile";
import { MessageThread } from "@/components/assistant/message-thread";
import { useApiMutation } from "@/hooks/use-api-mutation";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { downloadAssistantTranscript } from "@/services/assistant";
import { useDeskStore } from "@/stores/desk-store";
import type { AssistantThread } from "@/types/assistant";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@trenova/shared/components/ui/resizable";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { DownloadIcon, PanelRightOpenIcon, PinIcon, Trash2Icon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { ArtifactsPane } from "./artifacts/artifacts-pane";

const MAX_TITLE_CHARS = 200;

export type DeskConversationProps = {
  thread: AssistantThread;
  agent: AgentDefinitionRow | null;
  agentsUnavailable: boolean;
  onStartNew?: () => void;
  onDelete: (thread: AssistantThread) => void;
  onTogglePin: (thread: AssistantThread) => void;
};

/**
 * One conversation at the Desk: a slim title bar, the thread with the
 * agent's accent down its spine, and the artifacts it produced in a pane
 * beside it. The title is edited in place; the pin and the transcript are
 * a click away because they are the two things a person does to a
 * conversation they want to keep.
 */
export function DeskConversation({
  thread,
  agent,
  agentsUnavailable,
  onStartNew,
  onDelete,
  onTogglePin,
}: DeskConversationProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const pane = useDeskStore((state) => state.pane);
  const setPane = useDeskStore((state) => state.setPane);
  const setActiveArtifact = useDeskStore((state) => state.setActiveArtifact);
  const [liveArtifactIds, setLiveArtifactIds] = useState<string[]>([]);

  const artifactsQuery = useQuery(queries.assistant.artifacts(thread.id));
  const artifacts = useMemo(() => artifactsQuery.data?.results ?? [], [artifactsQuery.data]);

  // A turn that produces something opens the pane on it, even if the pane
  // was folded away: the person asked for the thing it holds.
  const onLiveArtifact = useCallback(
    (id: string) => {
      setLiveArtifactIds((ids) => (ids.includes(id) ? ids : [...ids, id]));
      setPane("open");
    },
    [setPane],
  );
  const openArtifact = useCallback(
    (id: string) => {
      setActiveArtifact(thread.id, id);
      setPane("open");
    },
    [setActiveArtifact, setPane, thread.id],
  );

  const renameMutation = useApiMutation({
    mutationFn: (title: string) => apiService.assistantService.updateThread(thread.id, { title }),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: queries.assistant.threads().queryKey }),
    resourceName: "Conversation",
  });

  return (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col">
      <header className="border-border flex h-11 shrink-0 items-center gap-2 border-b pr-1.5 pl-3">
        <AgentTile agent={agent} size="md" />
        <TitleField
          key={thread.id}
          title={thread.title}
          placeholder={agent ? t("Conversation with {0}", agent.name) : t("Untitled conversation")}
          onCommit={(title) => renameMutation.mutate(title)}
        />
        <div className="ml-auto flex shrink-0 items-center gap-0.5">
          <IconButton
            label={thread.pinned ? t("Unpin conversation") : t("Pin conversation")}
            pressed={thread.pinned}
            onClick={() => onTogglePin(thread)}
          >
            <PinIcon className="size-4" />
          </IconButton>
          <IconButton
            label={t("Download transcript")}
            onClick={() => downloadAssistantTranscript(thread.id)}
          >
            <DownloadIcon className="size-4" />
          </IconButton>
          <IconButton label={t("Delete conversation")} destructive onClick={() => onDelete(thread)}>
            <Trash2Icon className="size-4" />
          </IconButton>
          {pane === "closed" && (
            <IconButton label={t("Show artifacts")} onClick={() => setPane("open")}>
              <PanelRightOpenIcon className="size-4" />
            </IconButton>
          )}
        </div>
      </header>

      <ResizablePanelGroup orientation="horizontal" className="min-h-0 flex-1">
        <ResizablePanel minSize="40%">
          <div className="flex h-full min-h-0 flex-col">
            <MessageThread
              key={thread.id}
              thread={thread}
              agent={agent}
              agentsUnavailable={agentsUnavailable}
              expanded
              onStartNew={onStartNew}
              artifacts={artifacts}
              onOpenArtifact={openArtifact}
              onLiveArtifact={onLiveArtifact}
              spine
            />
          </div>
        </ResizablePanel>
        {pane === "open" && (
          <>
            <ResizableHandle />
            <ResizablePanel defaultSize="38%" minSize="320px" maxSize="60%">
              <ArtifactsPane
                threadId={thread.id}
                liveArtifactIds={liveArtifactIds}
                onClose={() => setPane("closed")}
                className="h-full"
              />
            </ResizablePanel>
          </>
        )}
      </ResizablePanelGroup>
    </div>
  );
}

/**
 * The title, edited where it is read. Enter or leaving the field saves;
 * Escape puts the saved title back. An empty title falls back to the
 * agent's name, which is what the rail shows for it.
 */
function TitleField({
  title,
  placeholder,
  onCommit,
}: {
  title: string;
  placeholder: string;
  onCommit: (title: string) => void;
}) {
  const t = useT();
  const [value, setValue] = useState(title);
  // A title saved elsewhere replaces what is typed here, derived during
  // render rather than a render later.
  const [seen, setSeen] = useState(title);
  if (seen !== title) {
    setSeen(title);
    setValue(title);
  }

  const commit = () => {
    const next = value.trim().slice(0, MAX_TITLE_CHARS);
    if (next !== title) {
      onCommit(next);
    }
  };

  return (
    <Input
      inputContainerClassName="min-w-0 flex-1"
      value={value}
      onChange={(event) => setValue(event.target.value)}
      onBlur={commit}
      onKeyDown={(event) => {
        if (event.key === "Enter") {
          event.preventDefault();
          event.currentTarget.blur();
        } else if (event.key === "Escape") {
          setValue(title);
          event.currentTarget.blur();
        }
      }}
      placeholder={placeholder}
      aria-label={t("Conversation title")}
      className="h-8 border-transparent bg-transparent text-sm font-medium shadow-none hover:bg-surface-hover focus-visible:bg-field"
    />
  );
}

function IconButton({
  label,
  pressed,
  destructive = false,
  onClick,
  children,
}: {
  label: string;
  pressed?: boolean;
  destructive?: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label={label}
            aria-pressed={pressed}
            className={cn(
              pressed ? "text-foreground" : "text-muted-foreground hover:text-foreground",
              destructive && "hover:text-destructive",
            )}
            onClick={onClick}
          />
        }
      >
        {children}
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}
