export interface Session {
  id: string;
  user_id: string;
  name: string;
  started_at: string;
  stopped_at: string | null;
  active: boolean;
  duration: string | null;
}

export interface CreateSessionResponse {
  session: Session;
}

export interface SessionsResponse {
  sessions: Session[];
}

export interface APIResponse<T> {
  error: string;
  data: T | null;
}