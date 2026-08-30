export type QuestionSpeechStatus = "idle" | "loading" | "playing" | "error";

export interface TtsTask {
  taskId: string;
  taskStatus: string;
  code: number;
  message?: string;
  completed: boolean;
  success: boolean;
  audioPath: string;
}
