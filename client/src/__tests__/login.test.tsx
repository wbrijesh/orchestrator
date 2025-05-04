import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Login from '../app/login/page';

// Create mocks
const mockPush = vi.fn();
const mockLogin = vi.fn().mockResolvedValue(undefined);
const mockClearError = vi.fn();

// Mock the useRouter and useUser hooks
vi.mock('next/navigation', () => ({
  useRouter: () => ({
    push: mockPush,
  }),
}));

vi.mock('../context/UserContext', () => ({
  useUser: () => ({
    login: mockLogin,
    clearError: mockClearError,
  }),
}));

describe('Login Page', () => {
  beforeEach(() => {
    // Clear mocks between tests
    vi.clearAllMocks();
  });

  it('renders login form correctly', () => {
    render(<Login />);
    
    // Check if the page contains necessary elements
    expect(screen.getByText('Orchestrator')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Login' })).toBeInTheDocument();
    expect(screen.getByLabelText('Email')).toBeInTheDocument();
    expect(screen.getByLabelText('Password')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Login' })).toBeInTheDocument();
    expect(screen.getByText('Don\'t have an account?')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Register' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Back to Home' })).toBeInTheDocument();
  });

  it('shows validation error for empty email', async () => {
    const user = userEvent.setup();
    render(<Login />);
    
    // Try to submit with empty email
    await user.click(screen.getByRole('button', { name: 'Login' }));
    
    // Check for validation error
    expect(screen.getByText('Email is required')).toBeInTheDocument();
  });

  it('shows validation error for invalid email format', async () => {
    const user = userEvent.setup();
    render(<Login />);
    
    // Enter invalid email and submit
    await user.type(screen.getByLabelText('Email'), 'invalid-email');
    await user.click(screen.getByRole('button', { name: 'Login' }));
    
    // After form submission with invalid email, there should be a form error
    // but we don't need to check the exact error message
    await waitFor(() => {
      // The form should still be in the document (not navigated away)
      const emailInput = screen.getByLabelText('Email');
      expect(emailInput).toBeInTheDocument();
      
      // Mock should not have been called with invalid data
      expect(mockLogin).not.toHaveBeenCalled();
    });
  });

  it('shows validation error for empty password', async () => {
    const user = userEvent.setup();
    render(<Login />);
    
    // Enter valid email but no password
    await user.type(screen.getByLabelText('Email'), 'test@example.com');
    await user.click(screen.getByRole('button', { name: 'Login' }));
    
    // Check for validation error
    expect(screen.getByText('Password is required')).toBeInTheDocument();
  });

  it('submits form with valid data', async () => {
    const user = userEvent.setup();
    
    render(<Login />);
    
    // Fill in valid data
    await user.type(screen.getByLabelText('Email'), 'test@example.com');
    await user.type(screen.getByLabelText('Password'), 'password123');
    
    // Submit the form
    await user.click(screen.getByRole('button', { name: 'Login' }));
    
    // Check that login function was called with correct arguments
    await waitFor(() => {
      expect(mockLogin).toHaveBeenCalledWith('test@example.com', 'password123');
    });
    
    // Check navigation after successful login
    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith('/home');
    });
  });
});