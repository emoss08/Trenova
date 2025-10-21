import { LazyImage } from "@/components/ui/image";
import { UserSchema } from "@/lib/schemas/user-schema";
import { cn, truncateText } from "@/lib/utils";
import type { SuggestionProps } from "@tiptap/suggestion";
import {
  forwardRef,
  useEffectEvent,
  useImperativeHandle,
  useState,
} from "react";

export interface MentionListRef {
  onKeyDown: (props: { event: KeyboardEvent }) => boolean;
}

interface MentionListProps extends SuggestionProps {
  items: UserSchema[];
  onMentionedUsersChange?: (userIds: string[]) => void;
}

export const MentionList = forwardRef<MentionListRef, MentionListProps>(
  (props, ref) => {
    const { items } = props;
    const [selectedIndex, setSelectedIndex] = useState(0);

    const selectItem = (index: number) => {
      const item = items[index];

      if (item) {
        props.command({ id: item.id, label: item.username });
      }
    };

    const upHandler = () => {
      setSelectedIndex((selectedIndex + items.length - 1) % items.length);
    };

    const downHandler = () => {
      setSelectedIndex((selectedIndex + 1) % items.length);
    };

    const enterHandler = () => {
      selectItem(selectedIndex);
    };

    useImperativeHandle(ref, () => ({
      onKeyDown: ({ event }: { event: KeyboardEvent }) => {
        if (event.key === "ArrowUp") {
          upHandler();
          return true;
        }

        if (event.key === "ArrowDown") {
          downHandler();
          return true;
        }

        if (event.key === "Enter") {
          enterHandler();
          return true;
        }

        return false;
      },
    }));

    useEffectEvent(() => {
      setSelectedIndex(0);
    });

    return (
      <div
        className="flex flex-col gap-0.5 size-full text-popover-foreground bg-background rounded-md border border-border p-1 shadow-md"
        style={{ pointerEvents: "auto" }}
      >
        {items.length ? (
          items.map((item, index) => (
            <button
              key={index}
              onClick={() => selectItem(index)}
              className={cn(
                "flex cursor-pointer select-none items-center gap-2 focus:bg-muted-foreground/20 hover:bg-muted-foreground/20 rounded-sm px-2 py-1 text-sm outline-hidden focus:text-accent-foreground transition-colors data-disabled:opacity-50 data-disabled:pointer-events-none [&>svg]:size-3 [&>svg]:shrink-0",
                selectedIndex === index && "bg-muted-foreground/20",
              )}
              type="button"
            >
              <div className="flex flex-row items-center gap-1.5 shrink-0">
                <LazyImage
                  src={
                    item.profilePicUrl ||
                    `https://avatar.vercel.sh/${item.name}.svg`
                  }
                  alt={item.name}
                  className="size-3 rounded-full"
                />
                <span className="text-xs font-medium">
                  {truncateText(item.name, 20)}
                </span>
              </div>
            </button>
          ))
        ) : (
          <div className="flex flex-col items-center justify-center size-full">
            <span className="text-sm text-muted-foreground">No result</span>
          </div>
        )}
      </div>
    );
  },
);

MentionList.displayName = "MentionList";
