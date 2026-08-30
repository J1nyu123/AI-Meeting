import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { interviewApi } from "@/features/interview/interview-api";
import { ResumePreviewPanel } from "./ResumePreviewPanel";

describe("ResumePreviewPanel", () => {
  const createObjectUrl = vi.fn(() => "blob:resume-preview");
  const revokeObjectUrl = vi.fn();

  beforeEach(() => {
    Object.defineProperty(URL, "createObjectURL", {
      configurable: true,
      value: createObjectUrl,
    });
    Object.defineProperty(URL, "revokeObjectURL", {
      configurable: true,
      value: revokeObjectUrl,
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("renders an authenticated PDF and releases its object URL", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    vi.spyOn(interviewApi, "resumePreview").mockResolvedValue({
      blob: new Blob(["%PDF-test"], { type: "application/pdf" }),
      filename: "candidate.pdf",
    });
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    const view = render(
      <QueryClientProvider client={queryClient}>
        <ResumePreviewPanel sessionId="session-1" onClose={onClose} />
      </QueryClientProvider>,
    );

    const frame = await screen.findByTitle("简历预览：candidate.pdf");
    expect(frame).toHaveAttribute(
      "src",
      "blob:resume-preview#view=FitH&toolbar=1&navpanes=0",
    );
    expect(screen.getByLabelText("下载简历 PDF")).toHaveAttribute(
      "download",
      "candidate.pdf",
    );

    await user.click(screen.getByRole("button", { name: "关闭简历预览" }));
    expect(onClose).toHaveBeenCalledOnce();

    view.unmount();
    await waitFor(() => {
      expect(revokeObjectUrl).toHaveBeenCalledWith("blob:resume-preview");
    });
  });
});
