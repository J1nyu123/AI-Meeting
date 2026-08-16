import { useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ROUTES } from "@/lib/constants";
import {
  buildReportSearch,
  getReportSessionIdFromLocation,
} from "@/lib/interviewReportRoute";
import SidebarInterviewList from "@/components/layout/sidebar/SidebarInterviewList";
import { useSidebarHistoryController } from "@/hooks/layout/useSidebarHistoryController";
import {
  interviewService,
  type InterviewRecordResult,
} from "@/services/interviewService";

type SidebarHistoryProps = {
  isCollapsed?: boolean;
};

export const getInterviewRecordTarget = (record: InterviewRecordResult) =>
  record.interviewStatus?.toUpperCase() === "COMPLETED"
    ? `${ROUTES.interviewReport}${buildReportSearch(record.sessionId)}`
    : `${ROUTES.interviewRoom}/${encodeURIComponent(record.sessionId)}`;

export default function SidebarHistory({ isCollapsed }: SidebarHistoryProps) {
  const navigate = useNavigate();
  const location = useLocation();
  const queryClient = useQueryClient();
  const [recordToDelete, setRecordToDelete] =
    useState<InterviewRecordResult | null>(null);
  const [deletingSessionId, setDeletingSessionId] = useState<string | null>(
    null,
  );
  const [deleteError, setDeleteError] = useState("");
  const {
    interviewRecords,
    hasNextInterviewPage,
    isFetchingNextInterviewPage,
    handleScroll,
  } = useSidebarHistoryController(isCollapsed);

  const activeInterviewSessionId = location.pathname.startsWith(
    ROUTES.interviewReport,
  )
    ? getReportSessionIdFromLocation(location)
    : null;

  if (isCollapsed) {
    return null;
  }

  const handleDelete = async () => {
    if (!recordToDelete || deletingSessionId) return;
    const sessionId = recordToDelete.sessionId;
    setDeletingSessionId(sessionId);
    setDeleteError("");
    try {
      await interviewService.deleteInterview(sessionId);
      await queryClient.invalidateQueries({ queryKey: ["interview-records"] });
      setRecordToDelete(null);
      if (activeInterviewSessionId === sessionId) {
        navigate(ROUTES.interviewIntro);
      }
    } catch (error) {
      setDeleteError(
        error instanceof Error ? error.message : "删除失败，请稍后重试",
      );
    } finally {
      setDeletingSessionId(null);
    }
  };

  return (
    <>
      <div className="mt-6 flex min-h-0 flex-1 flex-col overflow-hidden px-3">
        <div className="mb-2 shrink-0 px-2 text-xs text-slate-400">
          历史面试
        </div>

        <ScrollArea className="-mx-2 flex-1" onScrollCapture={handleScroll}>
          <div className="space-y-1 px-2 pb-2">
            <SidebarInterviewList
              records={interviewRecords}
              activePathname={location.pathname}
              activeSessionId={activeInterviewSessionId}
              hasNextPage={hasNextInterviewPage}
              isFetchingNextPage={isFetchingNextInterviewPage}
              deletingSessionId={deletingSessionId}
              onRequestDelete={(record) => {
                setDeleteError("");
                setRecordToDelete(record);
              }}
              onOpenRecord={(record) => {
                const target = getInterviewRecordTarget(record);
                navigate(target, {
                  state:
                    record.interviewStatus?.toUpperCase() === "COMPLETED"
                      ? { sessionId: record.sessionId }
                      : undefined,
                });
              }}
            />
          </div>
        </ScrollArea>
      </div>

      <Dialog
        open={Boolean(recordToDelete)}
        onOpenChange={(open) => {
          if (!open && !deletingSessionId) setRecordToDelete(null);
        }}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>删除历史面试</DialogTitle>
            <DialogDescription>
              将永久删除“{recordToDelete?.interviewDirection || "面试记录"}”
              {recordToDelete?.startTime
                ? `（${new Date(recordToDelete.startTime).toLocaleDateString()}）`
                : ""}
              ，此操作不可恢复。
            </DialogDescription>
          </DialogHeader>
          {deleteError ? (
            <p role="alert" className="text-sm text-red-600">
              {deleteError}
            </p>
          ) : null}
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              disabled={Boolean(deletingSessionId)}
              onClick={() => setRecordToDelete(null)}
            >
              取消
            </Button>
            <Button
              type="button"
              variant="destructive"
              disabled={Boolean(deletingSessionId)}
              onClick={handleDelete}
            >
              {deletingSessionId ? "删除中..." : "确认删除"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
