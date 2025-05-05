#!/usr/bin/env python3
"""
Entry point script to start the browser session service
"""
import uvicorn
from config import settings

if __name__ == "__main__":
    print(f"Starting Browser Session Service on {settings.HOST}:{settings.PORT}")
    print("Initialization will happen during application startup")
    
    # Start the server - FastAPI will handle initialization through the startup event
    uvicorn.run(
        "main:app", 
        host=settings.HOST, 
        port=settings.PORT, 
        reload=settings.DEBUG
    )