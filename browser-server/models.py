from typing import Dict, List, Optional
from pydantic import BaseModel, Field
from datetime import datetime
from config import settings

class ViewportSize(BaseModel):
    width: int = Field(settings.DEFAULT_VIEW_WIDTH, description="Viewport width in pixels")
    height: int = Field(settings.DEFAULT_VIEW_HEIGHT, description="Viewport height in pixels")

class SessionCreateRequest(BaseModel):
    browser_type: str = Field(settings.DEFAULT_BROWSER, description="Browser type (chromium, firefox, webkit)")
    headless: bool = Field(settings.DEFAULT_HEADLESS, description="Run browser in headless mode")
    viewport_size: Optional[ViewportSize] = None
    user_agent: Optional[str] = None
    timeout: Optional[int] = Field(settings.DEFAULT_SESSION_TIMEOUT, description="Session timeout in seconds")

class SessionResponse(BaseModel):
    id: str = Field(..., description="Unique session identifier")
    browser_type: str = Field(..., description="Browser type used for this session")
    headless: bool = Field(..., description="Whether the browser is running in headless mode")
    created_at: datetime = Field(..., description="Time when the session was created")
    expires_at: datetime = Field(..., description="Time when the session will expire")
    cdp_url: str = Field(..., description="Chrome DevTools Protocol URL for this session")
    viewport_size: ViewportSize = Field(..., description="Browser viewport size")
    user_agent: Optional[str] = Field(None, description="User agent string if custom one is set")

class SessionListResponse(BaseModel):
    sessions: List[SessionResponse] = Field(..., description="List of active sessions")

class ErrorResponse(BaseModel):
    detail: str = Field(..., description="Error details")