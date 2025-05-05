import os
from pydantic import BaseModel
from pydantic_settings import BaseSettings
from dotenv import load_dotenv

# Load environment variables from .env file
load_dotenv()

class Settings(BaseSettings):
    """Application settings loaded from environment variables"""
    
    # Server settings
    HOST: str = os.getenv("HOST", "0.0.0.0")
    PORT: int = int(os.getenv("PORT", "8000"))
    DEBUG: bool = os.getenv("DEBUG", "False").lower() in ('true', '1', 't')
    
    # Session settings
    DEFAULT_SESSION_TIMEOUT: int = int(os.getenv("DEFAULT_SESSION_TIMEOUT", "300"))  # 5 minutes
    MAX_SESSIONS: int = int(os.getenv("MAX_SESSIONS", "10"))
    CLEANUP_INTERVAL: int = int(os.getenv("CLEANUP_INTERVAL", "60"))  # 1 minute
    
    # Browser settings
    DEFAULT_BROWSER: str = os.getenv("DEFAULT_BROWSER", "chromium")
    DEFAULT_HEADLESS: bool = os.getenv("DEFAULT_HEADLESS", "True").lower() in ('true', '1', 't')
    DEFAULT_VIEW_WIDTH: int = int(os.getenv("DEFAULT_VIEW_WIDTH", "1280"))
    DEFAULT_VIEW_HEIGHT: int = int(os.getenv("DEFAULT_VIEW_HEIGHT", "720"))


# Create a global settings object
settings = Settings()