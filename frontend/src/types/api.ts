export interface ApiResponse<T> {
  success: boolean;
  code: string;
  message: string | null;
  data: T;
  requestId: string;
}
