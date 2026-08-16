import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { XunfeiPreferredProvider } from "./XunfeiPreferredProvider";
import type { TranscriptionProvider, TranscriptionState } from "./types";

const initialState: TranscriptionState = {
  liveText: "",
  committedText: "",
  status: "idle",
};

export function useTranscription(
  createProvider: () => TranscriptionProvider = () =>
    new XunfeiPreferredProvider(),
) {
  const providerRef = useRef<TranscriptionProvider | null>(null);
  const revisionRef = useRef(0);
  const [state, setState] = useState(initialState);
  const [error, setError] = useState<string | null>(null);

  const start = useCallback(async () => {
    providerRef.current?.dispose();
    const provider = createProvider();
    providerRef.current = provider;
    setError(null);
    setState(initialState);
    revisionRef.current = 0;
    await provider.start({
      onPartial: (liveText) => setState((old) => ({ ...old, liveText })),
      onFinal: (committedText) =>
        setState((old) => ({ ...old, committedText, liveText: "" })),
      onStateChange: (status) => setState((old) => ({ ...old, status })),
      onError: setError,
      onSnapshot: (snapshot) => {
        if (snapshot.revision > 0 && snapshot.revision <= revisionRef.current) {
          return;
        }
        revisionRef.current = snapshot.revision;
        setState({
          committedText: snapshot.committedText,
          liveText: snapshot.liveText,
          status:
            snapshot.status === "error"
              ? "error"
              : snapshot.status === "final"
                ? "idle"
                : "listening",
        });
      },
    });
  }, [createProvider]);

  const stop = useCallback(async () => {
    await providerRef.current?.stop();
    setState((old) => ({ ...old, status: "idle" }));
  }, []);

  useEffect(() => () => providerRef.current?.dispose(), []);
  return useMemo(
    () => ({ state, error, start, stop, isSupported: new XunfeiPreferredProvider().isSupported() }),
    [error, start, state, stop],
  );
}
