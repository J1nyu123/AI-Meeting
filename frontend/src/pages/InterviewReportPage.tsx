import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import {
  ArrowLeft,
  Award,
  CheckCircle2,
  FileText,
  Lightbulb,
  ListChecks,
  MessageSquareText,
  Target,
} from "lucide-react";
import { Link, useParams } from "react-router-dom";
import {
  ErrorState,
  PageLoading,
} from "@/components/feedback/page-state";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { interviewApi } from "@/features/interview/interview-api";
import { ResumePreviewPanel } from "@/features/interview/components/ResumePreviewPanel";
import { ReportQaReview } from "@/features/report/ReportQaReview";
import { ReportRadarChart } from "@/features/report/ReportRadarChart";
import { getErrorMessage } from "@/lib/errors";

function ScoreCard({
  label,
  value,
  description,
  icon: Icon,
}: {
  label: string;
  value: number;
  description: string;
  icon: typeof Award;
}) {
  return (
    <Card className="shadow-none">
      <CardContent className="pt-1">
        <div className="flex items-center justify-between">
          <p className="text-sm text-neutral-500">{label}</p>
          <Icon className="size-5 text-neutral-400" />
        </div>
        <div className="mt-4 flex items-end gap-1">
          <span className="text-4xl font-semibold tracking-tight">{value}</span>
          <span className="pb-1 text-sm text-neutral-400">/ 100</span>
        </div>
        <p className="mt-2 text-xs text-neutral-500">{description}</p>
      </CardContent>
    </Card>
  );
}

function AdviceList({
  title,
  items,
  icon: Icon,
  tone,
}: {
  title: string;
  items: string[];
  icon: typeof CheckCircle2;
  tone: "positive" | "warning" | "neutral";
}) {
  const toneClasses = {
    positive: "bg-emerald-50 text-emerald-700",
    warning: "bg-amber-50 text-amber-700",
    neutral: "bg-neutral-100 text-neutral-700",
  } as const;

  return (
    <Card className="shadow-none">
      <CardHeader>
        <div
          className={`mb-2 flex size-9 items-center justify-center rounded-xl ${toneClasses[tone]}`}
        >
          <Icon className="size-4" />
        </div>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent>
        {items.length > 0 ? (
          <ul className="space-y-3">
            {items.map((item) => (
              <li key={item} className="flex items-start gap-2 text-sm leading-6">
                <span className="mt-2 size-1.5 shrink-0 rounded-full bg-neutral-400" />
                {item}
              </li>
            ))}
          </ul>
        ) : (
          <p className="text-sm text-neutral-500">暂无内容</p>
        )}
      </CardContent>
    </Card>
  );
}

export function InterviewReportPage() {
  const { sessionId } = useParams();
  const [resumePreviewOpen, setResumePreviewOpen] = useState(false);
  const reportQuery = useQuery({
    queryKey: ["interview-report", sessionId],
    enabled: Boolean(sessionId),
    queryFn: () => interviewApi.ensureReport(sessionId as string),
    retry: false,
  });

  if (reportQuery.isLoading) {
    return (
      <PageLoading
        fullPage
        title="正在生成面试报告"
        description="正在汇总得分、反馈和改进建议。"
      />
    );
  }

  if (reportQuery.error || !reportQuery.data) {
    const message = getErrorMessage(
      reportQuery.error,
      "面试报告暂时无法加载",
    );

    return (
      <div className="mx-auto max-w-3xl px-4 py-12 sm:px-6">
        <ErrorState
          title="无法加载面试报告"
          message={message}
          isRetrying={reportQuery.isFetching}
          onRetry={() => void reportQuery.refetch()}
          secondaryAction={
            <Button asChild variant="ghost">
              <Link to="/interviews">返回列表</Link>
            </Button>
          }
        />
      </div>
    );
  }

  const report = reportQuery.data;

  return (
    <div className="flex h-full min-h-0 bg-neutral-50/60">
      <div className="min-w-0 flex-1 overflow-y-auto">
        <div className="mx-auto max-w-6xl px-4 py-10 sm:px-6">
        <Button asChild variant="ghost" className="mb-6 -ml-2">
          <Link to="/interviews">
            <ArrowLeft data-icon="inline-start" />
            返回面试列表
          </Link>
        </Button>

        <div className="flex flex-wrap items-end justify-between gap-4">
          <div>
            <p className="text-sm font-medium text-neutral-500">面试报告</p>
            <h1 className="mt-2 text-3xl font-semibold tracking-tight sm:text-4xl">
              本次模拟面试复盘
            </h1>
            <p className="mt-3 max-w-2xl text-neutral-500">
              根据简历质量、面试回答和逐题反馈生成。
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => setResumePreviewOpen(true)}
            >
              <FileText />
              预览简历
            </Button>
            <div className="rounded-full bg-neutral-900 px-4 py-2 text-sm font-medium text-white">
              综合得分 {report.compositeScore}
            </div>
          </div>
        </div>

        <section className="mt-8 grid gap-4 md:grid-cols-3">
          <ScoreCard
            label="简历得分"
            value={report.resumeScore}
            description="简历内容与表达质量"
            icon={FileText}
          />
          <ScoreCard
            label="面试得分"
            value={report.interviewScore}
            description="主问题回答平均得分"
            icon={MessageSquareText}
          />
          <ScoreCard
            label="综合得分"
            value={report.compositeScore}
            description="简历 20% + 面试 80%"
            icon={Award}
          />
        </section>

        <section className="mt-6 grid items-start gap-6 lg:grid-cols-[1.05fr_0.95fr]">
          <Card className="shadow-none">
            <CardHeader>
              <CardTitle>能力雷达</CardTitle>
              <CardDescription>各项指标均按 100 分计算</CardDescription>
            </CardHeader>
            <CardContent>
              <ReportRadarChart metrics={report.radarMetrics ?? []} />
            </CardContent>
          </Card>

          <Card className="shadow-none">
            <CardHeader>
              <div className="mb-2 flex size-10 items-center justify-center rounded-xl bg-neutral-100">
                <Target className="size-5" />
              </div>
              <CardTitle>面试官总评</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="whitespace-pre-wrap text-base leading-8 text-neutral-700">
                {report.overallComment || "本次面试已完成。"}
              </p>
            </CardContent>
          </Card>
        </section>

        <section className="mt-6 grid gap-4 lg:grid-cols-3">
          <AdviceList
            title="表现亮点"
            items={report.highlights ?? []}
            icon={CheckCircle2}
            tone="positive"
          />
          <AdviceList
            title="改进建议"
            items={report.improvementTips ?? []}
            icon={Lightbulb}
            tone="warning"
          />
          <AdviceList
            title="下一步行动"
            items={report.nextActions ?? []}
            icon={ListChecks}
            tone="neutral"
          />
        </section>

        <Card className="mt-6 shadow-none">
          <CardHeader>
            <CardTitle>逐题复盘</CardTitle>
            <CardDescription>
              展开每一道题，重新检查回答内容和面试官反馈。
            </CardDescription>
          </CardHeader>
          <CardContent>
            <ReportQaReview turns={report.turns ?? []} />
          </CardContent>
        </Card>
        </div>
      </div>

      {resumePreviewOpen && sessionId && (
        <ResumePreviewPanel
          sessionId={sessionId}
          onClose={() => setResumePreviewOpen(false)}
        />
      )}
    </div>
  );
}
