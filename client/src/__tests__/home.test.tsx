import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import Home from "../app/page";

// Mock Next.js Link component since we're testing outside of Next.js
vi.mock("next/link", () => ({
  default: ({
    href,
    children,
  }: {
    href: string;
    children: React.ReactNode;
  }) => (
    <a href={href} data-testid="link">
      {children}
    </a>
  ),
}));

describe("Home Page", () => {
  it("renders the home page correctly", () => {
    render(<Home />);

    // Check if main elements are present
    expect(screen.getByText("Orchestrator")).toBeInTheDocument();

    // Check for login and register links
    const loginButton = screen.getByText("Login");
    expect(loginButton).toBeInTheDocument();
    expect(loginButton.closest('[data-testid="link"]')).toHaveAttribute(
      "href",
      "/login",
    );

    const registerButton = screen.getByText("Register");
    expect(registerButton).toBeInTheDocument();
    expect(registerButton.closest('[data-testid="link"]')).toHaveAttribute(
      "href",
      "/register",
    );

    // Check for copyright notice with current year
    const currentYear = new Date().getFullYear();
    expect(
      screen.getByText(`© ${currentYear} Orchestrator`),
    ).toBeInTheDocument();
  });
});
