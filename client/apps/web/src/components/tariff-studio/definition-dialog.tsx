import { TariffEditor } from "@/components/formula-editor/tariff-editor";
import type { TariffDefinitionTarget } from "@/components/formula-editor/tariff-service";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { useRules } from "@trenova/shared/hooks/use-rules";
import { useT } from "@trenova/shared/i18n/use-t";
import { tariffCatalog } from "@trenova/shared/rules/tariff";
import type { TariffContext } from "@trenova/shared/types/tariff";
import { tariffDocText } from "@/components/formula-editor/tariff-doc";
import { cn } from "@trenova/shared/lib/utils";

/**
 * Go to definition into an imported library: the library's source, read-only, with the
 * declaration lit and in view. An organisation library's source comes from the context the file
 * was checked against (the version it pins); a standard library declaration is shown from the
 * catalog, with its signature and doc.
 */
export function DefinitionDialog({
  target,
  context,
  onClose,
}: {
  target: TariffDefinitionTarget | null;
  context: TariffContext | null;
  onClose: () => void;
}) {
  const t = useT();
  const rules = useRules();
  const library = target
    ? (context?.libraries ?? []).find((candidate) => candidate.path === target.module)
    : undefined;
  const std =
    target && !library
      ? tariffCatalog(rules).modules.find((module) => module.path === target.module)
      : undefined;

  return (
    <Dialog open={target !== null} onOpenChange={(open) => !open && onClose()}>
      <DialogContent size="xl">
        <DialogHeader>
          <DialogTitle className="font-mono">{target?.module}</DialogTitle>
          <DialogDescription>
            {library
              ? t("The library as this file imports it: the version its check was given.")
              : t("Part of Tariff's standard library.")}
          </DialogDescription>
        </DialogHeader>
        {library && target ? (
          <TariffEditor
            kind="library"
            value={library.source}
            onChange={() => undefined}
            context={context}
            highlight={target.range}
            onView={(view) => {
              if (!view) return;
              const from = Math.min(target.range.from, view.state.doc.length);
              view.dispatch({ selection: { anchor: from }, scrollIntoView: true });
            }}
            readOnly
            ariaLabel={t("Source of {0}", target.module)}
            maxHeight="60vh"
          />
        ) : std ? (
          <ul className="flex max-h-[60vh] flex-col gap-2 overflow-auto">
            {[...std.declarations]
              .sort((a, b) => Number(b.name === target?.name) - Number(a.name === target?.name))
              .map((declaration) => (
                <li
                  key={declaration.name}
                  className={cn(
                    "flex flex-col gap-0.5 rounded-md px-2 py-1",
                    declaration.name === target?.name && "bg-surface-selected",
                  )}
                >
                  <code className="font-mono text-xs">{declaration.signature}</code>
                  {declaration.doc && (
                    <span className="text-foreground-muted text-xs">
                      {tariffDocText(declaration.doc)}
                    </span>
                  )}
                </li>
              ))}
          </ul>
        ) : (
          <p className="text-foreground-muted text-sm">
            {t("This library is not available to this tariff.")}
          </p>
        )}
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            {t("Close")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
