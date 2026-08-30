import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowLeft,
  CheckCircle2,
  FileText,
  MessageSquareText,
  Square,
} from "lucide-react";
import { Link, useParams } from "react-router-dom";
import { toast } from "sonner";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import {
  ErrorState,
  InlineError,
  PageLoading,
} from "@/components/feedback/page-state";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { AnswerComposer } from "@/features/interview/components/AnswerComposer";
import { InterviewConversation } from "@/features/interview/components/InterviewConversation";
import { ResumePreviewPanel } from "@/features/interview/components/ResumePreviewPanel";
import { interviewApi } from "@/features/interview/interview-api";
import { useVoiceTranscription } from "@/features/transcription/useVoiceTranscription";
import { useQuestionSpeech } from "@/features/tts/useQuestionSpeech";
import { getErrorMessage } from "@/lib/errors";

interface PendingAnswerAttempt {
  questionNumber: string;
  answerContent: string;
  idempotencyKey: string;
}

export function InterviewRoomPage() {
  const { sessionId } = useParams();
  const queryClient = useQueryClient();
  const conversationEndRef = useRef<HTMLDivElement | null>(null);
  const pendingAttemptRef = useRef<PendingAnswerAttempt | null>(null);
  const [answer, setAnswer] = useState("");
  const [resumePreviewOpen, setResumePreviewOpen] = useState(false);

  const stateQuery = useQuery({
    queryKey: ["interview-state", sessionId],
    enabled: Boolean(sessionId),
    queryFn: () => interviewApi.state(sessionId as string),
    retry: false,
  });

  const answerMutation = useMutation({
    mutationFn: (attempt: PendingAnswerAttempt) =>
      interviewApi.answer(
        sessionId as string,
        {
          questionNumber: attempt.questionNumber,
          answerContent: attempt.answerContent,
        },
        attempt.idempotencyKey,
      ),
    onSuccess: async () => {
      pendingAttemptRef.current = null;
      setAnswer("");
      await Promise.all([
        stateQuery.refetch(),
        queryClient.invalidateQueries({ queryKey: ["interviews"] }),
      ]);
      toast.success("回答已提交", {
        description: "面试官反馈和下一道问题已更新。",
      });
    },
    onError: (error) => {
      toast.error("回答提交失败", {
        description: getErrorMessage(error, "请保留回答内容并重新提交。"),
      });
    },
  });

  const finishMutation = useMutation({
    mutationFn: () => interviewApi.finish(sessionId as string),
    onSuccess: async () => {
      await Promise.all([
        stateQuery.refetch(),
        queryClient.invalidateQueries({ queryKey: ["interviews"] }),
      ]);
      toast.success("面试已结束", {
        description: "报告已经生成，可以开始复盘。",
      });
    },
    onError: (error) => {
      toast.error("结束面试失败", {
        description: getErrorMessage(error, "请稍后再试。"),
      });
    },
  });

  const state = stateQuery.data;
  const isActive = state?.status === "READY" || state?.status === "IN_PROGRESS";
  const interactionDisabled =
    !isActive || !state?.currentQuestion || finishMutation.isPending;
  const voiceInput = useVoiceTranscription({
    value: answer,
    disabled: interactionDisabled || answerMutation.isPending,
    onChange: setAnswer,
  });
  const questionSpeech = useQuestionSpeech({
    sessionId: sessionId ?? "",
    question: state?.currentQuestion
      ? {
          id: state.currentQuestion.number,
          text: state.currentQuestion.content,
        }
      : null,
    enabled: Boolean(isActive && state?.currentQuestion),
    blocked: voiceInput.isBusy,
  });

  useEffect(() => {
    conversationEndRef.current?.scrollIntoView({
      behavior: "smooth",
      block: "end",
    });
  }, [state?.currentQuestion?.number, state?.recentTurns.length]);

  function handleAnswerChange(value: string) {
    setAnswer(value);

    const pending = pendingAttemptRef.current;
    if (pending && pending.answerContent !== value.trim()) {
      pendingAttemptRef.current = null;
    }

    if (answerMutation.isError) {
      answerMutation.reset();
    }
  }

  function handleSubmitAnswer() {
    const currentQuestion = state?.currentQuestion;
    const answerContent = answer.trim();

    if (!currentQuestion || !answerContent || answerMutation.isPending) {
      return;
    }

    const previousAttempt = pendingAttemptRef.current;
    const attempt =
      previousAttempt?.questionNumber === currentQuestion.number &&
      previousAttempt.answerContent === answerContent
        ? previousAttempt
        : {
            questionNumber: currentQuestion.number,
            answerContent,
            idempotencyKey: crypto.randomUUID(),
          };

    pendingAttemptRef.current = attempt;
    answerMutation.mutate(attempt);
  }

  function handleVoiceToggle() {
    if (!voiceInput.isBusy) questionSpeech.stop();
    voiceInput.toggle();
  }

  if (stateQuery.isLoading) {
    return (
      <PageLoading
        className="h-full"
        title="正在进入面试房间"
        description="正在恢复题目与作答进度。"
      />
    );
  }

  if (stateQuery.error || !state) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-12 sm:px-6">
        <ErrorState
          title="无法进入面试房间"
          message={getErrorMessage(stateQuery.error, "无法加载面试状态")}
          isRetrying={stateQuery.isFetching}
          onRetry={() => void stateQuery.refetch()}
          secondaryAction={
            <Button asChild variant="ghost">
              <Link to="/interviews">返回面试列表</Link>
            </Button>
          }
        />
      </div>
    );
  }

  if (state.status === "COMPLETED") {
    return (
      <div className="flex h-full items-center justify-center overflow-y-auto bg-neutral-50/50 px-4 py-10 sm:px-6">
        <Card className="w-full max-w-xl text-center shadow-sm">
          <CardHeader className="w-full items-center px-8 pt-4">
            <div className="mb-3 flex size-14 items-center justify-center rounded-2xl bg-emerald-50 text-emerald-700">
              <CheckCircle2 className="size-7" strokeWidth={1.8} />
            </div>
            <CardTitle className="text-2xl font-semibold tracking-tight">
              本次面试已完成
            </CardTitle>
            <CardDescription className="mt-1 max-w-sm leading-6">
              你的作答记录和面试官反馈已经保存，可以前往报告页面进行完整复盘。
            </CardDescription>
          </CardHeader>
          <CardContent className="w-full items-center px-8 pb-4">
            <div className="flex w-full max-w-sm items-center justify-between rounded-2xl bg-neutral-50 px-5 py-4 text-left ring-1 ring-neutral-200/70">
              <div>
                <p className="text-xs text-neutral-500">回答得分</p>
                <p className="mt-1 text-sm font-medium">本次面试表现</p>
              </div>
              <div className="text-3xl font-semibold tracking-tight">
                {state.progress.totalScore}
                <span className="ml-1 text-sm font-normal text-neutral-400">分</span>
              </div>
            </div>
            <div className="flex flex-wrap justify-center gap-3">
              <Button asChild>
                <Link
                  to={`/interviews/${encodeURIComponent(state.sessionId)}/report`}
                >
                  查看面试报告
                </Link>
              </Button>
              <Button asChild variant="outline">
                <Link to="/interviews">返回面试列表</Link>
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="flex h-full min-h-0 bg-neutral-50/60">
      <div className="flex min-w-0 flex-1 flex-col">
        <header className="shrink-0 border-b border-neutral-200 bg-white/90 backdrop-blur">
        <div className="mx-auto flex min-h-18 max-w-5xl flex-wrap items-center justify-between gap-3 px-4 py-3 sm:px-6">
          <div className="flex items-center gap-3">
            <Button asChild variant="ghost" size="icon">
              <Link to="/interviews" aria-label="返回面试列表">
                <ArrowLeft />
              </Link>
            </Button>
            <div>
              <p className="text-xs font-medium text-neutral-500">面试房间</p>
              <h1 className="mt-1 text-xl font-semibold">
                {state.interviewType || "AI 模拟面试"}
              </h1>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <div className="hidden items-center gap-4 text-sm text-neutral-500 sm:flex">
              <button
                type="button"
                className="flex items-center gap-1.5 rounded-lg px-2 py-1.5 transition-colors hover:bg-neutral-100 hover:text-neutral-950"
                aria-pressed={resumePreviewOpen}
                onClick={() => setResumePreviewOpen(true)}
              >
                <FileText className="size-4" />
                简历 {state.resumeScore} 分
              </button>
              <span className="flex items-center gap-1.5">
                <MessageSquareText className="size-4" />
                {state.progress.mainCompleted}/{state.progress.mainTotal}
              </span>
            </div>

            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="sm:hidden"
              aria-label="预览简历 PDF"
              aria-pressed={resumePreviewOpen}
              onClick={() => setResumePreviewOpen(true)}
            >
              <FileText />
            </Button>

            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button
                  variant="outline"
                  disabled={
                    !isActive || answerMutation.isPending || voiceInput.isBusy
                  }
                >
                  <Square />
                  结束面试
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>确认结束本次面试？</AlertDialogTitle>
                  <AlertDialogDescription>
                    结束后不能继续回答，系统会根据当前已完成题目生成面试报告。
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>继续面试</AlertDialogCancel>
                  <AlertDialogAction
                    disabled={finishMutation.isPending}
                    onClick={() => finishMutation.mutate()}
                  >
                    {finishMutation.isPending ? "结束中..." : "确认结束"}
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>
        </div>
        </header>

        <div className="min-h-0 flex-1 overflow-y-auto">
          <div className="mx-auto max-w-3xl px-4 py-8 sm:px-6">
          <InterviewConversation
            turns={state.recentTurns}
            currentQuestion={state.currentQuestion}
            isSubmitting={answerMutation.isPending}
            speech={{
              status: questionSpeech.status,
              error: questionSpeech.error,
              disabled: voiceInput.isBusy,
              onToggle: questionSpeech.toggle,
            }}
          />

          <div ref={conversationEndRef} />

          {answerMutation.error && (
            <div className="mt-6">
              <InlineError
                message={`${getErrorMessage(answerMutation.error, "提交回答失败，请重试")} 保持回答内容不变时会安全重试。`}
              />
            </div>
          )}

          {finishMutation.error && (
            <div className="mt-6">
              <InlineError
                message={getErrorMessage(
                  finishMutation.error,
                  "结束面试失败，请重试",
                )}
              />
            </div>
          )}

          <div className="h-4" />
          </div>
        </div>

        <div className="shrink-0 border-t border-neutral-200 bg-white px-4 py-3 shadow-[0_-10px_30px_rgba(0,0,0,0.035)] sm:px-6 sm:py-4">
          <div className="mx-auto max-w-3xl">
          <AnswerComposer
            value={answer}
            disabled={interactionDisabled}
            isSubmitting={answerMutation.isPending}
            voiceStatus={voiceInput.status}
            voiceSupported={voiceInput.isSupported}
            voiceError={voiceInput.error}
            onChange={handleAnswerChange}
            onSubmit={handleSubmitAnswer}
            onVoiceToggle={handleVoiceToggle}
          />
          </div>
        </div>
      </div>

      {resumePreviewOpen && (
        <ResumePreviewPanel
          sessionId={state.sessionId}
          onClose={() => setResumePreviewOpen(false)}
        />
      )}
    </div>
  );
}
