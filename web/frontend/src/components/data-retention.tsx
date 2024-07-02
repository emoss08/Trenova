import { useTranslation } from "react-i18next";
import { Separator } from "./ui/separator";

export default function GoogleApi() {
  const { t } = useTranslation(["admin.generalpage", "common"]);

  return (
    <>
      <div className="space-y-3">
        <div>
          <h1 className="text-foreground text-2xl font-semibold">
            {t("title")}
          </h1>
          <p className="text-muted-foreground text-sm">{t("subTitle")}</p>
        </div>
        <Separator />
      </div>
      <ul role="list" className="divide-foreground divide-y">
        <li className="flex py-4">
          <div className="ml-3">
            <p className="text-foreground text-sm font-medium">TEST</p>
            <p className="text-muted-foreground text-sm">TEST</p>
          </div>
        </li>
      </ul>
    </>
  );
}
