import asyncio
from typing import Dict, Optional, Tuple, Any
from playwright.async_api import async_playwright, Browser, BrowserType
from config import settings
import uuid


class BrowserManager:
    """Manages browser instances using Playwright"""

    def __init__(self):
        self.playwright = None
        self.browsers: Dict[str, Dict[str, Any]] = {}
        self._lock = asyncio.Lock()
        
    async def initialize(self):
        """Initialize the Playwright instance"""
        self.playwright = await async_playwright().start()
        
    async def create_browser_instance(
        self, 
        browser_type: str, 
        headless: bool = True, 
        viewport_size: Optional[Dict[str, int]] = None,
        user_agent: Optional[str] = None
    ) -> Tuple[Dict[str, Any], str]:
        """Create a new browser instance and return the browser object and CDP URL"""
        if not self.playwright:
            raise RuntimeError("BrowserManager not initialized")
            
        browser_types = {
            "chromium": self.playwright.chromium,
            "firefox": self.playwright.firefox,
            "webkit": self.playwright.webkit
        }
        
        if browser_type not in browser_types:
            raise ValueError(f"Unsupported browser type: {browser_type}. "
                            f"Supported types are: {', '.join(browser_types.keys())}")
            
        browser_instance: BrowserType = browser_types[browser_type]
        
        launch_options: Dict[str, Any] = {
            "headless": headless
        }
        
        # Only Chromium supports CDP
        if browser_type == "chromium":
            # Enable Chrome DevTools Protocol
            launch_options["args"] = ["--remote-debugging-port=0"]  # Use any available port
        
        # Launch the browser with a context
        browser = await browser_instance.launch(**launch_options)
        
        # For non-Chromium browsers, we can't get CDP URL but still create the browser
        cdp_url = ""
        if browser_type == "chromium":
            # For Chromium, extract the CDP URL
            # This is implementation-specific and might need adjustment
            cdp_url = browser.wsEndpoint
        
        # Generate a unique ID for this browser instance
        browser_id = str(uuid.uuid4())
        
        # Create browser data object with ID
        browser_data = {
            "id": browser_id,
            "instance": browser,
            "type": browser_type,
            "headless": headless,
            "viewport_size": viewport_size,
            "user_agent": user_agent
        }
        
        # Store the browser instance
        async with self._lock:
            self.browsers[browser_id] = browser_data
        
        return browser_data, cdp_url
    
    async def close_browser(self, browser_id: str) -> bool:
        """Close a specific browser instance"""
        async with self._lock:
            browser_data = self.browsers.pop(browser_id, None)
        
        if browser_data:
            await browser_data["instance"].close()
            return True
        return False
    
    async def cleanup(self):
        """Close all browser instances and cleanup resources"""
        if not self.playwright:
            return
            
        async with self._lock:
            for browser_id, browser_data in list(self.browsers.items()):
                try:
                    await browser_data["instance"].close()
                except Exception:
                    pass  # Ignore errors during cleanup
            self.browsers.clear()
            
        if self.playwright:
            await self.playwright.stop()
            self.playwright = None