import { useT } from "@trenova/shared/i18n/use-t";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { lading } from "@/lib/queries/lading";
import { apiService } from "@/services/api";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { Label } from "@trenova/shared/components/ui/label";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

type Role = "carrier" | "broker";

/** The sets a tender relationship trades, in the order the starter writes them. */
const SETS = ["204", "990", "214", "210"] as const;

/**
 * A partner with no project yet: pick whether Trenova is the carrier or the broker with this
 * partner and the sets it trades, and the starter for that role becomes the project's first
 * draft.
 */
export function LadingCreateProject({
  partnerId,
  partnerName,
}: {
  partnerId: string;
  partnerName: string;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canCreate } = usePermission(Resource.EDI, Operation.Create);
  const [role, setRole] = useState<Role>("carrier");
  const [sets, setSets] = useState<string[]>([...SETS]);

  const create = useApiMutation({
    mutationFn: () =>
      apiService.ladingService.createProject({
        partnerId,
        role,
        sets: SETS.filter((set) => sets.includes(set)),
        usage: "test",
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: lading.partnerProject(partnerId).queryKey,
      });
    },
  });

  const describe = (set: string) =>
    set === "204"
      ? t("204 load tender")
      : set === "990"
        ? t("990 tender response")
        : set === "214"
          ? t("214 shipment status")
          : t("210 freight invoice");

  return (
    <div className="mx-auto flex w-full max-w-xl flex-col gap-4 py-8">
      <div className="flex flex-col gap-1">
        <h2 className="text-lg font-semibold">{t("Start {0}'s Lading project", partnerName)}</h2>
        <p className="text-muted-foreground text-sm">
          {t(
            "The starter writes the partner file, a map for each set, the tender flow, samples and tests. It opens as a draft that sends nothing until it is certified and activated.",
          )}
        </p>
      </div>
      <div className="flex flex-col gap-2">
        <Label>{t("Trenova is")}</Label>
        <SegmentedControl<Role>
          value={role}
          aria-label={t("Trenova's role with this partner")}
          onValueChange={setRole}
          items={[
            { value: "carrier", label: t("The carrier: the partner tenders to us") },
            { value: "broker", label: t("The broker: we tender to the partner") },
          ]}
        />
      </div>
      <div className="flex flex-col gap-2">
        <Label>{t("Sets traded")}</Label>
        {SETS.map((set) => (
          <label key={set} className="flex items-center gap-2 text-sm">
            <Checkbox
              checked={sets.includes(set)}
              onCheckedChange={(checked) =>
                setSets((current) =>
                  checked === true ? [...current, set] : current.filter((value) => value !== set),
                )
              }
            />
            {describe(set)}
          </label>
        ))}
      </div>
      <div className="flex justify-end">
        <Button
          disabled={!canCreate || sets.length === 0}
          isLoading={create.isPending}
          onClick={() => create.mutate()}
        >
          {t("Start the project")}
        </Button>
      </div>
    </div>
  );
}
