import { ApiError, apiRawRequest, apiRequest } from "@/api/client";
import type {
  AnalysisJob,
  AnswerInterviewRequest,
  AnswerInterviewResult,
  InterviewListPage,
  InterviewReport,
  InterviewSession,
  InterviewState,
} from "./types";

export const interviewApi = {
  list(page = 1, size = 50) {
    const query = new URLSearchParams({
      page: String(page),
      size: String(size),
    });

    return apiRequest<InterviewListPage>(`/interviews?${query}`);
  },

  create() {
    return apiRequest<InterviewSession>("/interviews", {
      method: "POST",
    });
  },

  delete(sessionId: string) {
    return apiRequest<{ sessionId: string }>(
      `/interviews/${encodeURIComponent(sessionId)}`,
      {
        method: "DELETE",
      },
    );
  },

  uploadResume(sessionId: string, file: File) {
    const formData = new FormData();
    formData.append("resume", file);

    return apiRequest<AnalysisJob>(
      `/interviews/${encodeURIComponent(sessionId)}/resume`,
      {
        method: "POST",
        body: formData,
      },
    );
  },

  async resumePreview(sessionId: string, signal?: AbortSignal) {
    const response = await apiRawRequest(
      `/interviews/${encodeURIComponent(sessionId)}/resume`,
      { signal },
    );
    const disposition = response.headers.get("Content-Disposition") || "";
    const encodedName = disposition.match(/filename\*=UTF-8''([^;]+)/i)?.[1];
    const quotedName = disposition.match(/filename="([^"]+)"/i)?.[1];
    const fallbackName = `resume-${sessionId}.pdf`;
    let filename = quotedName || fallbackName;

    if (encodedName) {
      try {
        filename = decodeURIComponent(encodedName);
      } catch {
        filename = fallbackName;
      }
    }

    return {
      blob: await response.blob(),
      filename,
    };
  },

  async state(sessionId: string) {
    const state = await apiRequest<InterviewState>(
      `/interviews/${encodeURIComponent(sessionId)}/state`,
    );

    return {
      ...state,
      recentTurns: state.recentTurns ?? [],
    };
  },

  answer(
    sessionId: string,
    request: AnswerInterviewRequest,
    idempotencyKey: string,
  ) {
    return apiRequest<AnswerInterviewResult>(
      `/interviews/${encodeURIComponent(sessionId)}/answers`,
      {
        method: "POST",
        headers: {
          "Idempotency-Key": idempotencyKey,
        },
        body: JSON.stringify(request),
      },
    );
  },

  finish(sessionId: string) {
    return apiRequest<InterviewReport>(
      `/interviews/${encodeURIComponent(sessionId)}/finish`,
      {
        method: "POST",
      },
    );
  },

  report(sessionId: string) {
    return apiRequest<InterviewReport>(
      `/interviews/${encodeURIComponent(sessionId)}/report`,
    );
  },

  async ensureReport(sessionId: string) {
    try {
      return await interviewApi.report(sessionId);
    } catch (error) {
      if (error instanceof ApiError && error.code === "REPORT_NOT_FOUND") {
        const state = await interviewApi.state(sessionId);
        if (state.status === "COMPLETED") {
          return interviewApi.finish(sessionId);
        }
      }
      throw error;
    }
  },
};
