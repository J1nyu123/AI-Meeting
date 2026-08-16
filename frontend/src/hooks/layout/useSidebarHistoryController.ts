import { useCallback, type UIEvent } from "react";
import { useInterviewRecords } from "@/hooks/interview/records/useInterviewRecords";

export function useSidebarHistoryController(isCollapsed?: boolean) {
  const {
    interviewRecords,
    fetchNextPage: fetchNextInterviewPage,
    hasNextPage: hasNextInterviewPage,
    isFetchingNextPage: isFetchingNextInterviewPage,
  } = useInterviewRecords({
    enabled: !isCollapsed,
  });

  const handleScroll = useCallback(
    (event: UIEvent<HTMLDivElement>) => {
      const { scrollTop, clientHeight, scrollHeight } = event.currentTarget;

      if (scrollHeight - scrollTop > clientHeight * 1.5) {
        return;
      }

      if (hasNextInterviewPage && !isFetchingNextInterviewPage) {
        fetchNextInterviewPage();
      }
    },
    [fetchNextInterviewPage, hasNextInterviewPage, isFetchingNextInterviewPage],
  );

  return {
    interviewRecords,
    hasNextInterviewPage,
    isFetchingNextInterviewPage,
    handleScroll,
  };
}
