import { Card, CardContent } from "@trenova/shared/components/ui/card";

export function StatusCard({
  icon,
  title,
  body,
  action,
}: {
  icon: React.ReactNode;
  title: string;
  body: string;
  action?: React.ReactNode;
}) {
  return (
    <Card className="w-full max-w-md">
      <CardContent className="flex flex-col items-center gap-2 py-8 text-center">
        {icon}
        <p className="text-sm font-semibold">{title}</p>
        <p className="text-xs text-muted-foreground">{body}</p>
        {action}
      </CardContent>
    </Card>
  );
}
