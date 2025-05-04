import { describe, it, expect, vi, afterEach } from "vitest";
import { render } from "@testing-library/react";
import React from "react";

// Mock CSS imports to prevent PostCSS processing issues
vi.mock("../app/globals.css", () => ({}));

// Mock metadata
vi.mock("next/navigation", () => ({}));

// Mock Inter font
vi.mock("next/font/google", () => ({
  Inter: () => ({
    className: "mock-inter-class",
  }),
}));

// Create spy on UserProvider
const mockUserProvider = vi.fn(({ children }) => (
  <div data-testid="user-provider-mock">{children}</div>
));

// Mock UserContext module
vi.mock("../context/UserContext", () => ({
  UserProvider: (props: { children: React.ReactNode }) =>
    mockUserProvider(props),
}));

// Import the component implementation
import RootLayout from "../app/layout";

describe("RootLayout", () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("renders with UserProvider wrapper", () => {
    const testChild = <div data-testid="test-child">Test Content</div>;

    const { getByTestId } = render(<RootLayout>{testChild}</RootLayout>);

    // Verify the UserProvider was called
    expect(mockUserProvider).toHaveBeenCalledTimes(1);

    // Verify both the provider and children are in the document
    expect(getByTestId("user-provider-mock")).toBeInTheDocument();
    expect(getByTestId("test-child")).toBeInTheDocument();
  });
});
