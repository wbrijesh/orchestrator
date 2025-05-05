import asyncio
import uuid
from typing import Dict, List, Optional, Any
from datetime import datetime, timedelta

from models import SessionResponse, ViewportSize
from browser_manager import BrowserManager
from config import settings


class SessionManager:
    """Manages browser sessions including creation, tracking, and cleanup"""
    
    def __init__(self, browser_manager: BrowserManager):
        self.browser_manager = browser_manager
        self.sessions: Dict[str, Dict[str, Any]] = {}
        self._lock = asyncio.Lock()
        self._cleanup_task = None
    
    async def start_cleanup_task(self):
        """Start the background task to clean up expired sessions"""
        self._cleanup_task = asyncio.create_task(self._cleanup_loop())
    
    async def _cleanup_loop(self):
        """Periodically check and remove expired sessions"""
        while True:
            await asyncio.sleep(settings.CLEANUP_INTERVAL)
            await self._cleanup_expired_sessions()
    
    async def _cleanup_expired_sessions(self):
        """Remove sessions that have expired"""
        now = datetime.now()
        to_remove = []
        
        # Find expired sessions
        async with self._lock:
            for session_id, session in self.sessions.items():
                if session["expires_at"] < now:
                    to_remove.append(session_id)
        
        # Remove them one by one
        for session_id in to_remove:
            await self.delete_session(session_id)
    
    async def create_session(
        self,
        browser_type: str = settings.DEFAULT_BROWSER,
        headless: bool = settings.DEFAULT_HEADLESS,
        viewport_size: Optional[ViewportSize] = None,
        user_agent: Optional[str] = None,
        timeout: int = settings.DEFAULT_SESSION_TIMEOUT
    ) -> SessionResponse:
        """Create a new browser session and return its details"""
        # Check if we've reached the maximum number of sessions
        async with self._lock:
            if len(self.sessions) >= settings.MAX_SESSIONS:
                raise RuntimeError(f"Maximum number of sessions reached ({settings.MAX_SESSIONS})")
        
        # Create viewport size if not provided
        if viewport_size is None:
            viewport_size = ViewportSize(
                width=settings.DEFAULT_VIEW_WIDTH,
                height=settings.DEFAULT_VIEW_HEIGHT
            )
        
        # Create browser instance
        browser_data, cdp_url = await self.browser_manager.create_browser_instance(
            browser_type=browser_type,
            headless=headless,
            viewport_size={"width": viewport_size.width, "height": viewport_size.height},
            user_agent=user_agent
        )
        
        # Generate unique session ID
        session_id = str(uuid.uuid4())
        
        # Calculate expiration time
        created_at = datetime.now()
        expires_at = created_at + timedelta(seconds=timeout)
        
        # Create session record
        session = {
            "id": session_id,
            "browser_id": browser_data["id"],
            "browser_type": browser_type,
            "headless": headless,
            "created_at": created_at,
            "expires_at": expires_at,
            "cdp_url": cdp_url,
            "viewport_size": viewport_size,
            "user_agent": user_agent
        }
        
        # Store session
        async with self._lock:
            self.sessions[session_id] = session
        
        # Return session details
        return SessionResponse(**session)
    
    async def get_session(self, session_id: str) -> Optional[SessionResponse]:
        """Get details for a specific session"""
        async with self._lock:
            session = self.sessions.get(session_id)
            
        if session:
            return SessionResponse(**session)
        return None
    
    async def list_sessions(self) -> List[SessionResponse]:
        """List all active sessions"""
        session_list = []
        
        async with self._lock:
            for session in self.sessions.values():
                session_list.append(SessionResponse(**session))
                
        return session_list
    
    async def delete_session(self, session_id: str) -> bool:
        """Terminate and remove a session"""
        # Get session info
        async with self._lock:
            session = self.sessions.pop(session_id, None)
        
        if not session:
            return False
        
        # Close the browser
        browser_id = session["browser_id"]
        await self.browser_manager.close_browser(browser_id)
        
        return True