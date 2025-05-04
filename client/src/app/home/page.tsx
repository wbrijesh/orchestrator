"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useUser } from "@/context/UserContext";
import Navbar from "@/components/custom/navbar";
import {
  Session,
  SessionsResponse,
  CreateSessionResponse,
  APIResponse,
} from "@/types/session";
import { TbClockPlay, TbClockStop, TbClockPlus, TbTrash } from "react-icons/tb";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";

// API URL base
const API_URL = "http://localhost:8080";

export default function Home() {
  const router = useRouter();
  const { user, loading, getAuthHeader } = useUser();
  const [sessions, setSessions] = useState<Session[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [sessionToDelete, setSessionToDelete] = useState<string | null>(null);

  // Function to fetch user sessions
  const fetchSessions = async () => {
    if (!user) return;

    try {
      setIsLoading(true);
      const response = await fetch(`${API_URL}/sessions`, {
        method: "GET",
        headers: {
          ...getAuthHeader(),
          "Content-Type": "application/json",
        },
      });

      const data: APIResponse<SessionsResponse> = await response.json();

      if (data.error) {
        setError(data.error);
        return;
      }

      setSessions(data.data?.sessions || []);
    } catch (err) {
      setError("Failed to fetch sessions");
      console.error(err);
    } finally {
      setIsLoading(false);
    }
  };

  // Function to create a new session
  const createSession = async () => {
    if (!user) return;

    try {
      setIsLoading(true);
      const response = await fetch(`${API_URL}/sessions`, {
        method: "POST",
        headers: {
          ...getAuthHeader(),
          "Content-Type": "application/json",
        },
      });

      const data: APIResponse<CreateSessionResponse> = await response.json();

      if (data.error) {
        setError(data.error);
        return;
      }

      // Refresh sessions
      fetchSessions();
    } catch (err) {
      setError("Failed to create session");
      console.error(err);
    } finally {
      setIsLoading(false);
    }
  };

  // Function to stop a session
  const stopSession = async (sessionId: string) => {
    if (!user) return;

    try {
      setIsLoading(true);
      const response = await fetch(`${API_URL}/sessions/${sessionId}/stop`, {
        method: "POST",
        headers: {
          ...getAuthHeader(),
          "Content-Type": "application/json",
        },
      });

      const data = await response.json();

      if (data.error) {
        setError(data.error);
        return;
      }

      // Refresh sessions
      fetchSessions();
    } catch (err) {
      setError("Failed to stop session");
      console.error(err);
    } finally {
      setIsLoading(false);
    }
  };

  // Function to delete a session
  const deleteSession = async (sessionId: string) => {
    if (!user) return;
    
    try {
      setIsLoading(true);
      const response = await fetch(`${API_URL}/sessions/${sessionId}`, {
        method: "DELETE",
        headers: {
          ...getAuthHeader(),
          "Content-Type": "application/json",
        },
      });

      const data = await response.json();
      
      if (data.error) {
        setError(data.error);
        return;
      }

      // Refresh sessions
      fetchSessions();
    } catch (err) {
      setError("Failed to delete session");
      console.error(err);
    } finally {
      setIsLoading(false);
    }
  };

  // Fetch sessions on mount
  useEffect(() => {
    if (!loading && !user) {
      router.push("/login");
    } else if (user) {
      fetchSessions();
    }
  }, [user, loading, router]);

  if (loading) {
    return (
      <div>
        <Navbar />
        <div className="container mx-auto p-4">
          <p>Loading...</p>
        </div>
      </div>
    );
  }

  // Don't render anything while redirecting
  if (!user) {
    return null;
  }

  return (
    <div>
      <Navbar />
      <div className="container mx-auto p-4">
        <div className="flex justify-between items-center mb-6">
          <h1 className="text-xl font-medium">My Sessions</h1>
          <button
            onClick={createSession}
            disabled={isLoading}
            className="px-3 py-1.5 text-sm bg-sky-600 text-white rounded-md hover:bg-sky-700 transition-colors flex items-center gap-1.5"
          >
            <TbClockPlus className="text-white" size={18} />
            New Session
          </button>
        </div>

        {error && (
          <div className="p-3 mb-4 text-sm bg-red-100 border border-red-200 text-red-800 rounded">
            {error}
            <button
              className="ml-2 text-red-600 hover:text-red-800"
              onClick={() => setError(null)}
            >
              Dismiss
            </button>
          </div>
        )}

        {isLoading ? (
          <p>Loading sessions...</p>
        ) : sessions.length === 0 ? (
          <div className="text-center py-8">
            <div className="mb-3 flex justify-center">
              <TbClockPlay size={48} className="text-neutral-300" />
            </div>
            <h3 className="text-lg font-medium mb-1">No sessions found</h3>
            <p className="text-neutral-500 mb-4">
              You haven&apos;t created any sessions yet.
            </p>
            <button
              onClick={createSession}
              className="px-3 py-1.5 text-sm bg-sky-600 text-white rounded-md hover:bg-sky-700 transition-colors"
            >
              Create Your First Session
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
            {sessions.map((session) => (
              <div
                key={session.id}
                className="border border-neutral-200 rounded-lg p-4 shadow-sm"
              >
                <div className="flex justify-between items-start mb-2">
                  <h3 className="font-medium text-neutral-800">
                    {session.name}
                  </h3>
                  {session.active ? (
                    <span className="px-2 py-0.5 bg-green-100 text-green-800 text-xs rounded-full flex items-center gap-1">
                      <span className="h-1.5 w-1.5 rounded-full bg-green-600"></span>
                      Active
                    </span>
                  ) : (
                    <span className="px-2 py-0.5 bg-neutral-100 text-neutral-600 text-xs rounded-full">
                      Completed
                    </span>
                  )}
                </div>

                <p className="text-xs text-neutral-500 mb-3">
                  Started: {new Date(session.started_at).toLocaleString()}
                </p>

                {session.stopped_at && (
                  <p className="text-xs text-neutral-500 mb-3">
                    Stopped: {new Date(session.stopped_at).toLocaleString()}
                  </p>
                )}

                {session.duration && (
                  <p className="text-xs text-neutral-600 mb-3">
                    Duration: {session.duration}
                  </p>
                )}

                <div className="flex space-x-2">
                  {session.active && (
                    <button
                      onClick={() => stopSession(session.id)}
                      className="flex-1 px-3 py-1.5 text-xs bg-neutral-100 text-neutral-700 rounded-md hover:bg-neutral-200 transition-colors flex items-center justify-center gap-1.5"
                    >
                      <TbClockStop size={16} />
                      Stop
                    </button>
                  )}
                  
                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      <button
                        className="px-3 py-1.5 text-xs bg-neutral-100 text-red-600 rounded-md hover:bg-red-50 transition-colors flex items-center justify-center gap-1.5"
                      >
                        <TbTrash size={16} />
                        {!session.active && "Delete"}
                      </button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>Delete Session</AlertDialogTitle>
                        <AlertDialogDescription>
                          Are you sure you want to delete this session? This action cannot be undone.
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction onClick={() => deleteSession(session.id)}>
                          Delete
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
