import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { CopyableSecret } from "@/components/copyable-secret";
import { formatUnixDateTimeOrDash } from "@trenova/shared/lib/date";
import { toTitleCase } from "@trenova/shared/lib/utils";
import { apiService } from "@/services/api";
import { useMutation, useQuery } from "@tanstack/react-query";
import { KeyRoundIcon, PlusIcon } from "lucide-react";
import { memo, useEffect, useState } from "react";
import { toast } from "sonner";
import { EmptyState, PanelHeader, RowSkeleton } from "../security-access/shared";

type SCIMTokenPanelProps = {
  organizationId: string;
  directoryId: string;
  onProvisioningChange: () => Promise<void>;
};

export const SCIMTokenPanel = memo(function SCIMTokenPanel({
  organizationId,
  directoryId,
  onProvisioningChange,
}: SCIMTokenPanelProps) {
  const t = useT();

  const [tokenName, setTokenName] = useState("");
  const [createdToken, setCreatedToken] = useState("");

  useEffect(() => {
    setTokenName("");
    setCreatedToken("");
  }, [directoryId, organizationId]);

  const tokensQuery = useQuery({
    queryKey: ["scim-tokens", organizationId, directoryId],
    queryFn: async () => apiService.organizationService.listSCIMTokens(organizationId, directoryId),
    enabled: Boolean(directoryId),
  });
  const { mutate: createToken, isPending: isCreatingToken } = useMutation({
    mutationFn: async () =>
      apiService.organizationService.createSCIMToken(organizationId, directoryId, tokenName),
    onSuccess: async (response) => {
      setCreatedToken(response.token);
      setTokenName("");
      toast.success(t("SCIM token created"));
      await onProvisioningChange();
    },
  });
  const { mutate: revokeToken } = useMutation({
    mutationFn: async (tokenId: string) =>
      apiService.organizationService.revokeSCIMToken(organizationId, tokenId),
    onSuccess: async () => {
      toast.success(t("SCIM token revoked"));
      await onProvisioningChange();
    },
  });
  const tokens = tokensQuery.data ?? [];
  const createDisabled = !directoryId || isCreatingToken;

  return (
    <div className="bg-background rounded-lg border">
      <PanelHeader
        icon={<KeyRoundIcon />}
        title={t("SCIM tokens")}
        description={t("Issue bearer tokens for directory synchronization.")}
      />
      <div className="space-y-3">
        <div className="flex w-full flex-col gap-2 sm:flex-row">
          <div className="flex w-full flex-row justify-between gap-1 px-2 pt-2">
            <Input
              value={tokenName}
              placeholder={t("Token name")}
              onChange={(event) => setTokenName(event.target.value)}
            />
            <Button
              size="sm"
              onClick={() => createToken()}
              disabled={createDisabled || tokenName.trim() === ""}
            >
              <PlusIcon />
              {t("Create token")}
            </Button>
          </div>
        </div>
        {createdToken && (
          <CopyableSecret
            key={createdToken}
            value={createdToken}
            title={t("Copy this token now")}
            description={t("The plaintext token is only shown once.")}
            className="mx-2"
          />
        )}
        {tokensQuery.isLoading ? (
          <RowSkeleton rows={2} />
        ) : tokens.length > 0 ? (
          <div className="border-border overflow-x-auto border-t">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("Name")}</TableHead>
                  <TableHead>{t("Prefix")}</TableHead>
                  <TableHead>{t("Status")}</TableHead>
                  <TableHead>{t("Last used")}</TableHead>
                  <TableHead className="w-28">{t("Actions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {tokens.map((token) => (
                  <TableRow key={token.id}>
                    <TableCell className="font-medium">{token.name}</TableCell>
                    <TableCell>
                      <code className="bg-muted rounded-md px-1.5 py-0.5 text-xs">
                        {token.prefix}
                      </code>
                    </TableCell>
                    <TableCell>
                      <Badge variant={token.status === "active" ? "success" : "danger"}>
                        {toTitleCase(token.status)}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {token.lastUsedAt ? formatUnixDateTimeOrDash(token.lastUsedAt) : t("Never")}
                    </TableCell>
                    <TableCell>
                      <Button
                        size="sm"
                        variant="destructive"
                        disabled={token.status !== "active"}
                        onClick={() => revokeToken(token.id)}
                      >
                        {t("Revoke")}
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        ) : (
          <EmptyState
            icon={<KeyRoundIcon />}
            label={t("No SCIM tokens")}
            description={t("Create a token and copy it into your directory sync application.")}
            compact
          />
        )}
      </div>
    </div>
  );
});
