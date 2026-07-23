import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { formatUnixDateTimeOrDash } from "@/lib/date";
import { toTitleCase } from "@/lib/utils";
import { apiService } from "@/services/api";
import { useMutation, useQuery } from "@tanstack/react-query";
import { CheckIcon, ClipboardIcon, KeyRoundIcon, PlusIcon } from "lucide-react";
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
      toast.success("SCIM token created");
      await onProvisioningChange();
    },
  });
  const { mutate: revokeToken } = useMutation({
    mutationFn: async (tokenId: string) =>
      apiService.organizationService.revokeSCIMToken(organizationId, tokenId),
    onSuccess: async () => {
      toast.success("SCIM token revoked");
      await onProvisioningChange();
    },
  });
  const tokens = tokensQuery.data ?? [];
  const createDisabled = !directoryId || isCreatingToken;

  return (
    <div className="rounded-lg border bg-background">
      <PanelHeader
        icon={<KeyRoundIcon />}
        title="SCIM tokens"
        description="Issue bearer tokens for directory synchronization."
      />
      <div className="space-y-3">
        <div className="flex w-full flex-col gap-2 sm:flex-row">
          <div className="flex w-full flex-row justify-between gap-1 px-2 pt-2">
            <Input
              value={tokenName}
              placeholder="Token name"
              onChange={(event) => setTokenName(event.target.value)}
            />
            <Button
              size="sm"
              onClick={() => createToken()}
              disabled={createDisabled || tokenName.trim() === ""}
            >
              <PlusIcon />
              Create token
            </Button>
          </div>
        </div>
        {createdToken && <CopyableSecretBlock key={createdToken} value={createdToken} />}
        {tokensQuery.isLoading ? (
          <RowSkeleton rows={2} />
        ) : tokens.length > 0 ? (
          <div className="overflow-x-auto border-t border-border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Prefix</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Last used</TableHead>
                  <TableHead className="w-28">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {tokens.map((token) => (
                  <TableRow key={token.id}>
                    <TableCell className="font-medium">{token.name}</TableCell>
                    <TableCell>
                      <code className="rounded bg-muted px-1.5 py-0.5 text-xs">{token.prefix}</code>
                    </TableCell>
                    <TableCell>
                      <Badge variant={token.status === "active" ? "active" : "inactive"}>
                        {toTitleCase(token.status)}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {token.lastUsedAt ? formatUnixDateTimeOrDash(token.lastUsedAt) : "Never"}
                    </TableCell>
                    <TableCell>
                      <Button
                        size="sm"
                        variant="destructive"
                        disabled={token.status !== "active"}
                        onClick={() => revokeToken(token.id)}
                      >
                        Revoke
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
            label="No SCIM tokens"
            description="Create a token and copy it into your directory sync application."
            compact
          />
        )}
      </div>
    </div>
  );
});

function CopyableSecretBlock({ value }: { value: string }) {
  const { copy, isCopied } = useCopyToClipboard();

  return (
    <div className="mx-2 rounded-lg border border-amber-600/30 bg-amber-600/10 p-3">
      <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
        <div>
          <div className="text-sm font-medium text-amber-800 dark:text-amber-300">
            Copy this token now
          </div>
          <div className="text-xs text-amber-700/80 dark:text-amber-300/80">
            The plaintext token is only shown once.
          </div>
        </div>
        <Button size="sm" variant="outline" onClick={() => void copy(value, { withToast: true })}>
          {isCopied ? <CheckIcon /> : <ClipboardIcon />}
          {isCopied ? "Copied" : "Copy"}
        </Button>
      </div>
      <code className="block rounded-md border bg-background/80 p-2 font-mono text-xs break-all">
        {value}
      </code>
    </div>
  );
}
