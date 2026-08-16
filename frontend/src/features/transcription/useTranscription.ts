import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { BrowserSpeechProvider } from "./BrowserSpeechProvider";
import type { TranscriptionProvider, TranscriptionState } from "./types";

const initialState: TranscriptionState = {
  liveText: "",
  committedText: "",
  status: "idle",
};

export function useTranscription(
  createProvider: () => TranscriptionProvider = () =>
    new BrowserSpeechProvider(),
) {
  const providerRef = useRef<TranscriptionProvider | null>(null);
  const [state, setState] = useState(initialState);
  const [error, setError] = useState<string | null>(null);

  const start = useCallback(async () => {
    providerRef.current?.dispose();
    const provider = createProvider();
    providerRef.current = provider;
    setError(null);
    setState(initialState);
    await provider.start({
      onPartial: (liveText) => setState((old) => ({ ...old, liveText })),
      onFinal: (committedText) =>
        setState((old) => ({ ...old, committedText, liveText: "" })),
      onStateChange: (status) => setState((old) => ({ ...old, status })),
      onError: setError,
    });
  }, [createProvider]);

  const stop = useCallback(async () => {
    await providerRef.current?.stop();
  }, []);

  useEffect(() => () => providerRef.current?.dispose(), []);
  return useMemo(
    () => ({ state, error, start, stop, isSupported: new BrowserSpeechProvider().isSupported() }),
    [error, start, state, stop],
  );
}
