import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { ShipmentPageTab } from "@/types/shipment";
import debounce from "lodash-es/debounce";
import { useCallback, useEffect, useState } from "react";

export function ShipmentAsideMenu({
  tabs,
  activeTab,
  setActiveTab,
}: {
  tabs: Record<string, ShipmentPageTab>;
  activeTab: string;
  setActiveTab: (tabName: string) => void;
}) {
  const [errorTabs] = useState<string[]>([]);
  const [isScrolled, setIsScrolled] = useState(false);

  const handleTabClick = useCallback(
    (tabName: string) => {
      setActiveTab(tabName);
    },
    [setActiveTab],
  );

  const debouncedHandleScroll = debounce(() => {
    if (window.scrollY > 80) {
      setIsScrolled(true);
    } else {
      setIsScrolled(false);
    }
  }, 100);

  useEffect(() => {
    const handleScroll = () => debouncedHandleScroll();
    window.addEventListener("scroll", handleScroll);

    return () => {
      window.removeEventListener("scroll", handleScroll);
      debouncedHandleScroll.cancel && debouncedHandleScroll.cancel();
    };
  }, []);

  return (
    <aside
      className={cn(
        "transition-spacing fixed top-14 z-30 -ml-2 hidden h-[calc(100vh-10rem)] w-full shrink-0 duration-500 md:sticky md:block",
        isScrolled && "pt-10",
      )}
    >
      <div className="rounded-lg border bg-card p-2 text-card-foreground">
        <nav className="lg:flex-col lg:space-y-2">
          {Object.entries(tabs).map(([tabKey, tabInfo]) => (
            <div key={tabKey} className="space-y-2">
              <div
                onClick={() => handleTabClick(tabKey)}
                className={cn(
                  buttonVariants({ variant: "ghost", size: "nosize" }),
                  activeTab === tabKey
                    ? "bg-muted [&_svg]:text-foreground"
                    : "hover:bg-muted",
                  errorTabs.includes(tabKey) &&
                    "border text-destructive bg-destructive/20 border-destructive hover:bg-destructive/30",
                  "group flex flex-col items-start mx-2 my-1 p-2 text-wrap cursor-pointer select-none",
                )}
              >
                <div className="flex items-center space-x-2">
                  <span>{tabInfo.icon}</span>
                  <span>{tabInfo.name}</span>
                </div>
                <div className="text-xs text-muted-foreground">
                  {tabInfo.description}
                </div>
              </div>
            </div>
          ))}
        </nav>
      </div>
    </aside>
  );
}
