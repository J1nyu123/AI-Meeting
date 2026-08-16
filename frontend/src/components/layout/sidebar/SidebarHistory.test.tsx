import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ROUTES } from "@/lib/constants";
import SidebarHistory, {
  getInterviewRecordTarget,
} from "@/components/layout/sidebar/SidebarHistory";
import type { InterviewRecordResult } from "@/services/interviewService";

const mocks = vi.hoisted(() => ({
  deleteInterview: vi.fn(),
  useSidebarHistoryController: vi.fn(),
}));

vi.mock("@/services/interviewService", () => ({
  interviewService: { deleteInterview: mocks.deleteInterview },
}));

vi.mock("@/hooks/layout/useSidebarHistoryController", () => ({
  useSidebarHistoryController: mocks.useSidebarHistoryController,
}));

const record = (
  sessionId: string,
  interviewStatus: string,
): InterviewRecordResult => ({
  id: 0,
  userId: 1,
  sessionId,
  interviewStatus,
});

describe("getInterviewRecordTarget", () => {
  it("opens an in-progress interview in its room", () => {
    expect(
      getInterviewRecordTarget(record("active-session", "IN_PROGRESS")),
    ).toBe(`${ROUTES.interviewRoom}/active-session`);
  });

  it("opens a completed interview in its report", () => {
    expect(getInterviewRecordTarget(record("done-session", "COMPLETED"))).toBe(
      `${ROUTES.interviewReport}?sessionId=done-session`,
    );
  });
});

describe("SidebarHistory deletion", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.useSidebarHistoryController.mockReturnValue({
      interviewRecords: [
        {
          ...record("done-session", "COMPLETED"),
          interviewDirection: "Go 后端开发",
          startTime: "2026-08-15T10:00:00Z",
        },
        record("active-session", "IN_PROGRESS"),
      ],
      hasNextInterviewPage: false,
      isFetchingNextInterviewPage: false,
      handleScroll: vi.fn(),
    });
  });

  const renderHistory = () => {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const invalidate = vi.spyOn(client, "invalidateQueries");
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter>
          <SidebarHistory />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    return { invalidate };
  };

  it("only offers deletion for terminal interviews and cancel sends no request", () => {
    renderHistory();

    expect(screen.getAllByLabelText(/^删除 /)).toHaveLength(1);
    fireEvent.click(screen.getByLabelText("删除 Go 后端开发"));
    expect(screen.getByText("删除历史面试")).toBeTruthy();
    fireEvent.click(screen.getByText("取消"));
    expect(mocks.deleteInterview).not.toHaveBeenCalled();
  });

  it("deletes after confirmation and refreshes interview records", async () => {
    mocks.deleteInterview.mockResolvedValue({ sessionId: "done-session" });
    const { invalidate } = renderHistory();

    fireEvent.click(screen.getByLabelText("删除 Go 后端开发"));
    fireEvent.click(screen.getByText("确认删除"));

    await waitFor(() =>
      expect(mocks.deleteInterview).toHaveBeenCalledWith("done-session"),
    );
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: ["interview-records"],
    });
  });

  it("keeps the dialog open and shows an error when deletion fails", async () => {
    mocks.deleteInterview.mockRejectedValue(new Error("暂时无法删除"));
    renderHistory();

    fireEvent.click(screen.getByLabelText("删除 Go 后端开发"));
    fireEvent.click(screen.getByText("确认删除"));

    expect(await screen.findByRole("alert")).toHaveProperty(
      "textContent",
      "暂时无法删除",
    );
    expect(screen.getByText("删除历史面试")).toBeTruthy();
  });
});
