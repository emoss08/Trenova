import { AnimatePresence, m } from "motion/react";
import type { TypingUser } from "@/hooks/shipment-comments/use-shipment-typing";

function typingLabel(users: TypingUser[]): string {
  if (users.length === 1) return `${users[0].name} is typing…`;
  if (users.length === 2) return `${users[0].name} and ${users[1].name} are typing…`;
  return "Several people are typing…";
}

export function TypingIndicator({ typingUsers }: { typingUsers: TypingUser[] }) {
  return (
    <div className="flex h-5 items-center px-4" aria-live="polite">
      <AnimatePresence>
        {typingUsers.length > 0 && (
          <m.div
            initial={{ opacity: 0, y: 3 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 3 }}
            transition={{ duration: 0.15, ease: "easeOut" }}
            className="text-2xs text-muted-foreground flex items-center gap-1.5"
          >
            <span className="flex items-center gap-0.5">
              {[0, 1, 2].map((dot) => (
                <m.span
                  key={dot}
                  className="bg-muted-foreground size-1 rounded-full"
                  animate={{ opacity: [0.3, 1, 0.3] }}
                  transition={{
                    duration: 1.1,
                    repeat: Infinity,
                    delay: dot * 0.18,
                    ease: "easeInOut",
                  }}
                />
              ))}
            </span>
            {typingLabel(typingUsers)}
          </m.div>
        )}
      </AnimatePresence>
    </div>
  );
}
