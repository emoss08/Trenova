/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Button } from "@/components/ui/button";
import { DesktopIcon, MoonIcon, SunIcon } from "@radix-ui/react-icons";
import { Theme, useTheme } from "../theme-provider";

const ThemeIcons: Record<Theme, React.ReactNode> = {
  light: <SunIcon />,
  dark: <MoonIcon />,
  system: <DesktopIcon />,
};

export function SiteSearchFooter() {
  const { setTheme } = useTheme();

  return (
    <div className="bg-sidebar flex h-12 items-center justify-between border-t px-3 py-2 text-sm">
      <div className="text-muted-foreground flex space-x-1 text-xs">
        <span>&#8593;</span>
        <span>&#8595;</span>
        <p className="pr-2">to navigate</p>
        <span>&#x23CE;</span>
        <p className="pr-2">to select</p>
        <span>esc</span>
        <p>to close</p>
      </div>
      <div className="flex space-x-2">
        {Object.entries(ThemeIcons).map(([key, icon]) => (
          <Button
            variant="ghost"
            size="icon"
            className="text-foreground size-4 cursor-pointer"
            key={key}
            onClick={() => setTheme(key as Theme)}
          >
            {icon}
          </Button>
        ))}
      </div>
    </div>
  );
}
