"use client";

import {
  createContext,
  useContext,
  useState,
  useEffect,
  ReactNode,
} from "react";
import Cookies from "js-cookie";

interface User {
  id: string;
  email: string;
  firstName: string;
  lastName: string;
}

interface UserContextType {
  user: User | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (
    email: string,
    password: string,
    firstName: string,
    lastName: string,
  ) => Promise<void>;
  logout: () => void;
  clearError: () => void;
  getAuthHeader: () => { Authorization: string } | {};
}

// Interface for our API response format
interface APIResponse<T> {
  error: string;
  data: T | null;
}

interface AuthResponseData {
  token: string;
  user: {
    id: string;
    email: string;
    first_name: string;
    last_name: string;
  };
}

const UserContext = createContext<UserContextType | undefined>(undefined);

// API URL base
const API_URL = "http://localhost:8080";

// Helper function to save token in both localStorage and cookie
const saveToken = (token: string) => {
  localStorage.setItem("token", token);
  Cookies.set("token", token, { expires: 7, path: "/" }); // Set cookie to expire in 7 days
};

// Helper function to remove token from both localStorage and cookie
const removeToken = () => {
  localStorage.removeItem("token");
  Cookies.remove("token", { path: "/" });
};

export function UserProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState<boolean>(true);

  useEffect(() => {
    // Check if user is already logged in
    const storedUser = localStorage.getItem("user");
    if (storedUser) {
      try {
        setUser(JSON.parse(storedUser));
      } catch (e) {
        console.error("Failed to parse stored user data");
      }
    }
    setLoading(false);
  }, []);

  const clearError = () => {
    // This is now just a utility function to maintain API compatibility
  };

  const getAuthHeader = () => {
    const token = localStorage.getItem("token");
    return token ? { Authorization: `Bearer ${token}` } : {};
  };

  const login = async (email: string, password: string): Promise<void> => {
    try {
      const response = await fetch(`${API_URL}/login`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ email, password }),
      });

      const apiResponse: APIResponse<AuthResponseData> = await response.json();

      // Check if there's an error in the response
      if (apiResponse.error) {
        throw new Error(apiResponse.error);
      }

      // Make sure we have data
      if (!apiResponse.data) {
        throw new Error("No data received from server");
      }

      const userData = {
        id: apiResponse.data.user.id,
        email: apiResponse.data.user.email,
        firstName: apiResponse.data.user.first_name,
        lastName: apiResponse.data.user.last_name,
      };

      setUser(userData);
      saveToken(apiResponse.data.token);
      localStorage.setItem("user", JSON.stringify(userData));
    } catch (err) {
      throw err;
    }
  };

  const register = async (
    email: string,
    password: string,
    firstName: string,
    lastName: string,
  ): Promise<void> => {
    try {
      const response = await fetch(`${API_URL}/register`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          email,
          password,
          first_name: firstName,
          last_name: lastName,
        }),
      });

      const apiResponse: APIResponse<AuthResponseData> = await response.json();

      // Check if there's an error in the response
      if (apiResponse.error) {
        throw new Error(apiResponse.error);
      }

      // Make sure we have data
      if (!apiResponse.data) {
        throw new Error("No data received from server");
      }

      const userData = {
        id: apiResponse.data.user.id,
        email: apiResponse.data.user.email,
        firstName: apiResponse.data.user.first_name,
        lastName: apiResponse.data.user.last_name,
      };

      setUser(userData);
      saveToken(apiResponse.data.token);
      localStorage.setItem("user", JSON.stringify(userData));
    } catch (err) {
      throw err;
    }
  };

  const logout = () => {
    setUser(null);
    removeToken();
    localStorage.removeItem("user");
  };

  return (
    <UserContext.Provider
      value={{
        user,
        loading,
        login,
        register,
        logout,
        clearError,
        getAuthHeader,
      }}
    >
      {children}
    </UserContext.Provider>
  );
}

export function useUser() {
  const context = useContext(UserContext);
  if (context === undefined) {
    throw new Error("useUser must be used within a UserProvider");
  }
  return context;
}
