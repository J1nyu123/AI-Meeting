import { useEffect, useRef, useState, type FormEvent } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, FileText, LoaderCircle, Upload } from "lucide-react";
import { Link, useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { InlineError } from "@/components/feedback/page-state";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { waitForAnalysisJob } from "@/features/interview/analysis-stream";
import { interviewApi } from "@/features/interview/interview-api";
import type { AnalysisJob } from "@/features/interview/types";
import { getErrorMessage } from "@/lib/errors";

const MAX_RESUME_SIZE = 10 * 1024 * 1024;

type UploadStage = "idle" | "creating" | "uploading" | "analyzing";

const stageCopy: Record<UploadStage, string> = {
  idle: "等待上传",
  creating: "正在创建面试会话",
  uploading: "正在上传 PDF 简历",
  analyzing: "AI 正在分析简历并生成问题",
};

function validateResume(file: File) {
  if (!file.name.toLowerCase().endsWith(".pdf")) {
    return "请选择 PDF 格式的简历";
  }

  if (file.type && file.type !== "application/pdf") {
    return "文件类型不是有效的 PDF";
  }

  if (file.size > MAX_RESUME_SIZE) {
    return "简历大小不能超过 10 MiB";
  }

  if (file.size === 0) {
    return "简历文件不能为空";
  }

  return null;
}

export function NewInterviewPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const controllerRef = useRef<AbortController | null>(null);
  const [file, setFile] = useState<File | null>(null);
  const [fileError, setFileError] = useState("");
  const [stage, setStage] = useState<UploadStage>("idle");
  const [job, setJob] = useState<AnalysisJob | null>(null);

  useEffect(
    () => () => {
      controllerRef.current?.abort();
    },
    [],
  );

  const createInterviewMutation = useMutation({
    mutationFn: async (resume: File) => {
      controllerRef.current?.abort();
      const controller = new AbortController();
      controllerRef.current = controller;

      setStage("creating");
      const session = await interviewApi.create();

      setStage("uploading");
      const createdJob = await interviewApi.uploadResume(
        session.sessionId,
        resume,
      );
      setJob(createdJob);

      setStage("analyzing");
      await waitForAnalysisJob(createdJob.jobId, setJob, controller.signal);

      return session.sessionId;
    },
    onSuccess: async (sessionId) => {
      await queryClient.invalidateQueries({ queryKey: ["interviews"] });
      toast.success("简历分析完成", {
        description: "面试问题已经准备好，可以开始作答了。",
      });
      navigate(`/interviews/${encodeURIComponent(sessionId)}`, {
        replace: true,
      });
    },
    onError: (error) => {
      if (error instanceof DOMException && error.name === "AbortError") return;

      toast.error("创建面试失败", {
        description: getErrorMessage(error, "请检查网络后再试一次。"),
      });
    },
  });

  function handleFileChange(event: React.ChangeEvent<HTMLInputElement>) {
    const selectedFile = event.target.files?.[0] ?? null;
    createInterviewMutation.reset();
    setJob(null);
    setStage("idle");

    if (!selectedFile) {
      setFile(null);
      setFileError("");
      return;
    }

    const validationError = validateResume(selectedFile);
    setFileError(validationError ?? "");
    setFile(validationError ? null : selectedFile);
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (!file) {
      setFileError("请先选择一份 PDF 简历");
      return;
    }

    createInterviewMutation.mutate(file);
  }

  const progress =
    stage === "creating"
      ? 5
      : stage === "uploading"
        ? 15
        : stage === "analyzing"
          ? Math.max(20, job?.progress ?? 20)
          : 0;

  const mutationError = createInterviewMutation.error;
  const errorMessage =
    mutationError instanceof DOMException && mutationError.name === "AbortError"
      ? null
      : mutationError instanceof Error
        ? mutationError.message
        : mutationError
          ? "创建面试失败，请稍后重试"
          : null;

  return (
    <div className="mx-auto max-w-3xl px-4 py-12 sm:px-6">
      <Button asChild variant="ghost" className="mb-6 -ml-2">
        <Link to="/interviews">
          <ArrowLeft data-icon="inline-start" />
          返回面试列表
        </Link>
      </Button>

      <Card className="shadow-none">
        <CardHeader>
          <div className="mb-3 flex size-11 items-center justify-center rounded-xl bg-neutral-100">
            <FileText className="size-5" />
          </div>
          <CardTitle className="text-2xl">创建一场模拟面试</CardTitle>
          <CardDescription>
            上传文本型 PDF 简历，系统会分析经历并生成个性化面试问题。
          </CardDescription>
        </CardHeader>

        <CardContent>
          <form className="space-y-6" onSubmit={handleSubmit}>
            <div className="space-y-2">
              <Label htmlFor="resume">PDF 简历</Label>
              <Input
                id="resume"
                type="file"
                accept=".pdf,application/pdf"
                disabled={createInterviewMutation.isPending}
                onChange={handleFileChange}
              />
              <p className="text-xs text-neutral-500">
                仅支持可复制文本的 PDF，最大 10 MiB，暂不支持扫描件 OCR。
              </p>
            </div>

            {file && (
              <div className="flex items-center justify-between rounded-xl border border-neutral-200 bg-neutral-50 px-4 py-3">
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium">{file.name}</p>
                  <p className="text-xs text-neutral-500">
                    {(file.size / 1024 / 1024).toFixed(2)} MiB
                  </p>
                </div>
                <FileText className="size-5 shrink-0 text-neutral-400" />
              </div>
            )}

            {fileError && (
              <InlineError message={fileError} />
            )}

            {createInterviewMutation.isPending && (
              <div className="space-y-3 rounded-xl border border-neutral-200 p-4">
                <div className="flex items-center justify-between gap-4 text-sm">
                  <span className="flex items-center gap-2 font-medium">
                    <LoaderCircle className="size-4 animate-spin" />
                    {stageCopy[stage]}
                  </span>
                  <span className="text-neutral-500">{progress}%</span>
                </div>
                <div className="h-2 overflow-hidden rounded-full bg-neutral-100">
                  <div
                    className="h-full rounded-full bg-neutral-900 transition-[width] duration-300"
                    style={{ width: `${Math.min(100, progress)}%` }}
                  />
                </div>
                {job?.stage && (
                  <p className="text-xs text-neutral-500">
                    当前阶段：{job.stage}
                  </p>
                )}
              </div>
            )}

            {errorMessage && (
              <InlineError message={errorMessage} />
            )}

            <Button
              type="submit"
              size="lg"
              className="w-full"
              disabled={!file || createInterviewMutation.isPending}
            >
              {createInterviewMutation.isPending ? (
                <LoaderCircle className="animate-spin" />
              ) : (
                <Upload />
              )}
              {createInterviewMutation.isPending
                ? "正在准备面试..."
                : "上传简历并生成问题"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
