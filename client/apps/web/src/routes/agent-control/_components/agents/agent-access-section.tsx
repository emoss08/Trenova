import { describeToolCall } from "@/components/assistant/tool-presentation";
import { RoleAutocompleteField } from "@/components/autocomplete-fields";
import { FieldWrapper } from "@/components/fields/field-components";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import type {
  AgentAccessMode,
  AgentAudienceRole,
  AgentAudienceSuggestion,
} from "@/lib/graphql/agent-access";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { CheckIcon, PlusIcon, ShieldAlertIcon, UsersIcon, UserRoundCheckIcon } from "lucide-react";
import { useMemo } from "react";
import { useController, useFormContext, useWatch } from "react-hook-form";
import type { AgentFormValues } from "./agent-form-schema";
import { resourceLabel } from "./tool-catalog";

const COVERAGE_ORDER: Record<AgentAudienceRole["coverage"], number> = {
  Full: 0,
  Partial: 1,
  None: 2,
};

/**
 * The roles an agent could be given to, the ones that could use all of it
 * first, then those missing something, then those that cannot use the
 * assistant at all; alphabetical within each.
 */
export function rankAudience(roles: readonly AgentAudienceRole[]): AgentAudienceRole[] {
  return [...roles].sort(
    (a, b) =>
      COVERAGE_ORDER[a.coverage] - COVERAGE_ORDER[b.coverage] ||
      a.role.name.localeCompare(b.role.name),
  );
}

/**
 * Who may use the agent, among the people who may use the assistant: everyone,
 * or the roles chosen here. A system agent is always everyone's, because
 * Trenova runs it for people. Saved with its own request when the agent is.
 */
export function AgentAccessSection({
  mode,
  agentId,
  isSystem,
}: {
  mode: "create" | "edit";
  /** The saved agent; empty while it is being created. */
  agentId: string;
  isSystem: boolean;
}) {
  const t = useT();
  const { control, setValue } = useFormContext<AgentFormValues>();
  const { field: modeField } = useController({ control, name: "accessMode" });
  const roleIds = useWatch({ control, name: "accessRoleIds" });
  const restricted = modeField.value === "Roles";
  const saved = mode === "edit" && agentId !== "";

  const audienceQuery = useQuery({
    ...queries.assistant.agentAudience(agentId),
    enabled: saved,
    staleTime: 30_000,
  });

  const addRole = (id: string) => {
    if (roleIds.includes(id)) {
      return;
    }
    setValue("accessRoleIds", [...roleIds, id], { shouldDirty: true, shouldValidate: true });
  };

  return (
    <FormSection
      title={t("Who can use it")}
      description={t(
        "Among the people who can use the assistant, who may talk to this agent and decide what it proposes.",
      )}
    >
      <FieldWrapper label={t("Access")}>
        <SegmentedControl<AgentAccessMode>
          fullWidth
          value={isSystem ? "Everyone" : modeField.value}
          onValueChange={(value) => {
            if (isSystem) return;
            modeField.onChange(value);
          }}
          aria-label={t("Who can use it")}
          items={[
            {
              value: "Everyone",
              label: t("Everyone who can use the assistant"),
              icon: UsersIcon,
            },
            {
              value: "Roles",
              label: t("Specific roles"),
              icon: UserRoundCheckIcon,
              disabled: isSystem,
            },
          ]}
        />
      </FieldWrapper>

      {isSystem ? (
        <p className="text-muted-foreground text-xs">
          {t(
            "A system agent is open to everyone who can use the assistant, because Trenova runs it for them. It cannot be given to particular roles.",
          )}
        </p>
      ) : (
        <>
          {restricted && (
            <FormGroup cols={1}>
              <FormControl cols="full">
                <RoleAutocompleteField<AgentFormValues>
                  control={control}
                  name="accessRoleIds"
                  label={t("Roles")}
                  placeholder={t("Choose roles")}
                  description={t(
                    "People with any of these roles, or a role that inherits one, can use it.",
                  )}
                />
              </FormControl>
            </FormGroup>
          )}

          {restricted && roleIds.length === 0 && (
            <Alert variant="warning" size="sm">
              <ShieldAlertIcon />
              <AlertDescription>
                {t("No role is chosen, so nobody can use this agent until one is.")}
              </AlertDescription>
            </Alert>
          )}

          {!restricted && roleIds.length > 0 && (
            <p className="text-muted-foreground text-xs">
              {t(
                "{0, plural, one {# role stays chosen for when it is limited to specific roles again.} other {# roles stay chosen for when it is limited to specific roles again.}}",
                roleIds.length,
              )}
            </p>
          )}

          {!restricted && saved && audienceQuery.data && (
            <SensitiveToolsWarning suggestion={audienceQuery.data} />
          )}

          {restricted && (
            <SuggestedRoles
              saved={saved}
              isLoading={audienceQuery.isLoading}
              isError={audienceQuery.isError}
              roles={audienceQuery.data?.roles ?? null}
              chosen={roleIds}
              onAdd={addRole}
            />
          )}
        </>
      )}
    </FormSection>
  );
}

/**
 * The warning an agent open to everyone earns when it holds tools that reach
 * restricted or confidential data: anyone who can use the assistant can ask
 * for them, within their own permissions.
 */
function SensitiveToolsWarning({ suggestion }: { suggestion: AgentAudienceSuggestion }) {
  const t = useT();
  if (suggestion.sensitiveTools.length === 0) {
    return null;
  }
  const tools = suggestion.sensitiveTools.map((name) => describeToolCall(name, null).title);

  return (
    <Alert variant="warning" size="sm">
      <ShieldAlertIcon />
      <AlertTitle>{t("Open to everyone, with tools that reach sensitive data")}</AlertTitle>
      <AlertDescription>
        {t(
          "{0} reach restricted or confidential records. Anyone who can use the assistant can ask this agent to use them, within their own permissions. Consider limiting it to specific roles.",
          tools.join(", "),
        )}
      </AlertDescription>
    </Alert>
  );
}

/**
 * What each role could make of the agent, from the tools it holds as saved:
 * all of it, some of it and what is missing, or nothing because the role
 * cannot use the assistant. Suggestions are the system's, so they carry its
 * mark; choosing one adds the role to the list above.
 */
function SuggestedRoles({
  saved,
  isLoading,
  isError,
  roles,
  chosen,
  onAdd,
}: {
  saved: boolean;
  isLoading: boolean;
  isError: boolean;
  roles: readonly AgentAudienceRole[] | null;
  chosen: readonly string[];
  onAdd: (roleId: string) => void;
}) {
  const t = useT();
  const ranked = useMemo(() => (roles ? rankAudience(roles) : []), [roles]);
  const chosenSet = useMemo(() => new Set(chosen), [chosen]);

  return (
    <SectionPanel
      title={t("Suggested roles")}
      icon={<AssistMark />}
      count={ranked.length}
      help={t(
        "Worked out from the tools the agent holds as saved, against what each role may do. A role missing something can still use the agent; the tools it lacks permission for are refused.",
      )}
    >
      {!saved ? (
        <SectionPanelQuiet>
          {t("Suggestions appear once the agent is saved, from the tools it holds.")}
        </SectionPanelQuiet>
      ) : isLoading ? (
        <div className="flex flex-col gap-2 px-3 py-3">
          <Skeleton className="h-4 w-2/3" />
          <Skeleton className="h-4 w-1/2" />
        </div>
      ) : isError ? (
        <SectionPanelQuiet>{t("Suggestions could not be loaded.")}</SectionPanelQuiet>
      ) : ranked.length === 0 ? (
        <SectionPanelQuiet>{t("This organization has no roles yet.")}</SectionPanelQuiet>
      ) : (
        <ul className="max-h-64 overflow-y-auto" aria-label={t("Suggested roles")}>
          {ranked.map((entry) => (
            <SuggestedRoleRow
              key={entry.role.id}
              entry={entry}
              chosen={chosenSet.has(entry.role.id)}
              onAdd={() => onAdd(entry.role.id)}
            />
          ))}
        </ul>
      )}
    </SectionPanel>
  );
}

/** What a role's coverage means, in the words the list shows. */
export function coverageLine(
  entry: Pick<AgentAudienceRole, "coverage" | "missingResources">,
  t: ReturnType<typeof useT>,
): string {
  switch (entry.coverage) {
    case "Full":
      return t("Can use all of its tools");
    case "Partial":
      return t(
        "Missing: {0}",
        [...new Set(entry.missingResources.map((resource) => resourceLabel(resource)))].join(", "),
      );
    case "None":
      return t("Can't use the assistant");
  }
}

function SuggestedRoleRow({
  entry,
  chosen,
  onAdd,
}: {
  entry: AgentAudienceRole;
  chosen: boolean;
  onAdd: () => void;
}) {
  const t = useT();
  const unusable = entry.coverage === "None";

  return (
    <li className="border-border-subtle flex items-center gap-3 border-b px-3 py-2 last:border-b-0">
      <div className="flex min-w-0 flex-1 flex-col">
        <span className="text-foreground truncate text-sm">{entry.role.name}</span>
        <span
          className={cn(
            "truncate text-xs",
            entry.coverage === "Full" && "text-success",
            entry.coverage === "Partial" && "text-warning",
            unusable && "text-foreground-subtle",
          )}
        >
          {coverageLine(entry, t)}
        </span>
      </div>
      {chosen ? (
        <span className="text-muted-foreground inline-flex shrink-0 items-center gap-1 text-xs">
          <CheckIcon className="size-3" />
          {t("Chosen")}
        </span>
      ) : (
        <Button
          type="button"
          size="xs"
          variant="outline"
          disabled={unusable}
          onClick={onAdd}
          aria-label={t("Add {0}", entry.role.name)}
        >
          <PlusIcon className="size-3" />
          {t("Add")}
        </Button>
      )}
    </li>
  );
}
