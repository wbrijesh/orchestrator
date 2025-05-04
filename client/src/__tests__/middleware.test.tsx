import { describe, it, expect, vi, beforeEach } from 'vitest';
import { middleware } from '../middleware';
import { NextRequest, NextResponse } from 'next/server';

// Mock the NextRequest and NextResponse
vi.mock('next/server', () => {
  return {
    NextRequest: vi.fn().mockImplementation((url) => ({
      nextUrl: { pathname: url },
      url: 'http://localhost' + url,
      cookies: {
        get: vi.fn(),
      },
    })),
    NextResponse: {
      next: vi.fn().mockReturnValue({ type: 'next' }),
      redirect: vi.fn().mockReturnValue({
        type: 'redirect',
        url: 'http://localhost/account',
      }),
    },
  };
});

describe('Auth Middleware', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('redirects authenticated users from login page to account page', () => {
    // Create mock request for login page with valid URL
    const request = new NextRequest('/login') as unknown as NextRequest;
    
    // Mock the cookie to return a token (authenticated)
    request.cookies.get = vi.fn().mockReturnValue({ value: 'sample-token' });
    
    // Call the middleware
    const response = middleware(request);
    
    // Expect redirect to have been called
    expect(NextResponse.redirect).toHaveBeenCalled();
    
    // Check the response has expected properties
    expect(response).toBeDefined();
    expect(response).toHaveProperty('type', 'redirect');
  });

  it('redirects authenticated users from register page to account page', () => {
    // Create mock request for register page with valid URL
    const request = new NextRequest('/register') as unknown as NextRequest;
    
    // Mock the cookie to return a token (authenticated)
    request.cookies.get = vi.fn().mockReturnValue({ value: 'sample-token' });
    
    // Call the middleware
    const response = middleware(request);
    
    // Expect redirect to have been called
    expect(NextResponse.redirect).toHaveBeenCalled();
    
    // Check the response has expected properties
    expect(response).toBeDefined();
    expect(response).toHaveProperty('type', 'redirect');
  });

  it('allows unauthenticated users to access login page', () => {
    // Create mock request for login page
    const request = new NextRequest('/login') as unknown as NextRequest;
    
    // Mock the cookie to return null (unauthenticated)
    request.cookies.get = vi.fn().mockReturnValue(null);
    
    // Call the middleware
    const response = middleware(request);
    
    // Expect next() to be called (no redirect)
    expect(response).toHaveProperty('type', 'next');
    expect(NextResponse.next).toHaveBeenCalled();
    expect(NextResponse.redirect).not.toHaveBeenCalled();
  });

  it('allows unauthenticated users to access register page', () => {
    // Create mock request for register page
    const request = new NextRequest('/register') as unknown as NextRequest;
    
    // Mock the cookie to return null (unauthenticated)
    request.cookies.get = vi.fn().mockReturnValue(null);
    
    // Call the middleware
    const response = middleware(request);
    
    // Expect next() to be called (no redirect)
    expect(response).toHaveProperty('type', 'next');
    expect(NextResponse.next).toHaveBeenCalled();
    expect(NextResponse.redirect).not.toHaveBeenCalled();
  });
});