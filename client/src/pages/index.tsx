import { Card, CardContent } from "@/components/ui/card";
import { useTranslation } from "react-i18next";

export default function Index() {
  const { t } = useTranslation("homepage");

  return (
    <Card>
      <CardContent>
        <p className="mb-2 border-b border-dashed font-semibold">
          {t("homePage.cardTitle")}
        </p>
        <p className="mt-1 text-sm text-gray-400">{t("homePage.cardText")}</p>
      </CardContent>
    </Card>
  );
}
