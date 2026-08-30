import {
  Bot,
  CheckCircle2,
  CornerDownRight,
  LoaderCircle,
  Square,
  UserRound,
  Volume2,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import type {
  InterviewQuestion,
  InterviewTurn,
} from "@/features/interview/types";
import type { QuestionSpeechStatus } from "@/features/tts/types";

interface QuestionSpeechControl {
  status: QuestionSpeechStatus;
  error: string | null;
  disabled: boolean;
  onToggle: () => void;
}

interface InterviewConversationProps {
  turns: InterviewTurn[];
  currentQuestion: InterviewQuestion | null;
  isSubmitting: boolean;
  speech: QuestionSpeechControl;
}

function QuestionBubble({
  number,
  content,
  isFollowUp,
  speech,
}: {
  number: string;
  content: string;
  isFollowUp: boolean;
  speech?: QuestionSpeechControl;
}) {
  return (
    <div className="flex items-start gap-3">
      <div className="flex size-8 shrink-0 items-center justify-center rounded-full bg-neutral-900 text-white">
        <Bot className="size-4" />
      </div>
      <div className="max-w-2xl">
        <div className="mb-2 flex min-h-7 flex-wrap items-center gap-2 text-xs font-medium text-neutral-500">
          <div className="flex flex-1 items-center gap-2">
            <span>问题 {number}</span>
            {isFollowUp && (
              <span className="flex items-center gap-1 rounded-full bg-amber-50 px-2 py-0.5 text-amber-700">
                <CornerDownRight className="size-3" />
                动态追问
              </span>
            )}
          </div>

          {speech && (
            <Button
              type="button"
              variant="ghost"
              size="xs"
              disabled={speech.disabled}
              title={speech.error || undefined}
              onClick={speech.onToggle}
            >
              {speech.status === "loading" ? (
                <LoaderCircle className="animate-spin" />
              ) : speech.status === "playing" ? (
                <Square className="fill-current" />
              ) : (
                <Volume2 />
              )}
              {speech.status === "loading"
                ? "生成语音"
                : speech.status === "playing"
                  ? "停止播报"
                  : "播放题目"}
            </Button>
          )}
        </div>
        <div className="rounded-2xl rounded-tl-md border border-neutral-200 bg-white px-4 py-3 leading-7 shadow-xs">
          {content}
        </div>
      </div>
    </div>
  );
}

function TurnView({ turn }: { turn: InterviewTurn }) {
  return (
    <div className="space-y-5">
      <QuestionBubble
        number={turn.questionNumber}
        content={turn.question}
        isFollowUp={turn.isFollowUp}
      />

      <div className="flex justify-end gap-3">
        <div className="max-w-2xl rounded-2xl rounded-tr-md bg-neutral-900 px-4 py-3 leading-7 text-white">
          {turn.answer}
        </div>
        <div className="flex size-8 shrink-0 items-center justify-center rounded-full bg-neutral-100">
          <UserRound className="size-4" />
        </div>
      </div>

      <div className="ml-11 rounded-2xl border border-emerald-100 bg-emerald-50/70 p-4">
        <div className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-2 text-sm font-medium text-emerald-800">
            <CheckCircle2 className="size-4" />
            面试官反馈
          </div>
          <div className="text-sm font-semibold text-emerald-800">
            {turn.score} 分
          </div>
        </div>
        <p className="mt-2 text-sm leading-6 text-emerald-900/80">
          {turn.feedback || "本题已完成评分。"}
        </p>
      </div>
    </div>
  );
}

export function InterviewConversation({
  turns,
  currentQuestion,
  isSubmitting,
  speech,
}: InterviewConversationProps) {
  return (
    <div className="space-y-8">
      {turns.map((turn) => (
        <TurnView key={turn.sequence} turn={turn} />
      ))}

      {currentQuestion && (
        <QuestionBubble
          number={currentQuestion.number}
          content={currentQuestion.content}
          isFollowUp={currentQuestion.isFollowUp}
          speech={speech}
        />
      )}

      {isSubmitting && (
        <div className="ml-11 flex items-center gap-2 text-sm text-neutral-500">
          <span className="size-2 animate-pulse rounded-full bg-neutral-400" />
          AI 面试官正在评分并决定是否追问...
        </div>
      )}
    </div>
  );
}
