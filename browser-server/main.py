import uvicorn
from fastapi import FastAPI, HTTPException
from fastapi.responses import JSONResponse

from browser_manager import BrowserManager
from session_manager import SessionManager
from models import (
    SessionCreateRequest,
    SessionResponse,
    SessionListResponse,
    ErrorResponse
)
from config import settings

app = FastAPI(
    title="Browser Session Service",
    description="Service to create and manage browser sessions with CDP access",
    version="0.1.0"
)

# Initialize managers
browser_manager = BrowserManager()
session_manager = SessionManager(browser_manager)


@app.post("/sessions", 
          response_model=SessionResponse, 
          responses={400: {"model": ErrorResponse}, 500: {"model": ErrorResponse}})
async def create_session(request: SessionCreateRequest):
    """Create a new browser session and return its details with CDP URL"""
    try:
        session = await session_manager.create_session(
            browser_type=request.browser_type,
            headless=request.headless,
            viewport_size=request.viewport_size,
            user_agent=request.user_agent,
            timeout=request.timeout
        )
        return session
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/sessions", response_model=SessionListResponse)
async def list_sessions():
    """List all active browser sessions"""
    sessions = await session_manager.list_sessions()
    return {"sessions": sessions}


@app.get("/sessions/{session_id}", 
         response_model=SessionResponse,
         responses={404: {"model": ErrorResponse}, 500: {"model": ErrorResponse}})
async def get_session(session_id: str):
    """Get details of a specific browser session"""
    session = await session_manager.get_session(session_id)
    if not session:
        raise HTTPException(status_code=404, detail=f"Session {session_id} not found")
    return session


@app.delete("/sessions/{session_id}", 
            response_model=dict,
            responses={404: {"model": ErrorResponse}, 500: {"model": ErrorResponse}})
async def delete_session(session_id: str):
    """Terminate and remove a specific browser session"""
    success = await session_manager.delete_session(session_id)
    if not success:
        raise HTTPException(status_code=404, detail=f"Session {session_id} not found")
    return {"status": "success", "message": f"Session {session_id} terminated"}


@app.on_event("startup")
async def startup():
    """Initialize the browser manager on startup"""
    await browser_manager.initialize()
    await session_manager.start_cleanup_task()


@app.on_event("shutdown")
async def shutdown():
    """Close all browser instances on shutdown"""
    await browser_manager.cleanup()


if __name__ == "__main__":
    uvicorn.run("main:app", host=settings.HOST, port=settings.PORT, reload=settings.DEBUG)