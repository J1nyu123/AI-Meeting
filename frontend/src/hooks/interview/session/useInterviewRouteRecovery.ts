import { useEffect, useRef } from "react";
import { isMessageInWelcomeState } from "@/hooks/interview/session/interviewSessionFlow.shared";
import type { ChatMessage } from "@/lib/chat";

type UseInterviewRouteRecoveryParams = {
  routeSessionId: string | null;
  storedInterviewerSessionId: string | null;
  interviewerSessionId: string | null;
  persistInterviewerSessionId: (sessionId: string | null) => void;
  messages: ChatMessage[];
  syncNextQuestion: (
    sessionId: string,
    options?: { appendMessage?: boolean; restoreHistory?: boolean },
  ) => Promise<void>;
  setInterviewError: (message: string | null) => void;
};

export function useInterviewRouteRecovery({
  routeSessionId,
  storedInterviewerSessionId,
  interviewerSessionId,
  persistInterviewerSessionId,
  messages,
  syncNextQuestion,
  setInterviewError,
}: UseInterviewRouteRecoveryParams) {
  const restoredSessionRef = useRef<string | null>(null);

  useEffect(() => {
    if (!routeSessionId) {
      return;
    }
    if (storedInterviewerSessionId === routeSessionId) {
      return;
    }
    persistInterviewerSessionId(routeSessionId);
  }, [persistInterviewerSessionId, routeSessionId, storedInterviewerSessionId]);

  useEffect(() => {
    if (
      !interviewerSessionId ||
      restoredSessionRef.current === interviewerSessionId ||
      !isMessageInWelcomeState(messages)
    ) {
      return;
    }

    restoredSessionRef.current = interviewerSessionId;
    syncNextQuestion(interviewerSessionId, { restoreHistory: true }).catch(
      (error) => {
        restoredSessionRef.current = null;
        const message =
          error instanceof Error
            ? error.message
            : "Failed to restore interview state";
        setInterviewError(message);
      },
    );
  }, [interviewerSessionId, messages, setInterviewError, syncNextQuestion]);
}
