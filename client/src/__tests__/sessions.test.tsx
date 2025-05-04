import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent, within } from "@testing-library/react";
import Home from "../app/home/page";
import { UserProvider, useUser } from "../context/UserContext";
import { Session } from "@/types/session";

// Mock Next.js navigation
const mockRouter = { push: vi.fn() };
vi.mock("next/navigation", () => ({
  useRouter: () => mockRouter,
}));

// Mock React Icons
vi.mock("react-icons/tb", () => ({
  TbClockPlay: () => <span data-testid="icon-play">PlayIcon</span>,
  TbClockStop: () => <span data-testid="icon-stop">StopIcon</span>,
  TbClockPlus: () => <span data-testid="icon-plus">PlusIcon</span>,
  TbTrash: () => <span data-testid="icon-trash">TrashIcon</span>,
  TbUser: () => <span data-testid="icon-user">UserIcon</span>,
  TbLogout: () => <span data-testid="icon-logout">LogoutIcon</span>,
  TbSettings: () => <span data-testid="icon-settings">SettingsIcon</span>
}));

// Mock the UserContext
vi.mock("../context/UserContext", () => {
  const mockUseUser = vi.fn().mockReturnValue({
    user: {
      id: "user1",
      email: "test@example.com",
      firstName: "Test",
      lastName: "User",
    },
    loading: false,
    login: vi.fn().mockResolvedValue(undefined),
    register: vi.fn().mockResolvedValue(undefined),
    logout: vi.fn(),
    clearError: vi.fn(),
    getAuthHeader: () => ({ Authorization: "Bearer mock-token" }),
  });
  
  return {
    UserProvider: ({ children }: { children: React.ReactNode }) => <div data-testid="user-provider-mock">{children}</div>,
    useUser: mockUseUser,
  };
});

// Mock alert dialog from shadcn/ui
vi.mock("@/components/ui/alert-dialog", () => ({
  AlertDialog: ({ children }: { children: React.ReactNode }) => <div data-testid="alert-dialog">{children}</div>,
  AlertDialogTrigger: ({ children }: { children: React.ReactNode }) => <div data-testid="alert-dialog-trigger">{children}</div>,
  AlertDialogContent: ({ children }: { children: React.ReactNode }) => <div data-testid="alert-dialog-content">{children}</div>,
  AlertDialogHeader: ({ children }: { children: React.ReactNode }) => <div data-testid="alert-dialog-header">{children}</div>,
  AlertDialogTitle: ({ children }: { children: React.ReactNode }) => <div data-testid="alert-dialog-title">{children}</div>,
  AlertDialogDescription: ({ children }: { children: React.ReactNode }) => <div data-testid="alert-dialog-description">{children}</div>,
  AlertDialogFooter: ({ children }: { children: React.ReactNode }) => <div data-testid="alert-dialog-footer">{children}</div>,
  AlertDialogCancel: ({ children }: { children: React.ReactNode }) => <button data-testid="alert-dialog-cancel">{children}</button>,
  AlertDialogAction: ({ onClick, children }: { onClick?: () => void, children: React.ReactNode }) => <button data-testid="alert-dialog-action" onClick={onClick}>{children}</button>,
}));

// Mock Navbar component
vi.mock("@/components/custom/navbar", () => ({
  default: () => <div data-testid="navbar">Navbar</div>,
}));

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: vi.fn((key: string) => store[key] || null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value.toString();
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key];
    }),
    clear: vi.fn(() => {
      store = {};
    }),
  };
})();

Object.defineProperty(window, "localStorage", {
  value: localStorageMock,
});

// Mock session data
const mockSessions: Session[] = [
  {
    id: "1",
    user_id: "user1",
    name: "Session 1",
    started_at: "2025-05-01T10:00:00Z",
    stopped_at: null,
    active: true,
    duration: null,
  },
  {
    id: "2",
    user_id: "user1",
    name: "Session 2",
    started_at: "2025-04-30T08:00:00Z",
    stopped_at: "2025-04-30T10:00:00Z",
    active: false,
    duration: "2 hours",
  },
];

describe("Session Management", () => {
  beforeEach(() => {
    // Reset mocks between tests
    vi.clearAllMocks();
    localStorageMock.clear();

    // Reset router mock
    mockRouter.push.mockReset();

    // Setup localStorage with user data
    localStorageMock.setItem("user", JSON.stringify({
      id: "user1",
      email: "test@example.com",
      firstName: "Test",
      lastName: "User",
    }));
    localStorageMock.setItem("token", "mock-token");

    // Reset the useUser mock to default values
    const mockUseUser = vi.mocked(useUser);
    mockUseUser.mockReturnValue({
      user: {
        id: "user1",
        email: "test@example.com",
        firstName: "Test",
        lastName: "User",
      },
      loading: false,
      login: vi.fn().mockResolvedValue(undefined),
      register: vi.fn().mockResolvedValue(undefined),
      logout: vi.fn(),
      clearError: vi.fn(),
      getAuthHeader: () => ({ Authorization: "Bearer mock-token" }),
    });

    // Mock fetch globally
    global.fetch = vi.fn().mockImplementation((url) => {
      if (url.includes("/sessions") && !url.includes("/stop")) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({
            error: "",
            data: { sessions: mockSessions },
          }),
        }) as unknown as Response;
      }
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({
          error: "",
          data: { success: true },
        }),
      }) as unknown as Response;
    });
  });

  it("fetches and displays user sessions", async () => {
    // Unmock the Home component to test the actual component
    vi.doMock("../app/home/page", async () => {
      const actual = await import("../app/home/page");
      return { ...actual };
    });
    
    // Re-import the Home component
    const { default: ActualHome } = await import("../app/home/page");
    
    render(<ActualHome />);

    // Wait for sessions to load
    await waitFor(() => {
      expect(screen.queryByText(/Loading sessions.../i)).not.toBeInTheDocument();
    });

    // Verify fetch was called correctly
    expect(global.fetch).toHaveBeenCalledWith(
      "http://localhost:8080/sessions",
      expect.objectContaining({
        method: "GET",
        headers: expect.objectContaining({
          Authorization: "Bearer mock-token",
        }),
      })
    );

    // Check if sessions are displayed
    expect(screen.getByText("Session 1")).toBeInTheDocument();
    expect(screen.getByText("Session 2")).toBeInTheDocument();
    
    // Check active status
    expect(screen.getByText("Active")).toBeInTheDocument();
    expect(screen.getByText("Completed")).toBeInTheDocument();
    
    // Check duration
    expect(screen.getByText(/Duration: 2 hours/i)).toBeInTheDocument();
  });

  it("displays empty state when no sessions exist", async () => {
    // Mock fetch to return empty sessions
    global.fetch = vi.fn().mockImplementation(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve({
          error: "",
          data: { sessions: [] },
        }),
      }) as unknown as Response
    );

    // Unmock the Home component to test the actual component
    vi.doMock("../app/home/page", async () => {
      const actual = await import("../app/home/page");
      return { ...actual };
    });
    
    // Re-import the Home component
    const { default: ActualHome } = await import("../app/home/page");
    
    render(<ActualHome />);

    // Wait for sessions to load
    await waitFor(() => {
      expect(screen.queryByText(/Loading sessions.../i)).not.toBeInTheDocument();
    });

    // Check for empty state message
    expect(screen.getByText("No sessions found")).toBeInTheDocument();
    expect(
      screen.getByText(/You haven't created any sessions yet./i)
    ).toBeInTheDocument();
    expect(screen.getByText("Create Your First Session")).toBeInTheDocument();
  });

  it("handles error when fetching sessions", async () => {
    // Mock fetch to return an error
    global.fetch = vi.fn().mockImplementation(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve({
          error: "Failed to fetch sessions",
          data: null,
        }),
      }) as unknown as Response
    );

    // Unmock the Home component to test the actual component
    vi.doMock("../app/home/page", async () => {
      const actual = await import("../app/home/page");
      return { ...actual };
    });
    
    // Re-import the Home component
    const { default: ActualHome } = await import("../app/home/page");
    
    render(<ActualHome />);

    // Wait for error to display
    await waitFor(() => {
      expect(screen.getByText("Failed to fetch sessions")).toBeInTheDocument();
    });
  });

  it("creates a new session successfully", async () => {
    // Mock fetch to handle different API calls
    let fetchCount = 0;
    global.fetch = vi.fn().mockImplementation((url, options) => {
      fetchCount++;
      
      // Initial sessions fetch
      if (fetchCount === 1) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({
            error: "",
            data: { sessions: mockSessions },
          }),
        }) as unknown as Response;
      }
      // Session creation
      else if (fetchCount === 2 && options?.method === "POST") {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({
            error: "",
            data: {
              session: {
                id: "3",
                user_id: "user1",
                name: "Session 3",
                started_at: "2025-05-04T10:00:00Z",
                stopped_at: null,
                active: true,
                duration: null,
              },
            },
          }),
        }) as unknown as Response;
      }
      // Sessions refresh after creation
      else {
        const updatedSessions = [...mockSessions, {
          id: "3",
          user_id: "user1",
          name: "Session 3",
          started_at: "2025-05-04T10:00:00Z",
          stopped_at: null,
          active: true,
          duration: null,
        }];
        
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({
            error: "",
            data: { sessions: updatedSessions },
          }),
        }) as unknown as Response;
      }
    });

    // Unmock the Home component to test the actual component
    vi.doMock("../app/home/page", async () => {
      const actual = await import("../app/home/page");
      return { ...actual };
    });
    
    // Re-import the Home component
    const { default: ActualHome } = await import("../app/home/page");
    
    render(<ActualHome />);

    // Wait for sessions to load
    await waitFor(() => {
      expect(screen.queryByText(/Loading sessions.../i)).not.toBeInTheDocument();
    });

    // Click the "New Session" button
    const newSessionButton = screen.getByText("New Session");
    fireEvent.click(newSessionButton);

    // Verify session creation API was called
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        "http://localhost:8080/sessions",
        expect.objectContaining({
          method: "POST",
          headers: expect.objectContaining({
            Authorization: "Bearer mock-token",
          }),
        })
      );
    });
  });

  it("stops an active session", async () => {
    // Unmock the Home component to test the actual component
    vi.doMock("../app/home/page", async () => {
      const actual = await import("../app/home/page");
      return { ...actual };
    });
    
    // Re-import the Home component
    const { default: ActualHome } = await import("../app/home/page");
    
    render(<ActualHome />);

    // Wait for sessions to load
    await waitFor(() => {
      expect(screen.queryByText(/Loading sessions.../i)).not.toBeInTheDocument();
    });

    // Find and click the stop button for the active session
    const stopButtons = screen.getAllByText("Stop");
    fireEvent.click(stopButtons[0]);

    // Verify stop session API was called
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        "http://localhost:8080/sessions/1/stop",
        expect.objectContaining({
          method: "POST",
          headers: expect.objectContaining({
            Authorization: "Bearer mock-token",
          }),
        })
      );
    });
  });

  it("deletes a session", async () => {
    // Unmock the Home component to test the actual component
    vi.doMock("../app/home/page", async () => {
      const actual = await import("../app/home/page");
      return { ...actual };
    });
    
    // Re-import the Home component
    const { default: ActualHome } = await import("../app/home/page");
    
    render(<ActualHome />);

    // Wait for sessions to load
    await waitFor(() => {
      expect(screen.queryByText(/Loading sessions.../i)).not.toBeInTheDocument();
    });

    // Find and click the delete button for the first session
    const deleteButtons = screen.getAllByTestId("alert-dialog-trigger");
    fireEvent.click(deleteButtons[0]);

    // Find the dialog content and get the delete button within it
    const dialogContent = screen.getAllByTestId("alert-dialog-content")[0];
    const confirmDeleteButton = within(dialogContent).getByTestId("alert-dialog-action");
    fireEvent.click(confirmDeleteButton);

    // Verify delete session API was called
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining("http://localhost:8080/sessions/"),
        expect.objectContaining({
          method: "DELETE",
          headers: expect.objectContaining({
            Authorization: "Bearer mock-token",
          }),
        })
      );
    });
  });

  it("redirects to login page when user is not authenticated", async () => {
    // Mock useUser to return no user
    const mockUseUser = vi.mocked(useUser);
    mockUseUser.mockReturnValue({
      user: null,
      loading: false,
      login: vi.fn().mockResolvedValue(undefined),
      register: vi.fn().mockResolvedValue(undefined),
      logout: vi.fn(),
      clearError: vi.fn(),
      getAuthHeader: () => ({}),
    });

    // Unmock the Home component to test the actual component
    vi.doMock("../app/home/page", async () => {
      const actual = await import("../app/home/page");
      return { ...actual };
    });
    
    // Re-import the Home component
    const { default: ActualHome } = await import("../app/home/page");
    
    render(<ActualHome />);

    // Verify redirect was called
    await waitFor(() => {
      expect(mockRouter.push).toHaveBeenCalledWith("/login");
    });
  });
});
