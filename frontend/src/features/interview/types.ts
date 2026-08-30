export const INTERVIEW_STATUS = {
  created: "CREATED",
  analyzing: "ANALYZING",
  ready: "READY",
  inProgress: "IN_PROGRESS",
  completed: "COMPLETED",
  failed: "FAILED",
} as const;

export type InterviewStatus =
  (typeof INTERVIEW_STATUS)[keyof typeof INTERVIEW_STATUS];

export interface InterviewSession {
  sessionId: string;
  status: InterviewStatus;
  direction: string;
  resumeScore: number;
  currentQuestionNumber: string;
  currentMainIndex: number;
  followUpCount: number;
  totalScore: number;
  turnSequence: number;
  version: number;
  startedAt: string | null;
  completedAt: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface InterviewListPage {
  records: InterviewSession[];
  total: number;
  page: number;
  size: number;
}

export interface InterviewQuestion {
  number: string;
  content: string;
  isFollowUp: boolean;
}

export interface InterviewTurn {
  sequence: number;
  questionNumber: string;
  question: string;
  answer: string;
  score: number;
  feedback: string;
  isFollowUp: boolean;
}

export interface InterviewProgress {
  mainCompleted: number;
  mainTotal: number;
  followUpCount: number;
  totalScore: number;
}

export interface InterviewState {
  sessionId: string;
  status: InterviewStatus;
  canResume: boolean;
  version: number;
  resumeScore: number;
  interviewType: string;
  resumeFileUrl?: string;
  currentQuestion: InterviewQuestion | null;
  progress: InterviewProgress;
  recentTurns: InterviewTurn[];
}

export interface AnswerInterviewRequest {
  questionNumber: string;
  answerContent: string;
}

export interface AnswerInterviewResult {
  questionNumber: string;
  score: number;
  totalScore: number;
  feedback: string;
  missingPoints: string[];
  nextQuestion: InterviewQuestion | null;
  followUpNeeded: boolean;
  followUpCount: number;
  finished: boolean;
  version: number;
}

export interface InterviewRadarMetric {
  label: string;
  value: number;
}

export interface InterviewReport {
  id: string;
  sessionId: string;
  interviewScore: number;
  resumeScore: number;
  compositeScore: number;
  overallComment: string;
  highlights: string[];
  improvementTips: string[];
  nextActions: string[];
  radarMetrics: InterviewRadarMetric[];
  createdAt: string;
  updatedAt: string;
  turns: InterviewTurn[];
}

export type AnalysisJobStatus =
  | "QUEUED"
  | "PROCESSING"
  | "COMPLETED"
  | "FAILED";

export interface AnalysisJob {
  jobId: string;
  sessionId: string;
  status: AnalysisJobStatus;
  progress: number;
  stage: string;
  attempts: number;
  errorCode?: string;
  errorMessage?: string;
  createdAt: string;
  updatedAt: string;
}
