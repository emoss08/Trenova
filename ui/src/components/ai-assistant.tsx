import {
    faArrowUp,
    faFingerprint,
    faRefresh,
    faSparkles,
} from "@fortawesome/pro-regular-svg-icons";
import { useEffect, useState } from "react";
import { Button } from "./ui/button";
import { PopoverForm } from "./ui/form-popover";
import { Icon } from "./ui/icons";
import { Input } from "./ui/input";
import { ScrollArea } from "./ui/scroll-area";

interface Message {
  type: "user" | "ai";
  content: string;
}

interface MessageProps {
  message: string;
}

interface AIAssistantFieldProps {
  value: string;
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  onKeyDown: (e: React.KeyboardEvent<HTMLInputElement>) => void;
  onSend: () => void;
}

// Custom hook for text streaming effect
function useTextStream(text: string, isStreaming: boolean) {
  const [displayedText, setDisplayedText] = useState("");
  const [isComplete, setIsComplete] = useState(false);

  useEffect(() => {
    if (!isStreaming) {
      setDisplayedText(text);
      setIsComplete(true);
      return;
    }

    setDisplayedText("");
    setIsComplete(false);
    let index = 0;

    const interval = setInterval(() => {
      if (index <= text.length) {
        setDisplayedText(text.slice(0, index));
        index++;
      } else {
        clearInterval(interval);
        setIsComplete(true);
      }
    }, 30); // Adjust speed as needed

    return () => clearInterval(interval);
  }, [text, isStreaming]);

  return { displayedText, isComplete };
}

// Message Components
function UserMessage({ message }: MessageProps) {
  return (
    <div className="flex justify-end mb-3">
      <div className="bg-primary text-primary-foreground rounded-md py-2 px-3 max-w-[85%]">
        <p className="text-sm">{message}</p>
      </div>
    </div>
  );
}

// Updated AI Message component with streaming
function AIMessage({
  message,
  isStreaming = false,
}: {
  message: string;
  isStreaming?: boolean;
}) {
  const { displayedText } = useTextStream(message, isStreaming);

  return (
    <div className="flex mb-3">
      <div className="bg-muted rounded-md py-2 px-3 max-w-[85%]">
        <p className="text-sm whitespace-pre-wrap">{displayedText}</p>
      </div>
    </div>
  );
}

function LoadingMessage() {
  return (
    <div className="flex mb-3">
      <div className="bg-muted rounded-2xl rounded-tl-sm py-2 px-3">
        <div className="flex gap-1.5 items-center py-1">
          <div className="size-2 bg-primary/30 animate-pulse rounded-full" />
          <div className="size-2 bg-primary/30 animate-pulse rounded-full [animation-delay:0.2s]" />
          <div className="size-2 bg-primary/30 animate-pulse rounded-full [animation-delay:0.4s]" />
        </div>
      </div>
    </div>
  );
}

export function AIAssistantHeader() {
  return (
    <div className="flex flex-col items-center justify-center pt-14 px-6">
      <div className="relative flex flex-col items-center w-full max-w-[300px]">
        {/* Beta Tag */}
        <div className="absolute -right-2 -top-6 flex items-center gap-1">
          <span className="inline-flex items-center rounded-full bg-primary/10 gap-1 px-2 py-0.5 text-2xs font-medium text-primary ring-1 ring-inset ring-primary/20">
            <Icon icon={faSparkles} className="size-3 text-primary/70" />
            BETA
          </span>
        </div>

        {/* Main Header */}
        <div className="flex flex-col items-center gap-4">
          <div className="relative">
            <div className="absolute -inset-0.5 rounded-lg bg-gradient-to-r from-violet-500/30 to-fuchsia-500/30 blur-sm" />
            <div className="relative flex items-center justify-center rounded-lg bg-gradient-to-bl from-violet-500 to-fuchsia-500 p-3">
              <Icon icon={faFingerprint} className="size-6 text-white" />
            </div>
          </div>

          <div className="space-y-1 text-center">
            <h2 className="text-2xl font-semibold tracking-tight">
              Trenova Assistant
            </h2>
            <p className="text-sm font-medium text-muted-foreground">
              Your AI-Powered Logistics Partner
            </p>
          </div>
        </div>

        {/* Description */}
        <div className="mt-6 space-y-4 w-full">
          <p className="text-center text-xs leading-normal text-muted-foreground">
            Powered by advanced AI, I can help you with:
          </p>
          <div className="grid grid-cols-2 gap-2 text-center text-2xs text-muted-foreground">
            <Button variant="ghost" className="rounded-md bg-muted p-2">
              Shipment Tracking
            </Button>
            <Button variant="ghost" className="rounded-md bg-muted p-2">
              Billing Inquiries
            </Button>
            <Button variant="ghost" className="rounded-md bg-muted p-2">
              Data Analysis
            </Button>
            <Button variant="ghost" className="rounded-md bg-muted p-2">
              Real-time Updates
            </Button>
          </div>
        </div>

        {/* Separator */}
        <div className="mt-6 h-px w-full bg-gradient-to-r from-transparent via-border/60 to-transparent" />
      </div>
    </div>
  );
}

// Field Component with Props
function AIAssistantField({
  value,
  onChange,
  onKeyDown,
  onSend,
}: AIAssistantFieldProps) {
  return (
    <div className="flex flex-col items-center justify-center w-full relative">
      <div className="relative w-full">
        <Input
          value={value}
          onChange={onChange}
          placeholder="Ask me anything..."
          className="w-full pr-12"
          onKeyDown={onKeyDown}
        />
        <Button
          size="icon"
          onClick={onSend}
          className="absolute right-1 top-1/2 h-6 w-6 -translate-y-1/2 bg-primary hover:bg-primary/90"
        >
          <Icon icon={faArrowUp} className="size-3 text-primary-foreground" />
        </Button>
      </div>
    </div>
  );
}

function AIAssistantBody() {
  const [messages, setMessages] = useState<
    (Message & { isStreaming?: boolean })[]
  >([]);
  const [isLoading, setIsLoading] = useState(false);
  const [inputValue, setInputValue] = useState("");

  const simulateResponse = async (): Promise<void> => {
    setIsLoading(true);
    await new Promise((resolve) => setTimeout(resolve, 1000));
    setIsLoading(false);

    const response = {
      type: "ai" as const,
      content:
        "I can help you track that shipment. Could you please provide the tracking number or PRO number?",
      isStreaming: true,
    };

    setMessages((prev) => [...prev, response]);

    // After streaming is complete, update the message
    await new Promise((resolve) => setTimeout(resolve, 2000));
    setMessages((prev) =>
      prev.map((msg, idx) =>
        idx === prev.length - 1 ? { ...msg, isStreaming: false } : msg,
      ),
    );
  };

  // Rest of the component remains the same
  const handleSend = (): void => {
    if (!inputValue.trim()) return;

    setMessages((prev) => [...prev, { type: "user", content: inputValue }]);
    setInputValue("");
    simulateResponse();
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>): void => {
    if (e.key === "Enter") {
      handleSend();
    }
  };

  const handleReset = (): void => {
    setMessages([]);
    setInputValue("");
    setIsLoading(false);
  };

  return (
    <div className="flex flex-col w-full h-full mt-10 max-h-[430px]">
      {messages.length === 0 ? (
        <AIAssistantHeader />
      ) : (
        <>
          <div className="absolute right-10 top-2 z-20">
            <Button
              variant="ghost"
              size="sm"
              onClick={handleReset}
              className="rounded-sm px-1.5 transition-[border-color,box-shadow] duration-100 ease-in-out focus:border focus:border-blue-600 focus:outline-hidden focus:ring-4 focus:ring-blue-600/20 disabled:pointer-events-none"
            >
              <Icon icon={faRefresh} className="size-4" />
              <span className="sr-only">Close</span>
            </Button>
          </div>
          <ScrollArea className="relative flex-1 overflow-hidden max-h-[400px] mt-auto p-4">
            <div className="overflow-y-auto pr-2">
              {messages.map((message, index) =>
                message.type === "user" ? (
                  <UserMessage key={index} message={message.content} />
                ) : (
                  <AIMessage
                    key={index}
                    message={message.content}
                    isStreaming={message.isStreaming}
                  />
                ),
              )}
              {isLoading && <LoadingMessage />}
            </div>
            <div className="pointer-events-none absolute bottom-0 left-0 right-0 h-8 bg-gradient-to-t from-sidebar to-transparent" />
          </ScrollArea>
        </>
      )}

      {/* Fixed Input Area */}
      <div className="mt-auto pt-2 px-4">
        <AIAssistantField
          value={inputValue}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
            setInputValue(e.target.value)
          }
          onKeyDown={handleKeyDown}
          onSend={handleSend}
        />
        <p className="text-2xs font-medium text-center text-muted-foreground mt-2">
          Chatbots can make mistakes. Check important info.
        </p>
      </div>
    </div>
  );
}

export function AIAssistant() {
  const [open, setOpen] = useState(false);

  return (
    <PopoverForm
      title="AI Assistant"
      open={open}
      setOpen={setOpen}
      width="400px"
      height="480px"
      showCloseButton
      openChild={<AIAssistantBody />}
    />
  );
}
