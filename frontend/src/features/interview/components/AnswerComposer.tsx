import { LoaderCircle, Mic, Send, Square } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import type { VoiceInputStatus } from "@/features/transcription/types";

interface AnswerComposerProps {
  value: string;
  disabled: boolean;
  isSubmitting: boolean;
  voiceStatus: VoiceInputStatus;
  voiceSupported: boolean;
  voiceError: string | null;
  onChange: (value: string) => void;
  onSubmit: () => void;
  onVoiceToggle: () => void;
}

export function AnswerComposer({
  value,
  disabled,
  isSubmitting,
  voiceStatus,
  voiceSupported,
  voiceError,
  onChange,
  onSubmit,
  onVoiceToggle,
}: AnswerComposerProps) {
  const voiceBusy =
    voiceStatus === "connecting" ||
    voiceStatus === "listening" ||
    voiceStatus === "stopping";

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    onSubmit();
  }

  return (
    <form
      className="rounded-2xl border border-neutral-200 bg-white p-2.5 shadow-lg shadow-neutral-950/5 transition-shadow focus-within:border-neutral-300 focus-within:shadow-xl"
      onSubmit={handleSubmit}
    >
      <Textarea
        value={value}
        disabled={disabled || isSubmitting || voiceBusy}
        placeholder="输入你的回答。建议结合具体情境、行动和结果进行说明..."
        className="max-h-40 min-h-20 resize-none border-0 bg-transparent px-2 py-2 shadow-none focus-visible:ring-0"
        onChange={(event) => onChange(event.target.value)}
        onKeyDown={(event) => {
          if (
            event.key === "Enter" &&
            (event.ctrlKey || event.metaKey) &&
            !disabled &&
            !isSubmitting &&
            !voiceBusy &&
            value.trim()
          ) {
            event.preventDefault();
            onSubmit();
          }
        }}
      />

      <div className="flex items-center justify-between gap-3 px-1 pt-2">
        <div className="flex min-w-0 items-center gap-2">
          <Button
            type="button"
            variant={voiceStatus === "listening" ? "destructive" : "outline"}
            size="sm"
            aria-pressed={voiceStatus === "listening"}
            disabled={
              !voiceSupported ||
              isSubmitting ||
              voiceStatus === "connecting" ||
              voiceStatus === "stopping" ||
              (disabled && voiceStatus !== "listening")
            }
            onClick={onVoiceToggle}
          >
            {voiceStatus === "connecting" || voiceStatus === "stopping" ? (
              <LoaderCircle className="animate-spin" />
            ) : voiceStatus === "listening" ? (
              <Square className="fill-current" />
            ) : (
              <Mic />
            )}
            {voiceStatus === "connecting"
              ? "连接中"
              : voiceStatus === "stopping"
                ? "转写中"
                : voiceStatus === "listening"
                  ? "停止录音"
                  : "语音输入"}
          </Button>

          <p
            className={
              voiceError
                ? "truncate text-xs text-red-600"
                : voiceStatus === "listening"
                  ? "truncate text-xs text-red-600"
                  : "hidden text-xs text-neutral-400 sm:block"
            }
          >
            {voiceError
              ? voiceError
              : voiceStatus === "listening"
                ? "正在听，请自然说出你的回答"
                : voiceSupported
                  ? "Ctrl / ⌘ + Enter 提交"
                  : "当前浏览器不支持语音输入"}
          </p>
        </div>
        <Button
          type="submit"
          disabled={disabled || isSubmitting || voiceBusy || !value.trim()}
        >
          {isSubmitting ? (
            <LoaderCircle className="animate-spin" />
          ) : (
            <Send />
          )}
          {isSubmitting ? "评分中" : "提交回答"}
        </Button>
      </div>
    </form>
  );
}
