import logoRainbow from "@/assets/logo.webp";
import { LazyImage } from "@/components/image";

export function PublicPageShell({
  children,
  footer,
}: {
  children: React.ReactNode;
  footer: string;
}) {
  return (
    <div className="bg-background fixed inset-0 h-svh w-full overflow-y-auto">
      <div className="flex min-h-full flex-col items-center justify-center gap-6 p-6 md:p-10">
        <LazyImage src={logoRainbow} alt="Trenova Logo" className="size-12 object-contain" />
        {children}
        <p className="text-muted-foreground text-center text-[11px]">{footer}</p>
      </div>
    </div>
  );
}
