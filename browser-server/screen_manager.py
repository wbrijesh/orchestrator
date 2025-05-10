import asyncio
import os
import subprocess
import logging
from typing import Dict, Optional, List

logger = logging.getLogger(__name__)

class ScreenManager:
    """
    Manages multiple Xvfb virtual displays for browser sessions.
    Each browser session can be assigned its own virtual display.
    """
    
    def __init__(self, max_screens: int = 10, base_display: int = 99):
        """
        Initialize the screen manager.
        
        Args:
            max_screens: Maximum number of virtual displays to manage
            base_display: Starting display number (e.g., :99, :100, etc.)
        """
        self.max_screens = max_screens
        self.base_display = base_display
        self.screens: Dict[int, Dict] = {}
        self._lock = asyncio.Lock()
        
        # Initialize the screen tracking dictionary
        for i in range(max_screens):
            display_num = base_display + i
            self.screens[display_num] = {
                "in_use": False,
                "process": None,
                "width": 1280,
                "height": 1024,
                "depth": 24
            }
    
    async def start_screen(self, display_num: int, width: int = 1280, height: int = 1024, depth: int = 24) -> bool:
        """
        Start an Xvfb virtual display.
        
        Args:
            display_num: The display number to use (e.g., 99 for :99)
            width: Screen width in pixels
            height: Screen height in pixels
            depth: Color depth in bits
            
        Returns:
            bool: True if the screen was started successfully, False otherwise
        """
        if display_num not in self.screens:
            logger.error(f"Invalid display number: {display_num}")
            return False
            
        screen_info = self.screens[display_num]
        
        # If the screen is already running, return True
        if screen_info["process"] and screen_info["process"].poll() is None:
            return True
            
        # Start Xvfb with the specified parameters
        try:
            cmd = [
                "Xvfb", 
                f":{display_num}", 
                "-screen", "0", 
                f"{width}x{height}x{depth}",
                "-ac",
                "+extension", "GLX",
                "+render",
                "-noreset"
            ]
            
            process = subprocess.Popen(
                cmd, 
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE
            )
            
            # Wait a moment to ensure Xvfb starts properly
            await asyncio.sleep(1)
            
            # Check if the process is still running
            if process.poll() is not None:
                stderr = process.stderr.read().decode('utf-8')
                logger.error(f"Failed to start Xvfb on display :{display_num}. Error: {stderr}")
                return False
                
            # Update screen info
            screen_info["process"] = process
            screen_info["width"] = width
            screen_info["height"] = height
            screen_info["depth"] = depth
            
            logger.info(f"Started Xvfb on display :{display_num} ({width}x{height}x{depth})")
            return True
            
        except Exception as e:
            logger.error(f"Error starting Xvfb on display :{display_num}: {str(e)}")
            return False
    
    async def stop_screen(self, display_num: int) -> bool:
        """
        Stop an Xvfb virtual display.
        
        Args:
            display_num: The display number to stop
            
        Returns:
            bool: True if the screen was stopped successfully, False otherwise
        """
        if display_num not in self.screens:
            logger.error(f"Invalid display number: {display_num}")
            return False
            
        screen_info = self.screens[display_num]
        
        if screen_info["process"]:
            try:
                screen_info["process"].terminate()
                await asyncio.sleep(0.5)
                
                # Force kill if still running
                if screen_info["process"].poll() is None:
                    screen_info["process"].kill()
                    
                screen_info["process"] = None
                logger.info(f"Stopped Xvfb on display :{display_num}")
                return True
                
            except Exception as e:
                logger.error(f"Error stopping Xvfb on display :{display_num}: {str(e)}")
                return False
        
        return True  # Already stopped
    
    async def get_available_screen(self) -> Optional[int]:
        """
        Get an available screen number.
        
        Returns:
            int or None: The display number of an available screen, or None if all screens are in use
        """
        async with self._lock:
            for display_num, screen_info in self.screens.items():
                if not screen_info["in_use"]:
                    # Start the screen if it's not already running
                    success = await self.start_screen(
                        display_num,
                        screen_info["width"],
                        screen_info["height"],
                        screen_info["depth"]
                    )
                    
                    if success:
                        screen_info["in_use"] = True
                        return display_num
            
            return None  # All screens are in use
    
    async def release_screen(self, display_num: int) -> bool:
        """
        Release a screen so it can be used by another session.
        
        Args:
            display_num: The display number to release
            
        Returns:
            bool: True if the screen was released successfully, False otherwise
        """
        async with self._lock:
            if display_num not in self.screens:
                logger.error(f"Invalid display number: {display_num}")
                return False
                
            self.screens[display_num]["in_use"] = False
            logger.info(f"Released display :{display_num}")
            return True
    
    async def cleanup(self):
        """Stop all Xvfb processes and clean up resources"""
        async with self._lock:
            for display_num in self.screens:
                await self.stop_screen(display_num)
                
    def get_display_env(self, display_num: int) -> Dict[str, str]:
        """
        Get environment variables needed for a specific display.
        
        Args:
            display_num: The display number
            
        Returns:
            dict: Environment variables for the display
        """
        return {"DISPLAY": f":{display_num}"}
    
    async def get_active_screens(self) -> List[int]:
        """
        Get a list of active screen numbers.
        
        Returns:
            list: List of active display numbers
        """
        active_screens = []
        async with self._lock:
            for display_num, screen_info in self.screens.items():
                if screen_info["in_use"]:
                    active_screens.append(display_num)
        return active_screens
