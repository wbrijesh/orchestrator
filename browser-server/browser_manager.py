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
        vnc_url = None
        if not headless:
            display_num, vnc_url = await self.screen_manager.get_available_screen()
            if display_num is None:
                logger.warning("No available virtual displays. Falling back to headless mode.")
                headless = True
            else:
                # Print detailed information about the VNC connection
                print("\n" + "=" * 80)
                print(f"BROWSER SESSION CREATED WITH VNC ACCESS")
                print(f"Display Number: :{display_num}")
                print(f"VNC URL: {vnc_url}")
                print(f"VNC Password: {self.screen_manager.vnc_password}")
                print(f"VNC Port: {self.screen_manager.base_vnc_port + (display_num - self.screen_manager.base_display)}")
                print(f"To connect with VNC viewer: vncviewer {vnc_url.replace('vnc://', '')} -passwd {self.screen_manager.vnc_password}")
                print("=" * 80 + "\n")
                
                # Also log to the standard logger
                logger.info(f"Using virtual display :{display_num} for browser session")
                logger.info(f"VNC URL for browser session: {vnc_url} (password: {self.screen_manager.vnc_password})")
        
        # Set up launch options
        launch_options: Dict[str, Any] = {
            "headless": headless
        }
        
        # Set environment variables for the display if using a virtual display
        if display_num is not None:
            launch_options["env"] = self.screen_manager.get_display_env(display_num)
        
        # Configure browser to be visible in Xvfb
        if "args" not in launch_options:
            launch_options["args"] = []
            
        # Add arguments to make the browser visible in the virtual display
        if not headless:
            # Force window to be shown
            launch_options["args"].extend([
                "--start-maximized",
                "--no-sandbox",
                "--disable-gpu",
                "--disable-dev-shm-usage"
            ])
            
        # Only Chromium supports CDP
        if browser_type == "chromium":
            # Enable Chrome DevTools Protocol
            launch_options["args"].append("--remote-debugging-port=0")  # Use any available port
        
        # Launch the browser
        browser = await browser_instance.launch(**launch_options)
        
        # For non-Chromium browsers, we can't get CDP URL but still create the browser
        cdp_url = ""
        if browser_type == "chromium":
            # For Chromium, extract the CDP URL
            # This is implementation-specific and might need adjustment
            cdp_url = browser.wsEndpoint
            
        # For non-headless mode, create a context and page to make it visible in VNC
        if not headless:
            try:
                # Create a browser context with the specified viewport size
                context_options = {}
                if viewport_size:
                    context_options["viewport"] = viewport_size
                if user_agent:
                    context_options["user_agent"] = user_agent
                    
                context = await browser.new_context(**context_options)
                
                # Create a page in this context - this will be visible in VNC
                page = await context.new_page()
                
                # Navigate to a blank page to initialize the window
                await page.goto("about:blank")
                
                # Store the context and page for later use
                context_data = {
                    "context": context,
                    "page": page
                }
                
                print(f"Created visible browser window for display :{display_num}")
            except Exception as e:
                logger.error(f"Failed to create visible browser window: {str(e)}")
                context_data = None
        else:
            context_data = None
        
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
            "display_num": display_num,
            "vnc_url": vnc_url,
            "context_data": context_data  # Store the context and page
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
            # Close the context and page if they exist
            if browser_data["context_data"]:
                try:
                    # Close the page first
                    if browser_data["context_data"]["page"]:
                        await browser_data["context_data"]["page"].close()
                    
                    # Then close the context
                    if browser_data["context_data"]["context"]:
                        await browser_data["context_data"]["context"].close()
                except Exception as e:
                    logger.error(f"Error closing browser context: {str(e)}")
            
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