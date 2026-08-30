import { fetchEventSource } from "@microsoft/fetch-event-source";
import { ApiError, getAccessToken } from "@/api/client";
import type { AnalysisJob } from "./types";

export function waitForAnalysisJob(
  jobId: string,
  onUpdate: (job: AnalysisJob) => void,
  externalSignal?: AbortSignal,
) {
  return new Promise<AnalysisJob>((resolve, reject) => {
    const accessToken = getAccessToken();
    const controller = new AbortController();
    let settled = false;

    const cleanUp = () => {
      externalSignal?.removeEventListener("abort", handleExternalAbort);
    };

    const resolveOnce = (job: AnalysisJob) => {
      if (settled) return;
      settled = true;
      cleanUp();
      resolve(job);
      controller.abort();
    };

    const rejectOnce = (error: unknown) => {
      if (settled) return;
      settled = true;
      cleanUp();
      reject(error);
      controller.abort();
    };

    function handleExternalAbort() {
      rejectOnce(new DOMException("分析已取消", "AbortError"));
    }

    if (!accessToken) {
      rejectOnce(new ApiError(401, "UNAUTHORIZED", "请先登录"));
      return;
    }

    if (externalSignal?.aborted) {
      handleExternalAbort();
      return;
    }

    externalSignal?.addEventListener("abort", handleExternalAbort, {
      once: true,
    });

    void fetchEventSource(
      `/api/v1/jobs/${encodeURIComponent(jobId)}/events`,
      {
        headers: {
          Authorization: `Bearer ${accessToken}`,
        },
        signal: controller.signal,
        openWhenHidden: true,
        async onopen(response) {
          if (!response.ok) {
            throw new ApiError(
              response.status,
              "ANALYSIS_STREAM_FAILED",
              "无法连接简历分析进度服务",
            );
          }
        },
        onmessage(message) {
          if (message.event !== "status") return;

          try {
            const job = JSON.parse(message.data) as AnalysisJob;
            onUpdate(job);

            if (job.status === "COMPLETED") {
              resolveOnce(job);
            } else if (job.status === "FAILED") {
              rejectOnce(
                new Error(
                  job.errorMessage || job.errorCode || "简历分析失败",
                ),
              );
            }
          } catch (error) {
            rejectOnce(error);
          }
        },
        onerror(error) {
          rejectOnce(error);
          throw error;
        },
      },
    ).catch((error: unknown) => {
      if (!settled && !controller.signal.aborted) {
        rejectOnce(error);
      }
    });
  });
}
