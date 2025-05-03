"use client";

import { useState, FormEvent, useRef } from "react";
import { useUser } from "@/context/UserContext";
import { useRouter } from "next/navigation";

export default function Login() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [formError, setFormError] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const formRef = useRef<HTMLFormElement>(null);

  const { login, clearError } = useUser();
  const router = useRouter();

  // Client-side validation function
  const validateForm = () => {
    // Reset previous errors
    setFormError("");

    // Check required fields
    if (!email.trim()) {
      setFormError("Email is required");
      return false;
    }

    if (!password) {
      setFormError("Password is required");
      return false;
    }

    // Simple email validation
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(email)) {
      setFormError("Please enter a valid email address");
      return false;
    }

    return true;
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    setIsSubmitting(true);

    try {
      await login(email, password);
      router.push("/account");
    } catch (err) {
      if (err instanceof Error) {
        setFormError(err.message);
      } else {
        setFormError("An unexpected error occurred");
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  // Custom function to handle navigation with error clearing
  const handleNavigation = (path: string) => {
    clearError();
    router.push(path);
  };

  return (
    <div className="h-screen flex flex-col">
      <div className="flex flex-col items-center justify-center flex-grow px-5">
        <h1 className="text-xl font-medium text-center mb-8">Orchestrator</h1>

        <div className="flex flex-col items-center max-w-xs w-full">
          <h1 className="text-lg font-medium mb-4 text-neutral-800">Login</h1>

          {formError && <div className="text-sm text-red-600">{formError}</div>}

          <form
            ref={formRef}
            onSubmit={handleSubmit}
            className="flex flex-col gap-3 w-full"
          >
            <div>
              <label
                htmlFor="email"
                className="block text-sm text-neutral-600 mb-0.5"
              >
                Email
              </label>
              <input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="w-full px-2 py-1.5 text-sm border border-neutral-300 rounded-md focus:outline-none focus:border-sky-500 focus:ring-1 focus:ring-sky-500"
                disabled={isSubmitting}
              />
            </div>

            <div>
              <label
                htmlFor="password"
                className="block text-sm text-neutral-600 mb-0.5"
              >
                Password
              </label>
              <input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full px-2 py-1 text-sm border border-neutral-300 rounded-md focus:outline-none focus:border-sky-500 focus:ring-1 focus:ring-sky-500"
                disabled={isSubmitting}
              />
            </div>

            <button
              type="submit"
              className="px-2 py-1.5 text-sm bg-sky-600 text-white text-center rounded-md hover:bg-sky-700 transition-colors disabled:bg-sky-400 mt-2"
              disabled={isSubmitting}
            >
              {isSubmitting ? "Logging in..." : "Login"}
            </button>
          </form>

          <div className="text-sm text-neutral-500 mt-4 text-center">
            <p>
              Don&apos;t have an account?{" "}
              <button
                onClick={() => handleNavigation("/register")}
                className="text-sky-600 hover:underline"
              >
                Register
              </button>
            </p>
            <button
              onClick={() => handleNavigation("/")}
              className="text-sm text-neutral-400 hover:underline mt-1"
            >
              Back to Home
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
