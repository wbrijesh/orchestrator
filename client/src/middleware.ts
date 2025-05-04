import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

// This middleware handles authentication redirects
export function middleware(request: NextRequest) {
  // Get the path
  const path = request.nextUrl.pathname;
  
  // Define public paths that don't require authentication checks
  const isAuthPath = path === '/login' || path === '/register';
  
  // Check if user is authenticated by looking for the token in cookies
  const token = request.cookies.get('token')?.value;
  
  // If the user is already logged in and trying to access login or register page,
  // redirect them to the home page
  if (isAuthPath && token) {
    return NextResponse.redirect(new URL('/home', request.url));
  }
  
  // Continue with the request
  return NextResponse.next();
}

// Configure which paths this middleware should run on
export const config = {
  matcher: ['/login', '/register'],
};