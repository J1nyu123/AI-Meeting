import { ChevronDown, CornerDownRight, MessageSquareOff } from "lucide-react";
import { ContentEmptyState } from "@/components/feedback/page-state";
import type { InterviewTurn } from "@/features/interview/types";

export function ReportQaReview({ turns }: { turns: InterviewTurn[] }) {
  if (turns.length === 0) {
    return (
      <ContentEmptyState
        icon={MessageSquareOff}
        title="暂无问答记录"
        description="本次面试没有已完成的问答。"
      />
    );
  }

  return (
    <div className="divide-y divide-neutral-200">
      {turns.map((turn) => (
        <details key={turn.sequence} className="group py-1">
          <summary className="flex cursor-pointer list-none items-start gap-4 rounded-xl px-3 py-4 hover:bg-neutral-50">
            <span className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-neutral-100 text-sm font-semibold">
              {turn.sequence}
            </span>
            <span className="min-w-0 flex-1">
              <span className="flex flex-wrap items-center gap-2">
                <span className="font-medium">{turn.question}</span>
                {turn.isFollowUp && (
                  <span className="inline-flex items-center gap-1 rounded-full bg-amber-50 px-2 py-0.5 text-xs text-amber-700">
                    <CornerDownRight className="size-3" />
                    追问
                  </span>
                )}
              </span>
              <span className="mt-1 block text-sm text-neutral-500">
                问题 {turn.questionNumber} · {turn.score} 分
              </span>
            </span>
            <ChevronDown className="mt-1 size-4 shrink-0 text-neutral-400 transition-transform group-open:rotate-180" />
          </summary>

          <div className="space-y-4 px-3 pb-5 pl-15">
            <div>
              <p className="text-xs font-medium text-neutral-400">你的回答</p>
              <p className="mt-1 whitespace-pre-wrap text-sm leading-6 text-neutral-700">
                {turn.answer}
              </p>
            </div>
            <div className="rounded-xl bg-emerald-50 p-4">
              <p className="text-xs font-medium text-emerald-700">面试官反馈</p>
              <p className="mt-1 text-sm leading-6 text-emerald-900/80">
                {turn.feedback || "本题已完成评分。"}
              </p>
            </div>
          </div>
        </details>
      ))}
    </div>
  );
}
