import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { getErrorMessage } from "@/lib/errors";
import { BrowserSpeechPlayer } from "./browser-speech-player";
import { ttsApi } from "./tts-api";
import type { QuestionSpeechStatus } from "./types";

interface QuestionToSpeak {
  id: string;
  text: string;
}

interface UseQuestionSpeechOptions {
  sessionId: string;
  question: QuestionToSpeak | null;
  enabled: boolean;
  blocked: boolean;
}

function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === "AbortError";
}

function idempotencyKey(sessionId: string, questionId: string) {
  const normalizedQuestionId = questionId.replace(/[^a-zA-Z0-9._-]/g, "-");
  return `tts-${sessionId}-${normalizedQuestionId}`.slice(0, 128);
}

export function useQuestionSpeech({
  sessionId,
  question,
  enabled,
  blocked,
}: UseQuestionSpeechOptions) {
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const controllerRef = useRef<AbortController | null>(null);
  const browserPlayerRef = useRef(new BrowserSpeechPlayer());
  const cachedUrlsRef = useRef(new Map<string, string>());
  const autoPlayedRef = useRef(new Set<string>());
  const playbackTokenRef = useRef(0);
  const [status, setStatus] = useState<QuestionSpeechStatus>("idle");
  const [error, setError] = useState<string | null>(null);

  const stop = useCallback(() => {
    playbackTokenRef.current += 1;
    controllerRef.current?.abort();
    controllerRef.current = null;
    browserPlayerRef.current.stop();

    if (audioRef.current) {
      audioRef.current.pause();
      audioRef.current.currentTime = 0;
      audioRef.current = null;
    }

    setStatus("idle");
  }, []);

  const play = useCallback(
    async (target: QuestionToSpeak, userInitiated = false) => {
      if (!enabled || blocked || !target.text.trim()) return;

      stop();
      const token = playbackTokenRef.current;
      const controller = new AbortController();
      controllerRef.current = controller;
      setError(null);
      setStatus("loading");

      try {
        let objectUrl = cachedUrlsRef.current.get(target.id);
        if (!objectUrl) {
          const task = await ttsApi.synthesize(
            target.text,
            idempotencyKey(sessionId, target.id),
            controller.signal,
          );

          if (!task.success || !task.audioPath) {
            throw new Error(task.message || "远程语音合成失败");
          }

          const blob = await ttsApi.audioBlob(task.audioPath, controller.signal);
          objectUrl = URL.createObjectURL(blob);
          cachedUrlsRef.current.set(target.id, objectUrl);
        }

        if (controller.signal.aborted || token !== playbackTokenRef.current) return;

        const audio = new Audio(objectUrl);
        audioRef.current = audio;
        await audio.play();
        if (token === playbackTokenRef.current) setStatus("playing");

        await new Promise<void>((resolve, reject) => {
          audio.onended = () => resolve();
          audio.onerror = () => reject(new Error("合成音频播放失败"));
          controller.signal.addEventListener(
            "abort",
            () => resolve(),
            { once: true },
          );
        });
      } catch (remoteError) {
        if (isAbortError(remoteError) || controller.signal.aborted) return;

        try {
          setStatus("playing");
          await browserPlayerRef.current.speak(target.text, controller.signal);
        } catch (browserError) {
          if (isAbortError(browserError) || controller.signal.aborted) return;

          const message = getErrorMessage(
            browserError,
            getErrorMessage(remoteError, "题目语音播报失败"),
          );
          setError(message);
          setStatus("error");
          if (userInitiated) {
            toast.error("无法播放题目", { description: message });
          }
          return;
        }
      } finally {
        if (token === playbackTokenRef.current) {
          controllerRef.current = null;
          audioRef.current = null;
          setStatus((current) => (current === "error" ? current : "idle"));
        }
      }
    },
    [blocked, enabled, sessionId, stop],
  );

  const toggle = useCallback(() => {
    if (status === "loading" || status === "playing") {
      stop();
      return;
    }

    if (question) void play(question, true);
  }, [play, question, status, stop]);

  useEffect(() => {
    if (!enabled || blocked || !question) return;
    if (autoPlayedRef.current.has(question.id)) return;

    autoPlayedRef.current.add(question.id);
    void play(question);
  }, [blocked, enabled, play, question]);

  useEffect(
    () => () => {
      controllerRef.current?.abort();
      browserPlayerRef.current.stop();
      audioRef.current?.pause();
      cachedUrlsRef.current.forEach((url) => URL.revokeObjectURL(url));
      cachedUrlsRef.current.clear();
    },
    [],
  );

  return {
    status,
    error,
    stop,
    toggle,
    isBusy: status === "loading" || status === "playing",
  };
}
