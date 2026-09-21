import { useCallback, useEffect, useRef, useState } from "react";

/**
 * Speaking into the composer, on browsers that transcribe locally or through
 * the vendor's service. Interim words are shown as they come and replaced
 * by the final phrase, so what lands in the draft is what was recognised
 * last, not every guess along the way.
 */
export type DictationState = {
  /** Whether this browser can transcribe speech at all. */
  supported: boolean;
  listening: boolean;
  /** Why the last attempt stopped, when it was not the person stopping it. */
  error: string | null;
  start: () => void;
  stop: () => void;
  toggle: () => void;
};

function recognitionConstructor(): typeof SpeechRecognition | null {
  if (typeof window === "undefined") {
    return null;
  }

  return window.SpeechRecognition ?? window.webkitSpeechRecognition ?? null;
}

/**
 * `onTranscript` receives the finished phrases as they are recognised, and
 * `onInterim` the guess in progress, so the caller can show the guess and
 * commit the phrase.
 */
export function useDictation({
  lang,
  onTranscript,
  onInterim,
}: {
  lang?: string;
  onTranscript: (text: string) => void;
  onInterim?: (text: string) => void;
}): DictationState {
  const [listening, setListening] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const recognitionRef = useRef<SpeechRecognition | null>(null);
  const onTranscriptRef = useRef(onTranscript);
  const onInterimRef = useRef(onInterim);
  useEffect(() => {
    onTranscriptRef.current = onTranscript;
    onInterimRef.current = onInterim;
  });

  const supported = recognitionConstructor() !== null;

  const stop = useCallback(() => {
    recognitionRef.current?.stop();
    recognitionRef.current = null;
    setListening(false);
    onInterimRef.current?.("");
  }, []);

  const start = useCallback(() => {
    const Recognition = recognitionConstructor();
    if (!Recognition || recognitionRef.current) {
      return;
    }
    const recognition = new Recognition();
    recognition.lang = lang ?? (typeof navigator === "undefined" ? "en-US" : navigator.language);
    recognition.continuous = true;
    recognition.interimResults = true;
    recognition.maxAlternatives = 1;
    recognition.onresult = (event) => {
      let interim = "";
      for (let index = event.resultIndex; index < event.results.length; index += 1) {
        const result = event.results[index];
        const transcript = result[0]?.transcript ?? "";
        if (result.isFinal) {
          onTranscriptRef.current(transcript.trim());
        } else {
          interim += transcript;
        }
      }
      onInterimRef.current?.(interim.trim());
    };
    recognition.onerror = (event) => {
      // "aborted" and "no-speech" are the person stopping or saying nothing;
      // neither is worth a message.
      if (event.error !== "aborted" && event.error !== "no-speech") {
        setError(event.error);
      }
    };
    recognition.onend = () => {
      recognitionRef.current = null;
      setListening(false);
      onInterimRef.current?.("");
    };
    recognitionRef.current = recognition;
    setError(null);
    setListening(true);
    recognition.start();
  }, [lang]);

  const toggle = useCallback(() => {
    if (recognitionRef.current) {
      stop();
    } else {
      start();
    }
  }, [start, stop]);

  useEffect(() => stop, [stop]);

  return { supported, listening, error, start, stop, toggle };
}
