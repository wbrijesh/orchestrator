import asyncio
import os
import subprocess
import logging
import socket
from typing import Dict, Optional, List, Tuple

logger = logging.getLogger(__name__)

class ScreenManager:
    """
    Manages multiple Xvfb virtual displays for browser sessions.
    Each browser session can be assigned its own virtual display.
    Also provides VNC access to these displays for remote viewing.
    """
    
    def __init__(self, max_screens: int = 10, base_display: int = 99, base_vnc_port: int = 5900):
        """
        Initialize the screen manager.
        
        Args:
            max_screens: Maximum number of virtual displays to manage
            base_display: Starting display number (e.g., :99, :100, etc.)
            base_vnc_port: Starting VNC port (default is 5900)
        """
        self.max_screens = max_screens
        self.base_display = base_display
        self.base_vnc_port = base_vnc_port
        self.screens: Dict[int, Dict] = {}
        self._lock = asyncio.Lock()
        
        # Get VNC password from environment or use default
        self.vnc_password = os.environ.get('VNC_PASSWORD', 'vncpass')
        
        # Get hostname for VNC connection URLs
        self.hostname = self._get_hostname()
        
        # Initialize the screen tracking dictionary
        for i in range(max_screens):
            display_num = base_display + i
            vnc_port = base_vnc_port + i
            self.screens[display_num] = {
                "in_use": False,
                "xvfb_process": None,
                "vnc_process": None,
                "width": 1280,
                "height": 1024,
                "depth": 24,
                "vnc_port": vnc_port,
                "vnc_url": f"vnc://{self.hostname}:{vnc_port}"
            }
    
    def _get_hostname(self) -> str:
        """
        Get the hostname for VNC connection URLs.
        In Docker, we use the container's hostname which is accessible from the host.
        """
        try:
            # Try to get the hostname
            hostname = socket.gethostname()
            # In Docker, this will be the container ID or name
            return hostname
        except Exception:
            # Fall back to localhost if we can't get the hostname
            return "localhost"
    
    async def start_screen(self, display_num: int, width: int = 1280, height: int = 1024, depth: int = 24) -> Tuple[bool, Optional[str]]:
        """
        Start an Xvfb virtual display and a VNC server for it.
        
        Args:
            display_num: The display number to use (e.g., 99 for :99)
            width: Screen width in pixels
            height: Screen height in pixels
            depth: Color depth in bits
            
        Returns:
            Tuple[bool, Optional[str]]: (success, vnc_url) - success is True if the screen was started successfully,
                                         vnc_url is the URL to connect to the VNC server or None if failed
        """
        if display_num not in self.screens:
            logger.error(f"Invalid display number: {display_num}")
            return False, None
            
        screen_info = self.screens[display_num]
        
        # If the screen is already running, return True and the VNC URL
        if screen_info["xvfb_process"] and screen_info["xvfb_process"].poll() is None:
            return True, screen_info["vnc_url"]
            
        # Start Xvfb with the specified parameters
        try:
            # 1. Start Xvfb
            xvfb_cmd = [
                "Xvfb", 
                f":{display_num}", 
                "-screen", "0", 
                f"{width}x{height}x{depth}",
                "-ac",
                "+extension", "GLX",
                "+render",
                "-noreset"
            ]
            
            xvfb_process = subprocess.Popen(
                xvfb_cmd, 
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE
            )
            
            # Wait a moment to ensure Xvfb starts properly
            await asyncio.sleep(1)
            
            # Check if the process is still running
            if xvfb_process.poll() is not None:
                stderr = xvfb_process.stderr.read().decode('utf-8')
                logger.error(f"Failed to start Xvfb on display :{display_num}. Error: {stderr}")
                return False, None
            
            # Update screen info with Xvfb process
            screen_info["xvfb_process"] = xvfb_process
            screen_info["width"] = width
            screen_info["height"] = height
            screen_info["depth"] = depth
            
            logger.info(f"Started Xvfb on display :{display_num} ({width}x{height}x{depth})")
            
            # 2. Start x11vnc server for this display
            vnc_port = screen_info["vnc_port"]
            vnc_cmd = [
                "x11vnc",
                "-display", f":{display_num}",
                "-forever",
                "-shared",
                "-rfbport", str(vnc_port),
                "-passwd", self.vnc_password,
                "-noxdamage",
                "-noxfixes",
                "-noxrecord",
                "-nopw",  # Don't ask for a password interactively
                "-q"      # Quiet mode
            ]
            
            vnc_process = subprocess.Popen(
                vnc_cmd,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE
            )
            
            # Wait a moment to ensure VNC server starts properly
            await asyncio.sleep(1)
            
            # Check if the VNC process is still running
            if vnc_process.poll() is not None:
                stderr = vnc_process.stderr.read().decode('utf-8')
                logger.error(f"Failed to start VNC server for display :{display_num}. Error: {stderr}")
                # We still return True because Xvfb started successfully
                return True, None
            
            # Update screen info with VNC process
            screen_info["vnc_process"] = vnc_process
            
            # Log the VNC connection URL
            vnc_url = screen_info["vnc_url"]
            vnc_port = screen_info["vnc_port"]
            
            # Print detailed VNC connection information
            print("\n" + "*" * 80)
            print(f"VNC SERVER STARTED")
            print(f"Display: :{display_num}")
            print(f"Resolution: {width}x{height}x{depth}")
            print(f"VNC URL: {vnc_url}")
            print(f"VNC Port: {vnc_port}")
            print(f"VNC Password: {self.vnc_password}")
            print(f"Host: {self.hostname}")
            print(f"Connection command: vncviewer {self.hostname}:{vnc_port} -passwd {self.vnc_password}")
            print("*" * 80 + "\n")
            
            # Also log to the standard logger
            logger.info(f"Started VNC server for display :{display_num} - Connect at {vnc_url} with password '{self.vnc_password}'")
            
            return True, vnc_url
            
        except Exception as e:
            logger.error(f"Error starting display :{display_num}: {str(e)}")
            return False, None
    
    async def stop_screen(self, display_num: int) -> bool:
        """
        Stop an Xvfb virtual display and its associated VNC server.
        
        Args:
            display_num: The display number to stop
            
        Returns:
            bool: True if the screen was stopped successfully, False otherwise
        """
        if display_num not in self.screens:
            logger.error(f"Invalid display number: {display_num}")
            return False
            
        screen_info = self.screens[display_num]
        success = True
        
        # Stop VNC server first if it exists
        if screen_info["vnc_process"]:
            try:
                screen_info["vnc_process"].terminate()
                await asyncio.sleep(0.5)
                
                # Force kill if still running
                if screen_info["vnc_process"].poll() is None:
                    screen_info["vnc_process"].kill()
                    
                screen_info["vnc_process"] = None
                logger.info(f"Stopped VNC server for display :{display_num}")
            except Exception as e:
                logger.error(f"Error stopping VNC server for display :{display_num}: {str(e)}")
                success = False
        
        # Then stop Xvfb
        if screen_info["xvfb_process"]:
            try:
                screen_info["xvfb_process"].terminate()
                await asyncio.sleep(0.5)
                
                # Force kill if still running
                if screen_info["xvfb_process"].poll() is None:
                    screen_info["xvfb_process"].kill()
                    
                screen_info["xvfb_process"] = None
                logger.info(f"Stopped Xvfb on display :{display_num}")
            except Exception as e:
                logger.error(f"Error stopping Xvfb on display :{display_num}: {str(e)}")
                success = False
        
        return success
    
    async def get_available_screen(self) -> Tuple[Optional[int], Optional[str]]:
        """
        Get an available screen number and its VNC URL.
        
        Returns:
            Tuple[Optional[int], Optional[str]]: (display_num, vnc_url) - The display number and VNC URL of an available screen,
                                                 or (None, None) if all screens are in use
        """
        async with self._lock:
            for display_num, screen_info in self.screens.items():
                if not screen_info["in_use"]:
                    # Start the screen if it's not already running
                    success, vnc_url = await self.start_screen(
                        display_num,
                        screen_info["width"],
                        screen_info["height"],
                        screen_info["depth"]
                    )
                    
                    if success:
                        screen_info["in_use"] = True
                        return display_num, vnc_url
            
            return None, None  # All screens are in use
    
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
        """Stop all Xvfb and VNC processes and clean up resources"""
        async with self._lock:
            for display_num in self.screens:
                await self.stop_screen(display_num)
            logger.info("All virtual displays and VNC servers stopped")
                
    def get_display_env(self, display_num: int) -> Dict[str, str]:
        """
        Get environment variables needed for a specific display.
        
        Args:
            display_num: The display number
            
        Returns:
            dict: Environment variables for the display
        """
        return {"DISPLAY": f":{display_num}"}
    
    def get_vnc_url(self, display_num: int) -> Optional[str]:
        """
        Get the VNC URL for a specific display.
        
        Args:
            display_num: The display number
            
        Returns:
            str or None: The VNC URL if the display exists, None otherwise
        """
        if display_num not in self.screens:
            return None
        return self.screens[display_num]["vnc_url"]
    
    async def get_active_screens(self) -> List[Tuple[int, str]]:
        """
        Get a list of active screens with their VNC URLs.
        
        Returns:
            list: List of tuples (display_num, vnc_url) for active displays
        """
        active_screens = []
        async with self._lock:
            for display_num, screen_info in self.screens.items():
                if screen_info["in_use"]:
                    active_screens.append((display_num, screen_info["vnc_url"]))
        return active_screens
        
    async def get_all_vnc_urls(self) -> Dict[int, str]:
        """
        Get a dictionary of all VNC URLs for active screens.
        
        Returns:
            dict: Dictionary mapping display numbers to VNC URLs for active displays
        """
        vnc_urls = {}
        async with self._lock:
            for display_num, screen_info in self.screens.items():
                if screen_info["in_use"] and screen_info["vnc_process"] and screen_info["vnc_process"].poll() is None:
                    vnc_urls[display_num] = screen_info["vnc_url"]
        return vnc_urls
