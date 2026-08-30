import { useQuery } from "@tanstack/react-query";
import { Download, ExternalLink, FileText, X } from "lucide-react";
import { useEffect, useState, type ReactNode } from "react";
import { ErrorState, PageLoading } from "@/components/feedback/page-state";
import { Button } from "@/components/ui/button";
import { interviewApi } from "@/features/interview/interview-api";
import { getErrorMessage } from "@/lib/errors";

interface ResumePreviewPanelProps {
  sessionId: string;
  onClose: () => void;
}

function PanelShell({ children }: { children: ReactNode }) {
  return (
    <aside className="fixed inset-0 z-50 flex min-h-0 flex-col border-l border-neutral-200 bg-white shadow-2xl xl:static xl:z-auto xl:w-[40vw] xl:min-w-[28rem] xl:max-w-[42rem] xl:shadow-none">
      {children}
    </aside>
  );
}

function PanelHeader({
  filename,
  objectUrl,
  onClose,
}: {
  filename?: string;
  objectUrl?: string;
  onClose: () => void;
}) {
  return (
    <header className="flex h-14 shrink-0 items-center gap-3 border-b border-neutral-200 px-3">
      <div className="flex size-8 items-center justify-center rounded-lg bg-neutral-100">
        <FileText className="size-4" />
      </div>
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium">简历预览</p>
        <p className="truncate text-xs text-neutral-500">
          {filename || "正在加载 PDF"}
        </p>
      </div>

      {objectUrl && filename && (
        <div className="flex items-center gap-1">
          <Button asChild variant="ghost" size="icon-sm">
            <a
              href={objectUrl}
              target="_blank"
              rel="noreferrer"
              aria-label="在新标签页打开简历"
            >
              <ExternalLink />
            </a>
          </Button>
          <Button asChild variant="ghost" size="icon-sm">
            <a href={objectUrl} download={filename} aria-label="下载简历 PDF">
              <Download />
            </a>
          </Button>
        </div>
      )}

      <Button
        type="button"
        variant="ghost"
        size="icon-sm"
        aria-label="关闭简历预览"
        onClick={onClose}
      >
        <X />
      </Button>
    </header>
  );
}

function PdfDocument({
  blob,
  filename,
  onClose,
}: {
  blob: Blob;
  filename: string;
  onClose: () => void;
}) {
  const [objectUrl] = useState(() => URL.createObjectURL(blob));

  useEffect(
    () => () => {
      URL.revokeObjectURL(objectUrl);
    },
    [objectUrl],
  );

  return (
    <PanelShell>
      <PanelHeader
        filename={filename}
        objectUrl={objectUrl}
        onClose={onClose}
      />
      <div className="min-h-0 flex-1 bg-neutral-100 p-2 sm:p-3">
        <iframe
          title={`简历预览：${filename}`}
          src={`${objectUrl}#view=FitH&toolbar=1&navpanes=0`}
          className="h-full w-full rounded-lg border border-neutral-200 bg-white shadow-sm"
        />
      </div>
    </PanelShell>
  );
}

export function ResumePreviewPanel({
  sessionId,
  onClose,
}: ResumePreviewPanelProps) {
  const previewQuery = useQuery({
    queryKey: ["resume-preview", sessionId],
    queryFn: ({ signal }) => interviewApi.resumePreview(sessionId, signal),
    staleTime: 5 * 60 * 1000,
    retry: false,
  });

  if (previewQuery.data) {
    return (
      <PdfDocument
        blob={previewQuery.data.blob}
        filename={previewQuery.data.filename}
        onClose={onClose}
      />
    );
  }

  return (
    <PanelShell>
      <PanelHeader onClose={onClose} />
      {previewQuery.isLoading ? (
        <PageLoading
          className="min-h-0 flex-1"
          title="正在加载简历"
          description="正在安全获取 PDF 文件。"
        />
      ) : (
        <div className="p-4">
          <ErrorState
            title="无法预览简历"
            message={getErrorMessage(previewQuery.error, "简历 PDF 加载失败")}
            isRetrying={previewQuery.isFetching}
            onRetry={() => void previewQuery.refetch()}
          />
        </div>
      )}
    </PanelShell>
  );
}
