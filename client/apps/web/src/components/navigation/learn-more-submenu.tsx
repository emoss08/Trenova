import { TRENOVA_LINKS } from "@/lib/trenova-links";
import { useT } from "@trenova/shared/i18n/use-t";
import { BookOpen01Icon, LinkExternal01Icon } from "@trenova/shared/components/icons";
import {
  DropdownMenuLinkItem,
  DropdownMenuPortal,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";

function ExternalLinkItem({ href, title }: { href: string; title: string }) {
  return (
    <DropdownMenuLinkItem
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      title={title}
      endContent={<LinkExternal01Icon className="text-muted-foreground size-3.5" />}
    />
  );
}

/**
 * Where to read more about Trenova from the user menu: the Academy first, then
 * the documents that govern using it. Each opens in a new tab.
 */
export function LearnMoreSubmenu() {
  const t = useT();

  return (
    <DropdownMenuSub>
      <DropdownMenuSubTrigger>
        <BookOpen01Icon className="mr-2 size-4" />
        <span>{t("Learn more")}</span>
      </DropdownMenuSubTrigger>
      <DropdownMenuPortal>
        <DropdownMenuSubContent sideOffset={5} className="min-w-56">
          <ExternalLinkItem href={TRENOVA_LINKS.academy} title={t("Trenova Academy")} />
          <DropdownMenuSeparator />
          <ExternalLinkItem href={TRENOVA_LINKS.terms} title={t("Terms of Service")} />
          <ExternalLinkItem href={TRENOVA_LINKS.privacy} title={t("Privacy policy")} />
          <ExternalLinkItem href={TRENOVA_LINKS.cloudTerms} title={t("Trenova Cloud terms")} />
          <ExternalLinkItem
            href={TRENOVA_LINKS.cloudPrivacy}
            title={t("Trenova Cloud privacy policy")}
          />
          <ExternalLinkItem href={TRENOVA_LINKS.license} title={t("License")} />
        </DropdownMenuSubContent>
      </DropdownMenuPortal>
    </DropdownMenuSub>
  );
}
