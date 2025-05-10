FROM mcr.microsoft.com/playwright/python:v1.52.0-jammy

# Install x11vnc for VNC access to Xvfb displays
# Set environment variables to make apt-get non-interactive
ENV DEBIAN_FRONTEND=noninteractive \
    TZ=Etc/UTC

RUN apt-get update && apt-get install -y --no-install-recommends \
    x11vnc \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Set the working directory
WORKDIR /app

# Copy requirements and install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy the application code
COPY . .

# Create a non-root user to run the app
RUN useradd -m appuser
USER appuser

# Expose port
EXPOSE 8000

# Start the application
CMD ["python", "run.py"]