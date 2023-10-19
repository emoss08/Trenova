import { useTheme } from "@/components/theme-provider";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { ToastAction } from "@/components/ui/toast";
import { useToast } from "@/components/ui/use-toast";
import { ThemeOptions } from "@/types";
import { useState } from "react";

const getFormattedDate = (): string => {
  const today = new Date();
  return `${today.toLocaleString("en-US", {
    weekday: "long",
  })}, ${today.toLocaleString("en-US", {
    month: "long",
  })} ${today.getDate()}, ${today.getFullYear()} at ${today.toLocaleString(
    "en-US",
    { hour: "numeric", minute: "2-digit", hour12: true },
  )}`;
};

export function ThemeSwitcher() {
  const { theme, setTheme, setIsRainbowAnimationActive } = useTheme();
  const { toast } = useToast();
  const [currentTheme, setCurrentTheme] = useState(theme);
  const [previousTheme, setPreviousTheme] = useState(theme);

  const switchTheme = (selectedTheme: ThemeOptions) => {
    // If the selected theme is the same as the current one, just return
    if (currentTheme === selectedTheme) {
      return;
    }
    // First, set the previous theme to the current theme
    setPreviousTheme(currentTheme);
    // Now, set the current theme to the selected theme
    setCurrentTheme(selectedTheme);
    // Then, make necessary changes like showing toast and so on
    setTheme(selectedTheme);
    toast({
      title: `Theme changed to ${selectedTheme}`,
      description: getFormattedDate(),
      action: (
        <ToastAction altText="Goto schedule to undo" onClick={undoThemeChange}>
          Undo
        </ToastAction>
      ),
    });
  };

  const undoThemeChange = () => {
    // Set the current theme back to the previous theme
    setCurrentTheme(previousTheme);
    // Update the actual theme
    setTheme(previousTheme);
    toast({
      title: `Theme reverted to ${previousTheme}`,
      description: "Your theme change was undone.",
    });
  };

  return (
    <div className="space-y-1">
      <Label>Theme</Label>
      <p className="text-[0.8rem] text-muted-foreground">
        Select the theme you'd like to use.
      </p>
      <div className="grid max-w-xl grid-cols-3 gap-8 pt-2">
        <div className="space-y-2">
          <label className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 border-primary">
            <Button
              type="button"
              onClick={() => switchTheme("light")}
              className="aspect-square h-4 w-4 rounded-full border border-primary text-primary shadow focus:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 sr-only"
            />
            <div className="items-center rounded-md border-2 border-muted bg-popover p-1 hover:bg-accent hover:text-accent-foreground">
              <div className="space-y-2 rounded-sm bg-white p-2">
                <div className="space-y-2 rounded-md bg-black/10 p-2 shadow-sm">
                  <div className="h-2 w-[80px] rounded-lg bg-black/50"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-black/50"></div>
                </div>
                <div className="flex items-center space-x-2 rounded-md bg-black/10 p-2 shadow-sm">
                  <div className="h-4 w-4 rounded-full bg-black/50"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-black/50"></div>
                </div>
                <div className="flex items-center space-x-2 rounded-md bg-black/10 p-2 shadow-sm">
                  <div className="h-4 w-4 rounded-full bg-black/50"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-black/50"></div>
                </div>
              </div>
            </div>
            <span className="block w-full p-2 text-center font-normal">
              Light
            </span>
          </label>
        </div>
        <div className="space-y-2">
          <label className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 border-primary">
            <button
              type="button"
              onClick={() => switchTheme("dark")}
              className="aspect-square h-4 w-4 rounded-full border border-primary text-primary shadow focus:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 sr-only"
            ></button>
            <div className="items-center rounded-md border-2 border-muted bg-popover p-1 hover:bg-accent hover:text-accent-foreground">
              <div className="space-y-2 rounded-sm bg-zinc-950 p-2">
                <div className="space-y-2 rounded-md bg-zinc-800 p-2 shadow-sm">
                  <div className="h-2 w-[80px] rounded-lg bg-zinc-400"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-zinc-400"></div>
                </div>
                <div className="flex items-center space-x-2 rounded-md bg-zinc-800 p-2 shadow-sm">
                  <div className="h-4 w-4 rounded-full bg-zinc-400"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-zinc-400"></div>
                </div>
                <div className="flex items-center space-x-2 rounded-md bg-zinc-800 p-2 shadow-sm">
                  <div className="h-4 w-4 rounded-full bg-zinc-400"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-zinc-400"></div>
                </div>
              </div>
            </div>
            <span className="block w-full p-2 text-center font-normal">
              Dark
            </span>
          </label>
        </div>
        <div className="space-y-2">
          <label className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 border-primary">
            <button
              type="button"
              onClick={() => switchTheme("slate-dark")}
              className="aspect-square h-4 w-4 rounded-full border border-primary text-primary shadow focus:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 sr-only"
            ></button>
            <div className="items-center rounded-md border-2 border-muted bg-popover p-1 hover:bg-accent hover:text-accent-foreground">
              <div className="space-y-2 rounded-sm bg-slate-950 p-2">
                <div className="space-y-2 rounded-md bg-slate-800 p-2 shadow-sm">
                  <div className="h-2 w-[80px] rounded-lg bg-slate-400"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-slate-400"></div>
                </div>
                <div className="flex items-center space-x-2 rounded-md bg-slate-800 p-2 shadow-sm">
                  <div className="h-4 w-4 rounded-full bg-slate-400"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-slate-400"></div>
                </div>
                <div className="flex items-center space-x-2 rounded-md bg-slate-800 p-2 shadow-sm">
                  <div className="h-4 w-4 rounded-full bg-slate-400"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-slate-400"></div>
                </div>
              </div>
            </div>
            <span className="block w-full p-2 text-center font-normal">
              Slate Dark
            </span>
          </label>
        </div>
      </div>
      <Label>Topbar</Label>
      <p className="text-[0.8rem] text-muted-foreground">
        Select the topbar you'd like to use.
      </p>
      <div className="grid max-w-xl grid-cols-3 gap-8 pt-2">
        <div className="space-y-2">
          <label className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 border-primary">
            <Button
              type="button"
              onClick={() => setIsRainbowAnimationActive(false)}
              className="aspect-square h-4 w-4 rounded-full border border-primary text-primary shadow focus:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 sr-only"
            />
            <div className="items-center rounded-md border-2 border-muted bg-popover p-1 hover:bg-accent hover:text-accent-foreground">
              <div className="space-y-2 rounded-sm bg-background p-2">
                <div className="h-1 rounded-md bg-rainbow-gradient-light bg-200% w-full" />
                <div className="space-y-2 rounded-md bg-muted-foreground/20 p-2 shadow-sm">
                  <div className="h-2 w-[80px] rounded-lg bg-muted-foreground/90"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-muted-foreground/90"></div>
                </div>
                <div className="flex items-center space-x-2 rounded-md bg-muted-foreground/20 p-2 shadow-sm">
                  <div className="h-4 w-4 rounded-full bg-muted-foreground/90"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-muted-foreground/90"></div>
                </div>
                <div className="flex items-center space-x-2 rounded-md bg-muted-foreground/20 p-2 shadow-sm">
                  <div className="h-4 w-4 rounded-full bg-muted-foreground/90"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-muted-foreground/90"></div>
                </div>
              </div>
            </div>
            <span className="block w-full p-2 text-center font-normal">
              Rainbow
            </span>
          </label>
        </div>

        <div className="space-y-2">
          <label className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 border-primary">
            <Button
              type="button"
              onClick={() => setIsRainbowAnimationActive(true)}
              className="aspect-square h-4 w-4 rounded-full border border-primary text-primary shadow focus:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 sr-only"
            />
            <div className="items-center rounded-md border-2 border-muted bg-popover p-1 hover:bg-accent hover:text-accent-foreground">
              <div className="space-y-2 rounded-sm bg-background p-2">
                <div className="h-1 rounded-md bg-rainbow-gradient-light bg-200% animate-rainbow-flow" />
                <div className="space-y-2 rounded-md bg-foreground/10 p-2 shadow-sm">
                  <div className="h-2 w-[80px] rounded-lg bg-muted-foreground/90"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-muted-foreground/90"></div>
                </div>
                <div className="flex items-center space-x-2 rounded-md bg-foreground/10 p-2 shadow-sm">
                  <div className="h-4 w-4 rounded-full bg-muted-foreground/90"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-muted-foreground/90"></div>
                </div>
                <div className="flex items-center space-x-2 rounded-md bg-foreground/10 p-2 shadow-sm">
                  <div className="h-4 w-4 rounded-full bg-muted-foreground/90"></div>
                  <div className="h-2 w-[100px] rounded-lg bg-muted-foreground/90"></div>
                </div>
              </div>
            </div>
            <span className="block w-full p-2 text-center font-normal">
              Rainbow Animated
            </span>
          </label>
        </div>
      </div>
    </div>
  );
}
