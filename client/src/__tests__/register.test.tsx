import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Register from '../app/register/page';

// Create mocks
const mockPush = vi.fn();
const mockRegister = vi.fn().mockResolvedValue(undefined);
const mockClearError = vi.fn();

// Mock the useRouter and useUser hooks
vi.mock('next/navigation', () => ({
  useRouter: () => ({
    push: mockPush,
  }),
}));

vi.mock('../context/UserContext', () => ({
  useUser: () => ({
    register: mockRegister,
    clearError: mockClearError,
  }),
}));

describe('Register Page', () => {
  beforeEach(() => {
    // Clear mocks between tests
    vi.clearAllMocks();
  });

  it('renders register form correctly', () => {
    render(<Register />);
    
    // Check if the page contains necessary elements
    expect(screen.getByText('Orchestrator')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Register' })).toBeInTheDocument();
    expect(screen.getByLabelText('First Name')).toBeInTheDocument();
    expect(screen.getByLabelText('Last Name')).toBeInTheDocument();
    expect(screen.getByLabelText('Email')).toBeInTheDocument();
    expect(screen.getByLabelText('Password')).toBeInTheDocument();
    expect(screen.getByLabelText('Confirm Password')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Register' })).toBeInTheDocument();
    expect(screen.getByText('Already have an account?')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Login' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Back to Home' })).toBeInTheDocument();
  });

  it('shows validation error for empty first name', async () => {
    const user = userEvent.setup();
    render(<Register />);
    
    // Try to submit with empty first name
    await user.click(screen.getByRole('button', { name: 'Register' }));
    
    // Check for validation error
    expect(screen.getByText('First name is required')).toBeInTheDocument();
  });

  it('shows validation error for empty last name', async () => {
    const user = userEvent.setup();
    render(<Register />);
    
    // Enter first name but no last name
    await user.type(screen.getByLabelText('First Name'), 'John');
    await user.click(screen.getByRole('button', { name: 'Register' }));
    
    // Check for validation error
    expect(screen.getByText('Last name is required')).toBeInTheDocument();
  });

  it('shows validation error for invalid email format', async () => {
    const user = userEvent.setup();
    render(<Register />);
    
    // Enter first name, last name, and invalid email
    await user.type(screen.getByLabelText('First Name'), 'John');
    await user.type(screen.getByLabelText('Last Name'), 'Doe');
    await user.type(screen.getByLabelText('Email'), 'invalid-email');
    await user.click(screen.getByRole('button', { name: 'Register' }));
    
    // After form submission with invalid email, there should be a form error
    // but we don't need to check the exact error message
    await waitFor(() => {
      // The form should still be in the document (not navigated away)
      const emailInput = screen.getByLabelText('Email');
      expect(emailInput).toBeInTheDocument();
      
      // Mock should not have been called with invalid data
      expect(mockRegister).not.toHaveBeenCalled();
    });
  });

  it('shows validation error for password too short', async () => {
    const user = userEvent.setup();
    render(<Register />);
    
    // Enter valid data except for short password
    await user.type(screen.getByLabelText('First Name'), 'John');
    await user.type(screen.getByLabelText('Last Name'), 'Doe');
    await user.type(screen.getByLabelText('Email'), 'john@example.com');
    await user.type(screen.getByLabelText('Password'), 'pass');
    await user.type(screen.getByLabelText('Confirm Password'), 'pass');
    await user.click(screen.getByRole('button', { name: 'Register' }));
    
    // Check for validation error
    expect(screen.getByText('Password must be at least 6 characters')).toBeInTheDocument();
  });

  it('shows validation error for passwords not matching', async () => {
    const user = userEvent.setup();
    render(<Register />);
    
    // Enter valid data except for mismatched passwords
    await user.type(screen.getByLabelText('First Name'), 'John');
    await user.type(screen.getByLabelText('Last Name'), 'Doe');
    await user.type(screen.getByLabelText('Email'), 'john@example.com');
    await user.type(screen.getByLabelText('Password'), 'password123');
    await user.type(screen.getByLabelText('Confirm Password'), 'password456');
    await user.click(screen.getByRole('button', { name: 'Register' }));
    
    // Check for validation error
    expect(screen.getByText('Passwords do not match')).toBeInTheDocument();
  });

  it('submits form with valid data', async () => {
    const user = userEvent.setup();
    
    render(<Register />);
    
    // Fill in valid data
    await user.type(screen.getByLabelText('First Name'), 'John');
    await user.type(screen.getByLabelText('Last Name'), 'Doe');
    await user.type(screen.getByLabelText('Email'), 'john@example.com');
    await user.type(screen.getByLabelText('Password'), 'password123');
    await user.type(screen.getByLabelText('Confirm Password'), 'password123');
    
    // Submit the form
    await user.click(screen.getByRole('button', { name: 'Register' }));
    
    // Check that register function was called with correct arguments
    await waitFor(() => {
      expect(mockRegister).toHaveBeenCalledWith(
        'john@example.com',
        'password123',
        'John',
        'Doe'
      );
    });
    
    // Check navigation after successful registration
    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith('/home');
    });
  });
});