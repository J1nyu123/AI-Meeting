import { Button } from "@/components/ui/button";
import { Loader2, Trash2 } from "lucide-react";
import type { InterviewRecordResult } from "@/services/interviewService";
import { ROUTES } from "@/lib/constants";

type SidebarInterviewListProps = {
  records: InterviewRecordResult[];
  activePathname: string;
  activeSessionId: string | null;
  hasNextPage: boolean | undefined;
  isFetchingNextPage: boolean;
  onOpenRecord: (record: InterviewRecordResult) => void;
  onRequestDelete: (record: InterviewRecordResult) => void;
  deletingSessionId?: string | null;
};

const formatDate = (value?: string | null) => {
  if (!value) return "Unknown date";

  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return parsed.toLocaleDateString();
};

const formatScore = (value?: number | null) => {
  if (typeof value !== "number" || Number.isNaN(value)) return "--";
  return String(value);
};

export default function SidebarInterviewList({
  records,
  activePathname,
  activeSessionId,
  hasNextPage,
  isFetchingNextPage,
  onOpenRecord,
  onRequestDelete,
  deletingSessionId,
}: SidebarInterviewListProps) {
  return (
    <>
      {records.map((record) => {
        const isActive =
          activePathname.startsWith(ROUTES.interviewReport) &&
          activeSessionId === record.sessionId;
        const status = record.interviewStatus?.toUpperCase();
        const canDelete = status === "COMPLETED" || status === "FAILED";

        return (
          <div
            key={record.sessionId}
            className="mb-1 flex items-center gap-1 rounded-xl"
          >
            <Button
              variant={isActive ? "secondary" : "ghost"}
              className="h-auto min-w-0 flex-1 justify-start rounded-xl px-3 py-2 text-left font-normal hover:bg-slate-100"
              onClick={() => onOpenRecord(record)}
            >
              <div className="flex w-full flex-col gap-0.5 overflow-hidden">
                <span className="truncate text-sm font-medium text-slate-700">
                  {record.interviewDirection || "面试记录"}
                </span>
                <span className="truncate text-[10px] text-slate-400">
                  {formatDate(record.startTime || record.createTime)} · 得分{" "}
                  {formatScore(record.interviewScore)}
                </span>
              </div>
            </Button>
            {canDelete ? (
              <Button
                type="button"
                size="icon"
                variant="ghost"
                aria-label={`删除 ${record.interviewDirection || "面试记录"}`}
                className="h-8 w-8 shrink-0 text-slate-400 hover:bg-red-50 hover:text-red-600"
                disabled={deletingSessionId === record.sessionId}
                onClick={(event) => {
                  event.stopPropagation();
                  onRequestDelete(record);
                }}
              >
                {deletingSessionId === record.sessionId ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Trash2 className="h-4 w-4" />
                )}
              </Button>
            ) : null}
          </div>
        );
      })}

      {isFetchingNextPage ? (
        <div className="py-2 text-center text-xs text-slate-400">加载中...</div>
      ) : null}

      {!hasNextPage && records.length > 0 ? (
        <div className="py-2 text-center text-xs text-slate-300">
          没有更多了
        </div>
      ) : null}

      {!isFetchingNextPage && records.length === 0 ? (
        <div className="py-4 text-center text-xs text-slate-400">
          暂无面试记录
        </div>
      ) : null}
    </>
  );
}
