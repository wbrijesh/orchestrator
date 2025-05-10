import asyncio
import os
from typing import Dict, Optional, Tuple, Any
from playwright.async_api import async_playwright, Browser, BrowserType
from config import settings
import uuid
import logging
from screen_manager import ScreenManager

logger = logging.getLogger(__name__)


class BrowserManager:
    """Manages browser instances using Playwright"""

    def __init__(self, max_screens: int = 10):
        self.playwright = None
        self.browsers: Dict[str, Dict[str, Any]] = {}
        self._lock = asyncio.Lock()
        self.screen_manager = ScreenManager(max_screens=max_screens)
        # Track which browser is using which display
        self.browser_displays: Dict[str, int] = {}
        
    async def initialize(self):
        """Initialize the Playwright instance and screen manager"""
        self.playwright = await async_playwright().start()
        logger.info("Playwright initialized")
        
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
        
        # For headed browsers, get an available virtual display
        display_num = None
        if not headless:
            display_num = await self.screen_manager.get_available_screen()
            if display_num is None:
                logger.warning("No available virtual displays. Falling back to headless mode.")
                headless = True
            else:
                logger.info(f"Using virtual display :{display_num} for browser session")
        
        # Set up launch options
        launch_options: Dict[str, Any] = {
            "headless": headless
        }
        
        # Set environment variables for the display if using a virtual display
        if display_num is not None:
            launch_options["env"] = self.screen_manager.get_display_env(display_num)
        
        # Only Chromium supports CDP
        if browser_type == "chromium":
            # Enable Chrome DevTools Protocol
            if "args" not in launch_options:
                launch_options["args"] = []
            launch_options["args"].append("--remote-debugging-port=0")  # Use any available port
        
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
            "user_agent": user_agent,
            "display_num": display_num
        }
        
        # Store the browser instance and display mapping
        async with self._lock:
            self.browsers[browser_id] = browser_data
            if display_num is not None:
                self.browser_displays[browser_id] = display_num
        
        return browser_data, cdp_url
    
    async def close_browser(self, browser_id: str) -> bool:
        """Close a specific browser instance and release its virtual display if any"""
        browser_data = None
        display_num = None
        
        async with self._lock:
            browser_data = self.browsers.pop(browser_id, None)
            display_num = self.browser_displays.pop(browser_id, None)
        
        if browser_data:
            # Close the browser instance
            await browser_data["instance"].close()
            
            # Release the virtual display if one was used
            if display_num is not None:
                await self.screen_manager.release_screen(display_num)
                logger.info(f"Released virtual display :{display_num} from browser {browser_id}")
                
            return True
        return False
    
    async def cleanup(self):
        """Close all browser instances and cleanup resources including virtual displays"""
        if not self.playwright:
            return
            
        async with self._lock:
            # Close all browser instances
            for browser_id, browser_data in list(self.browsers.items()):
                try:
                    await browser_data["instance"].close()
                except Exception as e:
                    logger.error(f"Error closing browser {browser_id}: {str(e)}")
            
            # Clear browser tracking
            self.browsers.clear()
            self.browser_displays.clear()
            
        # Clean up the screen manager
        await self.screen_manager.cleanup()
        logger.info("All virtual displays cleaned up")
            
        # Stop playwright
        if self.playwright:
            await self.playwright.stop()
            self.playwright = None
            logger.info("Playwright stopped")