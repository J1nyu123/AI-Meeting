import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { getErrorMessage } from "@/lib/errors";
import { RemoteAsrSession } from "./remote-asr-session";
import type { VoiceInputStatus } from "./types";

interface UseVoiceTranscriptionOptions {
  value: string;
  disabled: boolean;
  onChange: (value: string) => void;
}

function mergeWithExistingAnswer(base: string, transcript: string) {
  const normalizedTranscript = transcript.trim();
  if (!normalizedTranscript) return base;
  if (!base.trim()) return normalizedTranscript;

  const separator = /[\s\n]$/.test(base) ? "" : "\n";
  return `${base}${separator}${normalizedTranscript}`;
}

function microphoneErrorMessage(error: unknown) {
  if (error instanceof DOMException) {
    if (error.name === "NotAllowedError") {
      return "麦克风权限被拒绝，请在浏览器设置中允许后重试";
    }
    if (error.name === "NotFoundError") {
      return "没有检测到可用的麦克风";
    }
    if (error.name === "NotReadableError") {
      return "麦克风正被其他应用占用";
    }
  }

  return getErrorMessage(error, "语音输入启动失败，请改用文字输入");
}

export function useVoiceTranscription({
  value,
  disabled,
  onChange,
}: UseVoiceTranscriptionOptions) {
  const sessionRef = useRef<RemoteAsrSession | null>(null);
  const valueRef = useRef(value);
  const onChangeRef = useRef(onChange);
  const baseAnswerRef = useRef("");
  const [status, setStatus] = useState<VoiceInputStatus>("idle");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    valueRef.current = value;
    onChangeRef.current = onChange;
  }, [onChange, value]);

  const start = useCallback(async () => {
    if (disabled || status === "connecting" || status === "listening") return;

    const session = new RemoteAsrSession();
    sessionRef.current = session;
    baseAnswerRef.current = valueRef.current;
    setError(null);

    try {
      await session.start({
        onStatusChange: setStatus,
        onTranscript: (snapshot) => {
          onChangeRef.current(
            mergeWithExistingAnswer(
              baseAnswerRef.current,
              snapshot.displayText,
            ),
          );
        },
        onError: (message) => {
          setError(message);
          toast.error("语音识别中断", { description: message });
        },
      });
    } catch (startError) {
      const message = microphoneErrorMessage(startError);
      setStatus("error");
      setError(message);
      toast.error("无法开始语音输入", { description: message });
      await session.dispose();
    }
  }, [disabled, status]);

  const stop = useCallback(async () => {
    try {
      await sessionRef.current?.stop();
    } catch (stopError) {
      const message = getErrorMessage(stopError, "停止录音失败");
      setStatus("error");
      setError(message);
      toast.error("语音输入异常", { description: message });
    }
  }, []);

  const toggle = useCallback(() => {
    if (status === "listening") {
      void stop();
    } else if (status === "idle" || status === "error") {
      void start();
    }
  }, [start, status, stop]);

  useEffect(
    () => () => {
      void sessionRef.current?.dispose();
    },
    [],
  );

  return {
    status,
    error,
    toggle,
    isSupported: RemoteAsrSession.isSupported(),
    isBusy:
      status === "connecting" ||
      status === "listening" ||
      status === "stopping",
  };
}
