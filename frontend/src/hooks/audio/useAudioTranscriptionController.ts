import { useCallback, useMemo } from "react";
import { mergeTranscript } from "@/features/transcription/dedupe";
import { useTranscription } from "@/features/transcription/useTranscription";
import type { UserRespDTO } from "@/types/auth";

export function useAudioTranscriptionController(
  currentUser: UserRespDTO | null,
) {
  const { state, error, start, stop, isSupported } = useTranscription();

  const startRecording = useCallback(async () => {
    if (!currentUser) {
      throw new Error("User is not logged in");
    }
    await start();
  }, [currentUser, start]);

  const stopRecording = useCallback(() => {
    void stop();
  }, [stop]);

  return useMemo(
    () => ({
      isRecording: state.status === "listening" || state.status === "restarting",
      currentSentence: state.liveText,
      historySentences: state.committedText
        ? state.committedText
            .split(/\n\n+/)
            .map((sentence) => sentence.trim())
            .filter(Boolean)
        : [],
      transcription: mergeTranscript(state.committedText, state.liveText),
      error,
      isSpeechSupported: isSupported,
      startRecording,
      stopRecording,
    }),
    [error, isSupported, startRecording, state, stopRecording],
  );
}
