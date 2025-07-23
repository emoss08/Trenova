/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { Dialog, DialogBody, DialogContent } from "@/components/ui/dialog";
import { Icon } from "@/components/ui/icons";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import { DatabaseBackup } from "@/types/database-backup";
import { APIError } from "@/types/errors";
import { faMinus, faServer } from "@fortawesome/pro-regular-svg-icons";
import { faSort, faX } from "@fortawesome/pro-solid-svg-icons";
import { useQuery } from "@tanstack/react-query";
import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";

const asciiArt = `
 ████████╗██████╗ ███████╗███╗   ██╗ ██████╗ ██╗   ██╗  █████╗ 
 ╚══██╔══╝██╔══██╗██╔════╝████╗  ██║██╔═══██╗██║   ██║  ██╔══██╗
    ██║   ██████╔╝█████╗  ██╔██╗ ██║██║   ██║██║   ██║  ███████║
    ██║   ██╔══██╗██╔══╝  ██║╚██╗██║██║   ██║╚██╗ ██╔╝  ██╔══██║
    ██║   ██║  ██║███████╗██║ ╚████║╚██████╔╝ ╚████╔╝   ██║  ██║
    ╚═╝   ╚═╝  ╚═╝╚══════╝╚═╝  ╚═══╝ ╚═════╝   ╚═══╝    ╚═╝  ╚═╝
`;

// Simple ASCII cat for fun
const catArt = `
 /\\_/\\
( o.o )
 > ^ <
`;

// Types definitions
type TerminalLine = {
  id: string;
  content: string;
  type:
    | "command"
    | "output"
    | "error"
    | "success"
    | "info"
    | "hint"
    | "detail"
    | "header"
    | "dropping"
    | "creating"
    | "processing"
    | "prompt";
  timestamp: Date;
  animation?: "typing" | "fade" | "none";
  delay?: number;
};

type RestoreStatus = "idle" | "ready" | "pending" | "success" | "error";

interface ErrorDetails {
  status?: number;
  title?: string;
  detail?: string;
}

const TerminalLineComponent = memo(function TerminalLineComponent({
  line,
}: {
  line: TerminalLine;
}) {
  // Color classes based on line type - memoized to prevent re-renders
  const colorClass = useMemo(() => {
    switch (line.type) {
      case "command":
        return "text-emerald-400 font-semibold";
      case "error":
        return "text-red-400";
      case "success":
        return "text-green-400";
      case "info":
        return "text-blue-400";
      case "hint":
        return "text-amber-400 italic";
      case "detail":
        return "text-gray-400 italic";
      case "header":
        return "text-fuchsia-400 font-bold";
      case "dropping":
        return "text-amber-300";
      case "creating":
        return "text-emerald-300";
      case "processing":
        return "text-cyan-300";
      case "prompt":
        return "text-yellow-300 font-semibold";
      default:
        return "text-gray-200";
    }
  }, [line.type]);

  // Animations are added via CSS classes - but only applied to specific line types
  const animationClass = useMemo(() => {
    // Only apply typing animation to headers for better emphasis
    if (line.animation === "typing" && line.type === "header") {
      return "terminal-typing";
    }
    // Use fade-in for all other animated content
    if (line.animation === "fade") {
      return "terminal-fade-in";
    }
    return "";
  }, [line.animation, line.type]);

  // For header type lines containing ASCII art, use pre tag with careful styling
  if (line.type === "header") {
    return (
      <pre
        className={`terminal-line ${colorClass} ${animationClass} m-0`}
        style={{
          fontFamily: "monospace",
          lineHeight: 1.2,
          overflow: "visible",
          marginBottom: "0.5rem",
          whiteSpace: "pre",
          fontSize: "0.8rem",
          letterSpacing: 0,
        }}
      >
        {line.content}
      </pre>
    );
  }

  return (
    <div
      className={cn(
        "terminal-line text-wrap whitespace-pre-wrap",
        colorClass,
        animationClass,
      )}
    >
      {line.content}
    </div>
  );
});

// Main terminal dialog component
function TerminalRestoreDialog({
  backup,
  open,
  onOpenChange,
  restoreMutation,
}: {
  backup: DatabaseBackup;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  restoreMutation: any;
}) {
  const [status, setStatus] = useState<RestoreStatus>("idle");
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [_, setErrorDetails] = useState<ErrorDetails | null>(null);
  const [terminalLines, setTerminalLines] = useState<TerminalLine[]>([]);
  const [inputValue, setInputValue] = useState<string>("");
  const [commandHistory, setCommandHistory] = useState<string[]>([]);
  const [historyIndex, setHistoryIndex] = useState<number>(-1);

  const terminalRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const lineProcessorRef = useRef<number | null>(null);
  const linesQueue = useRef<TerminalLine[]>([]);
  const autoScrollerRef = useRef<HTMLDivElement>(null);

  const { data: backupData } = useQuery({
    ...queries.organization.getDatabaseBackups(),
  });

  // Database connection config
  const dbConfig = useMemo(
    () => ({
      host: "localhost",
      port: 5432,
      username: "postgres",
      database: "postgres",
    }),
    [],
  );

  // Format pg_restore command
  const pgRestoreCommand = useMemo(() => {
    return [
      `pg_restore \\`,
      `  --host=${dbConfig.host} \\`,
      `  --port=${dbConfig.port} \\`,
      `  --username=${dbConfig.username} \\`,
      `  --clean \\`,
      `  --if-exists \\`,
      `  --no-owner \\`,
      `  --no-privileges \\`,
      `  --verbose \\`,
      `  --dbname=${dbConfig.database} \\`,
      `  ${backup.filename}`,
    ].join("\n");
  }, [backup.filename, dbConfig]);

  // Terminal command help text
  const helpText = `
Available commands:
  help          - Display this help message
  clear         - Clear the terminal screen
  ls            - List files in current directory
  cat           - Show the content of a file
  date          - Display current date and time
  whoami        - Show current user
  ascii         - Toggle ASCII art style
  fredrick      - Show a cool cat
  exit          - Close the terminal
  
Press ENTER to start/retry the database restore.
`;

  // Focus input field
  const focusInput = useCallback(() => {
    if (status === "ready" && inputRef.current) {
      inputRef.current.focus();
    }
  }, [status]);

  // Toggle between ASCII art versions
  const toggleAsciiArt = useCallback(() => {
    // Update the header line if it exists
    setTerminalLines((prev) => {
      const newLines = [...prev];
      const headerIndex = newLines.findIndex((line) => line.type === "header");

      if (headerIndex !== -1) {
        newLines[headerIndex] = {
          ...newLines[headerIndex],
          content: asciiArt,
        };
      }

      return newLines;
    });
  }, []);

  // Auto-scroll terminal to bottom when new lines are added
  useEffect(() => {
    if (autoScrollerRef.current) {
      autoScrollerRef.current.scrollIntoView({ behavior: "auto" });
    }
  }, [terminalLines]);

  // Focus the input when dialog opens or status changes
  useEffect(() => {
    if (open && status === "ready" && inputRef.current) {
      // Short delay to ensure the DOM is ready
      setTimeout(() => {
        inputRef.current?.focus();
      }, 100);
    }
  }, [open, status]);

  // Reset state when dialog opens/closes
  useEffect(() => {
    if (open) {
      // Initialize terminal with welcome message
      const initialLines: TerminalLine[] = [
        {
          id: `header-${Date.now()}-1`,
          content: asciiArt,
          type: "header",
          timestamp: new Date(),
          animation: "none",
          delay: 0,
        },
        {
          id: `welcome-${Date.now()}-2`,
          content: `Started at ${new Date().toLocaleString()}`,
          type: "info",
          timestamp: new Date(),
          animation: "fade",
          delay: 200,
        },
        {
          id: `info-${Date.now()}-3`,
          content: `Preparing to restore: ${backup.filename}`,
          type: "info",
          timestamp: new Date(),
          animation: "fade",
          delay: 200,
        },
        {
          id: `hint-${Date.now()}-4`,
          content: `Type 'help' for available commands`,
          type: "hint",
          timestamp: new Date(),
          animation: "fade",
          delay: 200,
        },
        {
          id: `blank-${Date.now()}-5`,
          content: "",
          type: "output",
          timestamp: new Date(),
          animation: "none",
          delay: 0,
        },
        {
          id: `prompt-${Date.now()}-6`,
          content: "Press ENTER to start database restore...",
          type: "prompt",
          timestamp: new Date(),
          animation: "fade",
          delay: 300,
        },
      ];

      setTerminalLines(initialLines);
      setStatus("ready");
    } else if (!restoreMutation.isPending) {
      // Only reset if we're not in the middle of an operation
      setStatus("idle");
      setErrorDetails(null);
      setTerminalLines([]);
      setInputValue("");
      setCommandHistory([]);
      setHistoryIndex(-1);

      // Clean up any pending timeouts
      if (lineProcessorRef.current) {
        clearTimeout(lineProcessorRef.current);
        lineProcessorRef.current = null;
      }
      linesQueue.current = [];
    }
  }, [open, backup.filename, restoreMutation.isPending, pgRestoreCommand]);

  // Process lines from the queue with appropriate delays
  const processNextLine = useCallback(() => {
    if (linesQueue.current.length === 0) {
      lineProcessorRef.current = null;
      return;
    }

    const nextLine = linesQueue.current.shift()!;

    // Use smaller delays for better UX
    const delay =
      nextLine.delay ||
      (nextLine.type === "header"
        ? 100
        : nextLine.type === "command"
          ? 30
          : nextLine.type === "error"
            ? 30
            : 10); // Fast display for most output

    // Add the line to the terminal
    setTerminalLines((lines) => [...lines, nextLine]);

    // Schedule processing of the next line
    lineProcessorRef.current = window.setTimeout(() => {
      processNextLine();
    }, delay);
  }, []);

  // Add line to terminal with controlled animation
  const addTerminalLine = useCallback(
    (
      content: string,
      type: TerminalLine["type"] = "output",
      animation: TerminalLine["animation"] = "fade",
      delay?: number,
    ) => {
      const newLine = {
        id: `${type}-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`,
        content,
        type,
        timestamp: new Date(),
        // Only allow typing animation for headers, use fade for everything else
        animation:
          animation === "typing" && type !== "header" ? "fade" : animation,
        delay,
      };

      linesQueue.current.push(newLine);

      // If this is the first line being added, start the processor
      if (linesQueue.current.length === 1 && !lineProcessorRef.current) {
        processNextLine();
      }
    },
    [processNextLine],
  );

  // Handle dialog close attempt
  const handleOpenChange = useCallback(
    (isOpen: boolean) => {
      if (!isOpen && restoreMutation.isPending) {
        // Prevent closing during operation
        return;
      }
      onOpenChange(isOpen);
    },
    [onOpenChange, restoreMutation.isPending],
  );

  // Process command entered by user
  const processCommand = useCallback(
    (command: string) => {
      if (!command.trim()) return;

      // Add command to history
      setCommandHistory((prev) => [...prev, command]);
      setHistoryIndex(-1);

      // Echo the command
      addTerminalLine(`$ ${command}`, "command", "none", 0);

      // Process command
      const lowercaseCommand = command.trim().toLowerCase();
      const parts = lowercaseCommand.split(" ");
      const mainCommand = parts[0];

      switch (mainCommand) {
        case "help":
          addTerminalLine(helpText, "info", "fade", 50);
          break;

        case "clear":
          setTerminalLines([]);
          break;

        case "ls":
          // Simulate listing files
          if (parts[1] === "-la") {
            addTerminalLine(
              `total 16
drwxr-xr-x  2 postgres postgres 4096 Apr  5 15:30 .
drwxr-xr-x 12 postgres postgres 4096 Apr  5 15:29 ..
-rw-r--r--  1 postgres postgres  220 Apr  5 15:20 .bash_logout
-rw-r--r--  1 postgres postgres 3526 Apr  5 15:20 .bashrc
-rw-r--r--  1 postgres postgres  807 Apr  5 15:20 .profile
${backupData?.data.backups.map((backup) => `-rw-r--r--  1 postgres postgres   15 Apr  5 15:30 ${backup.filename}`).join("\n")}`,
              "output",
              "none",
              50,
            );
          } else {
            addTerminalLine(backup.filename, "output", "none", 50);
          }
          break;

        case "cat":
          // Simulate cat command
          if (parts[1] === backup.filename) {
            addTerminalLine(
              "(binary data - cannot display content)",
              "output",
              "none",
              50,
            );
          } else if (parts[1]) {
            addTerminalLine(
              `cat: ${parts[1]}: No such file or directory`,
              "error",
              "none",
              50,
            );
          } else {
            addTerminalLine("Usage: cat [filename]", "output", "none", 50);
          }
          break;

        case "date":
          // Show current date and time
          addTerminalLine(new Date().toString(), "output", "none", 50);
          break;

        case "whoami":
          // Show current user
          addTerminalLine("postgres", "output", "none", 50);
          break;

        case "ascii":
          // Toggle ASCII art
          toggleAsciiArt();
          addTerminalLine("ASCII header refreshed", "success", "fade", 50);
          break;

        case "fredrick":
          // Show cat
          addTerminalLine(catArt, "header", "none", 50);
          break;

        case "exit":
          // Close the terminal
          handleOpenChange(false);
          break;

        case "sudo":
          // Funny sudo response
          addTerminalLine(
            "postgres is not in the sudoers file. This incident will be reported.",
            "error",
            "none",
            50,
          );
          break;

        default:
          // Command not found
          addTerminalLine(
            `Command not found: ${command}. Type 'help' for available commands.`,
            "error",
            "none",
            50,
          );
      }
    },
    [
      addTerminalLine,
      backup.filename,
      handleOpenChange,
      toggleAsciiArt,
      helpText,
      backupData?.data.backups,
    ],
  );

  // Format and process pg_restore output
  const processRestoreOutput = useCallback(
    (output: string) => {
      if (!output) return;

      // Split by newlines and process each line
      const lines = output.split("\n");

      // Process in batches for better performance with long outputs
      for (let i = 0; i < lines.length; i++) {
        const line = lines[i].trim();
        if (!line) continue;

        // Determine line type and add to terminal - no animations for most pg_restore output
        if (line.includes("pg_restore:")) {
          if (line.includes("dropping")) {
            addTerminalLine(line, "dropping", "fade", 5);
          } else if (line.includes("creating")) {
            addTerminalLine(line, "creating", "fade", 5);
          } else if (line.includes("processing")) {
            addTerminalLine(line, "processing", "fade", 5);
          } else {
            addTerminalLine(line, "output", "none", 3);
          }
        } else if (line.includes("ERROR:")) {
          addTerminalLine(line, "error", "none", 3);
        } else if (line.includes("HINT:")) {
          addTerminalLine(line, "hint", "none", 3);
        } else if (line.includes("DETAIL:")) {
          addTerminalLine(line, "detail", "none", 3);
        } else {
          addTerminalLine(line, "output", "none", 3);
        }
      }
    },
    [addTerminalLine],
  );

  // Process error details for terminal display
  const processErrorForTerminal = useCallback(
    (error: any) => {
      if (!error) return;

      // Add error header line with more emphasis
      addTerminalLine(
        "Encountered an error during restore process:",
        "error",
        "fade",
        100,
      );

      // Process API Error type
      if (error instanceof APIError) {
        // Add type and status
        addTerminalLine(
          `Error Type: ${error.data?.type || "API Error"}`,
          "error",
          "none",
          50,
        );
        addTerminalLine(`Status: ${error.status || 500}`, "error", "none", 5);

        // Add title if available
        if (error.data?.title) {
          addTerminalLine(`Error: ${error.data?.title}`, "error", "none", 5);
        }

        // Process error detail
        const detail = error.data?.detail;
        if (detail) {
          addTerminalLine("Error Details:", "error", "none", 5);
          processRestoreOutput(detail);
        }
      }
      // Process standard Error type
      else if (error instanceof Error) {
        addTerminalLine(
          `Error: ${error.name || "Unknown Error"}`,
          "error",
          "none",
          5,
        );

        if (error.message) {
          // Try to extract JSON from error message
          try {
            const jsonMatch = error.message.match(/{.*}/s);
            if (jsonMatch) {
              const jsonStr = jsonMatch[0];
              const parsed = JSON.parse(jsonStr);

              // Process parsed error data
              if (parsed.status) {
                addTerminalLine(`Status: ${parsed.status}`, "error", "none", 5);
              }
              if (parsed.title) {
                addTerminalLine(`Error: ${parsed.title}`, "error", "none", 5);
              }
              if (parsed.detail) {
                addTerminalLine("Error Details:", "error", "none", 5);
                processRestoreOutput(parsed.detail);
              }
            } else {
              // Regular error message
              addTerminalLine(error.message, "error", "none", 5);
            }
          } catch {
            // Fallback for parsing errors
            addTerminalLine(error.message, "error", "none", 5);
          }
        }
      }
      // Handle object-type errors
      else if (error && typeof error === "object") {
        const errorObj = error as Record<string, any>;

        // Extract common error properties
        for (const [key, value] of Object.entries(errorObj)) {
          if (key === "detail" && typeof value === "string") {
            addTerminalLine("Error Details:", "error", "none", 5);
            processRestoreOutput(value);
          } else if (key === "message" && typeof value === "string") {
            addTerminalLine(value, "error", "none", 5);
          } else if (key === "status" || key === "title" || key === "type") {
            addTerminalLine(`${key}: ${value}`, "error", "none", 5);
          }
        }
      }
      // Handle primitive error types
      else {
        addTerminalLine(String(error), "error", "none", 5);
      }

      // Summary message at end of error output
      addTerminalLine(
        "Restore operation failed. See details above.",
        "error",
        "none",
        5,
      );

      // Add prompt to try again
      addTerminalLine("", "output", "none", 5);
      addTerminalLine("Press ENTER to try again...", "prompt", "fade", 100);
      setStatus("ready");
    },
    [addTerminalLine, processRestoreOutput],
  );

  // Start restore process
  async function handleRestore() {
    if (status === "pending") return; // Prevent multiple restore attempts

    // Clear existing lines and set status
    setStatus("pending");
    setErrorDetails(null);
    setTerminalLines([]);
    setInputValue("");

    // Add command prompt and command with subtle animation
    addTerminalLine("postgres@db:~$ ", "command", "fade", 10);

    // Command gets simple fade in, not typing animation
    addTerminalLine(pgRestoreCommand, "command", "fade", 100);

    // Add connecting message
    addTerminalLine("", "output", "none", 100);
    addTerminalLine("Connecting to database...", "output", "fade", 100);

    try {
      // Show preparatory messages with minimal delays for good UX
      addTerminalLine(
        "Checking backup file integrity...",
        "output",
        "fade",
        100,
      );
      addTerminalLine(
        "Verifying database connection...",
        "output",
        "fade",
        100,
      );
      addTerminalLine(
        "Starting database restore process...",
        "output",
        "fade",
        100,
      );
      addTerminalLine(
        "This may take several minutes depending on the backup size.",
        "info",
        "fade",
        100,
      );
      addTerminalLine("", "output", "none", 50);

      // Actual restore operation
      const result = await restoreMutation.mutateAsync();

      // Success messages - use slightly longer delays for emphasis
      setStatus("success");
      addTerminalLine("", "output", "none", 100);
      addTerminalLine(
        "Database restore completed successfully!",
        "success",
        "fade",
        100,
      );
      addTerminalLine(
        "All database objects have been restored.",
        "success",
        "fade",
        100,
      );
      addTerminalLine(`Backup file: ${backup.filename}`, "info", "fade", 50);
      addTerminalLine(
        `Restore completed at: ${new Date().toLocaleString()}`,
        "info",
        "fade",
        50,
      );
      addTerminalLine("", "output", "none", 50);
      addTerminalLine("postgres@db:~$ ", "command", "fade", 100);
      addTerminalLine("Press ENTER to close...", "prompt", "fade", 100);

      // Reset status to ready so we can handle the enter key to close
      setStatus("ready");

      return result;
    } catch (error) {
      console.error("Original error caught in handleRestore:", error);
      setStatus("error");

      // Extract error details for display
      if (error instanceof APIError) {
        setErrorDetails({
          status: error.status,
          title: error.data?.title || "Restore Failed",
          detail: error.data?.detail || error.message,
        });
      } else if (error instanceof Error) {
        try {
          // Try to parse error message for API error details
          let errorData: ErrorDetails = {
            title: "Restore Failed",
            detail: error.message,
          };

          const jsonMatch = error.message.match(/{.*}/s);
          if (jsonMatch) {
            try {
              const jsonString = jsonMatch[0];
              const parsed = JSON.parse(jsonString);

              errorData = {
                status: parsed.status,
                title: parsed.title || "Restore Failed",
                detail: parsed.detail || error.message,
              };
            } catch (jsonError) {
              console.error(
                "Failed to parse JSON from error message",
                jsonError,
              );
            }
          }

          setErrorDetails(errorData);
        } catch (parseError) {
          console.error("Error handling failed:", parseError);
          setErrorDetails({
            title: "Restore Failed",
            detail: error.message,
          });
        }
      } else if (error && typeof error === "object") {
        // Extract properties from unknown error object
        const errorObj = error as any;
        setErrorDetails({
          status: errorObj.status || errorObj.statusCode,
          title: errorObj.title || errorObj.name || "Restore Failed",
          detail:
            errorObj.detail ||
            errorObj.message ||
            JSON.stringify(error, null, 2),
        });
      } else {
        setErrorDetails({
          title: "Restore Failed",
          detail:
            String(error) ||
            "An unknown error occurred during the restore process.",
        });
      }

      // Add a blank line before error details for readability
      addTerminalLine("", "output", "none", 100);

      // Process error for terminal display
      processErrorForTerminal(error);
    }
  }

  // Handle key presses in terminal
  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.preventDefault();

      // If there's text in the input, process it as a command
      if (inputValue.trim()) {
        processCommand(inputValue.trim());
        setInputValue("");
        return;
      }

      // Otherwise, handle Enter based on current status
      if (status === "ready") {
        if (
          terminalLines.some((line) =>
            line.content.includes("Press ENTER to close..."),
          )
        ) {
          // Close the dialog on enter if we're showing the close prompt
          handleOpenChange(false);
        } else if (
          terminalLines.some((line) =>
            line.content.includes("Press ENTER to try again..."),
          )
        ) {
          // Retry the restore on enter if there was an error
          handleRestore();
        } else {
          // Start the restore if we're showing the initial prompt
          handleRestore();
        }
      }
    } else if (e.key === "ArrowUp") {
      // Navigate command history upwards
      e.preventDefault();
      if (
        commandHistory.length > 0 &&
        historyIndex < commandHistory.length - 1
      ) {
        const newIndex = historyIndex + 1;
        setHistoryIndex(newIndex);
        setInputValue(commandHistory[commandHistory.length - 1 - newIndex]);
      }
    } else if (e.key === "ArrowDown") {
      // Navigate command history downwards
      e.preventDefault();
      if (historyIndex > 0) {
        const newIndex = historyIndex - 1;
        setHistoryIndex(newIndex);
        setInputValue(commandHistory[commandHistory.length - 1 - newIndex]);
      } else if (historyIndex === 0) {
        setHistoryIndex(-1);
        setInputValue("");
      }
    } else if (e.key === "Tab") {
      // Simple tab completion
      e.preventDefault();

      const currentInput = inputValue.trim().toLowerCase();
      const commands = [
        "help",
        "clear",
        "ls",
        "cat",
        "date",
        "whoami",
        "ascii",
        "fredrick",
        "exit",
        "sudo",
      ];

      const matches = commands.filter((cmd) => cmd.startsWith(currentInput));

      if (matches.length === 1) {
        setInputValue(matches[0]);
      } else if (matches.length > 1 && currentInput) {
        // Show options
        addTerminalLine("$ " + currentInput, "command", "none", 0);
        addTerminalLine(matches.join("  "), "output", "none", 0);
      }
    }
  };

  // Cleanup timeouts on unmount
  useEffect(() => {
    return () => {
      if (lineProcessorRef.current) {
        clearTimeout(lineProcessorRef.current);
      }
    };
  }, []);

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-4xl p-0" withClose={false}>
        <DialogBody className="p-0">
          <div className="size-full">
            <div className="terminal-header flex justify-between items-center p-2 bg-zinc-900 rounded-t-md border border-zinc-700">
              <div className="terminal-title flex items-center text-white text-sm font-mono">
                <Icon icon={faServer} className="size-3.5 mr-2" />
                <span>
                  pg_restore @ {dbConfig.host}:{dbConfig.port}
                </span>
              </div>
              <div className="group terminal-controls flex space-x-1.5">
                <div
                  className="flex items-center justify-center size-4 rounded-full bg-red-500"
                  onClick={() => handleOpenChange(false)}
                >
                  <Icon
                    icon={faX}
                    className="size-2.5 text-black opacity-0 group-hover:opacity-60 transition-opacity"
                  />
                </div>
                <div
                  className="flex items-center justify-center size-4 rounded-full bg-yellow-500"
                  onClick={() => handleOpenChange(false)}
                >
                  <Icon
                    icon={faMinus}
                    className="size-2.5 text-black opacity-0 group-hover:opacity-60 transition-opacity"
                  />
                </div>
                <div
                  className="flex items-center justify-center size-4 rounded-full bg-green-500"
                  onClick={() => handleOpenChange(false)}
                >
                  <Icon
                    icon={faSort}
                    className="size-3 rotate-45 text-black opacity-0 group-hover:opacity-60 transition-opacity"
                  />
                </div>
              </div>
            </div>
            <div
              ref={terminalRef}
              className="h-[400px] rounded-b-md bg-zinc-900 border border-t-0 border-zinc-700 p-4 font-mono text-sm overflow-auto relative"
              onClick={focusInput}
            >
              <div className="terminal-content space-y-0.5">
                {terminalLines.map((line) => (
                  <TerminalLineComponent key={line.id} line={line} />
                ))}
                {status === "pending" && (
                  <div className="terminal-cursor h-4 mt-1">
                    <span className="text-gray-300 terminal-blink">▊</span>
                  </div>
                )}
                {/* Input area */}
                <div className="relative flex items-center">
                  {status === "ready" && (
                    <>
                      <span className="text-emerald-400 mr-2">$</span>
                      <input
                        ref={inputRef}
                        type="text"
                        className="bg-transparent border-none outline-none text-white flex-1 focus:ring-0"
                        value={inputValue}
                        onChange={(e) => setInputValue(e.target.value)}
                        onKeyDown={handleKeyDown}
                        autoFocus
                      />
                    </>
                  )}
                </div>
                <div ref={autoScrollerRef} />
              </div>
            </div>
          </div>
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}

export { TerminalRestoreDialog };

