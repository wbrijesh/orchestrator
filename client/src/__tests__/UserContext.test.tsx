import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, act } from "@testing-library/react";
import { UserProvider, useUser } from "../context/UserContext";
import Cookies from "js-cookie";
import React from "react";

// Mock fetch API
global.fetch = vi.fn();

// Mock js-cookie
vi.mock("js-cookie", () => ({
  default: {
    set: vi.fn(),
    get: vi.fn(),
    remove: vi.fn(),
  },
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

// Create a test component that uses the UserContext
const TestComponent = () => {
  const { user, login, register, logout } = useUser();
  return (
    <div>
      {user ? (
        <>
          <div data-testid="user-info">
            <p>Email: {user.email}</p>
            <p>
              Name: {user.firstName} {user.lastName}
            </p>
          </div>
          <button onClick={logout}>Logout</button>
        </>
      ) : (
        <>
          <button onClick={() => login("test@example.com", "password")}>
            Login
          </button>
          <button
            onClick={() =>
              register("test@example.com", "password", "Test", "User")
            }
          >
            Register
          </button>
        </>
      )}
    </div>
  );
};

describe("UserContext", () => {
  beforeEach(() => {
    // Reset mocks between tests
    vi.clearAllMocks();
    localStorageMock.clear();

    // Reset fetch mock
    vi.mocked(global.fetch).mockReset();
  });

  it("provides user context with initial null user state", () => {
    render(
      <UserProvider>
        <TestComponent />
      </UserProvider>,
    );

    // When first rendered, user should be null, so login and register buttons should be present
    expect(screen.getByText("Login")).toBeInTheDocument();
    expect(screen.getByText("Register")).toBeInTheDocument();
    expect(screen.queryByTestId("user-info")).not.toBeInTheDocument();
  });

  it("loads user from localStorage on initialization if available", async () => {
    // Setup localStorage with user data
    const userData = {
      id: "123",
      email: "saved@example.com",
      firstName: "Saved",
      lastName: "User",
    };

    localStorageMock.getItem.mockReturnValueOnce(JSON.stringify(userData));

    // Render the component
    render(
      <UserProvider>
        <TestComponent />
      </UserProvider>,
    );

    // User data should be loaded from localStorage
    await waitFor(() => {
      expect(screen.getByText("Email: saved@example.com")).toBeInTheDocument();
      expect(screen.getByText("Name: Saved User")).toBeInTheDocument();
    });
  });

  it("handles login successfully", async () => {
    // Mock successful login response
    vi.mocked(global.fetch).mockResolvedValueOnce({
      json: () =>
        Promise.resolve({
          error: "",
          data: {
            token: "fake-token",
            user: {
              id: "123",
              email: "test@example.com",
              first_name: "Test",
              last_name: "User",
            },
          },
        }),
    } as Response);

    render(
      <UserProvider>
        <TestComponent />
      </UserProvider>,
    );

    // Click login button
    await act(async () => {
      screen.getByText("Login").click();
    });

    // Verify API was called with correct data
    expect(global.fetch).toHaveBeenCalledWith("http://localhost:8080/login", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        email: "test@example.com",
        password: "password",
      }),
    });

    // Verify user data was saved and displayed
    await waitFor(() => {
      expect(screen.getByText("Email: test@example.com")).toBeInTheDocument();
      expect(screen.getByText("Name: Test User")).toBeInTheDocument();

      // Check token and user data were saved to storage
      expect(Cookies.set).toHaveBeenCalledWith(
        "token",
        "fake-token",
        expect.any(Object),
      );
      expect(localStorageMock.setItem).toHaveBeenCalledWith(
        "token",
        "fake-token",
      );
      expect(localStorageMock.setItem).toHaveBeenCalledWith(
        "user",
        expect.stringContaining("test@example.com"),
      );
    });
  });

  it("handles registration successfully", async () => {
    // Mock successful register response
    vi.mocked(global.fetch).mockResolvedValueOnce({
      json: () =>
        Promise.resolve({
          error: "",
          data: {
            token: "fake-token",
            user: {
              id: "456",
              email: "test@example.com",
              first_name: "Test",
              last_name: "User",
            },
          },
        }),
    } as Response);

    render(
      <UserProvider>
        <TestComponent />
      </UserProvider>,
    );

    // Click register button
    await act(async () => {
      screen.getByText("Register").click();
    });

    // Verify API was called with correct data
    expect(global.fetch).toHaveBeenCalledWith(
      "http://localhost:8080/register",
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          email: "test@example.com",
          password: "password",
          first_name: "Test",
          last_name: "User",
        }),
      },
    );

    // Verify user data was saved and displayed
    await waitFor(() => {
      expect(screen.getByText("Email: test@example.com")).toBeInTheDocument();
      expect(screen.getByText("Name: Test User")).toBeInTheDocument();
    });
  });

  it("handles logout correctly", async () => {
    // Setup initial state with logged in user
    localStorageMock.getItem.mockReturnValueOnce(
      JSON.stringify({
        id: "123",
        email: "test@example.com",
        firstName: "Test",
        lastName: "User",
      }),
    );

    render(
      <UserProvider>
        <TestComponent />
      </UserProvider>,
    );

    // Verify user is initially logged in
    await waitFor(() => {
      expect(screen.getByText("Email: test@example.com")).toBeInTheDocument();
    });

    // Click logout button
    await act(async () => {
      screen.getByText("Logout").click();
    });

    // Verify user is logged out
    await waitFor(() => {
      expect(screen.getByText("Login")).toBeInTheDocument();
      expect(screen.getByText("Register")).toBeInTheDocument();
      expect(screen.queryByTestId("user-info")).not.toBeInTheDocument();

      // Check storage items were removed
      expect(Cookies.remove).toHaveBeenCalledWith("token", expect.any(Object));
      expect(localStorageMock.removeItem).toHaveBeenCalledWith("token");
      expect(localStorageMock.removeItem).toHaveBeenCalledWith("user");
    });
  });

  it("handles login API error", async () => {
    // Mock error response
    const errorMessage = "Invalid credentials";
    vi.mocked(global.fetch).mockResolvedValueOnce({
      json: () =>
        Promise.resolve({
          error: errorMessage,
          data: null,
        }),
    } as Response);

    // Mock console.error to prevent error output during test
    const consoleErrorSpy = vi
      .spyOn(console, "error")
      .mockImplementation(() => {});

    // Render with a custom component that captures errors
    const LoginWithErrorHandling = () => {
      const { login } = useUser();
      const [error, setError] = React.useState<string | null>(null);

      const handleLogin = async () => {
        try {
          await login("test@example.com", "wrong-password");
        } catch (err) {
          if (err instanceof Error) {
            setError(err.message);
          }
        }
      };

      return (
        <div>
          <button onClick={handleLogin}>Login</button>
          {error && <p data-testid="error-message">{error}</p>}
        </div>
      );
    };

    render(
      <UserProvider>
        <LoginWithErrorHandling />
      </UserProvider>,
    );

    // Click login button
    await act(async () => {
      screen.getByText("Login").click();
    });

    // Verify error is displayed
    await waitFor(() => {
      expect(screen.getByTestId("error-message").textContent).toBe(
        errorMessage,
      );
    });

    consoleErrorSpy.mockRestore();
  });
});
