import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Account from '../app/account/page';

// Mock user data
const mockUser = {
  id: '123',
  email: 'test@example.com',
  firstName: 'Test',
  lastName: 'User'
};

// Mock hooks
const mockPush = vi.fn();
const mockLogout = vi.fn();
let mockUserState: { user: typeof mockUser | null, loading: boolean };

vi.mock('next/navigation', () => ({
  useRouter: () => ({
    push: mockPush,
  }),
}));

vi.mock('../context/UserContext', () => ({
  useUser: () => ({
    user: mockUserState.user,
    loading: mockUserState.loading,
    logout: mockLogout,
  }),
}));

// Mock Next.js Link component
vi.mock('next/link', () => ({
  default: ({ href, children }: { href: string, children: React.ReactNode }) => (
    <a href={href} data-testid="link">
      {children}
    </a>
  ),
}));

describe('Account Page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUserState = { user: null, loading: false };
  });
  
  it('redirects to login when user is not logged in', () => {
    mockUserState = { user: null, loading: false };
    
    render(<Account />);
    
    // Check that it redirects to login
    expect(mockPush).toHaveBeenCalledWith('/login');
  });
  
  it('shows loading state while checking authentication', () => {
    mockUserState = { user: null, loading: true };
    
    render(<Account />);
    
    // Should show loading message
    expect(screen.getByText('Loading...')).toBeInTheDocument();
    expect(mockPush).not.toHaveBeenCalled();
  });
  
  it('displays user profile when logged in', () => {
    mockUserState = { user: mockUser, loading: false };
    
    render(<Account />);
    
    // Should display user information
    expect(screen.getByText('Profile')).toBeInTheDocument();
    expect(screen.getByText('Test User')).toBeInTheDocument();
    expect(screen.getByText('test@example.com')).toBeInTheDocument();
    expect(screen.getByText('123')).toBeInTheDocument();
  });
  
  it('handles logout correctly', async () => {
    mockUserState = { user: mockUser, loading: false };
    const user = userEvent.setup();
    
    render(<Account />);
    
    // Open the dropdown menu
    await user.click(screen.getByText('Account'));
    
    // Click logout
    const logoutButton = screen.getByText('Log out');
    await user.click(logoutButton);
    
    // Check that logout was called
    expect(mockLogout).toHaveBeenCalled();
    expect(mockPush).toHaveBeenCalledWith('/');
  });
});