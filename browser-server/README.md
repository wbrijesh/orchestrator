# Browser Session Service

A service that creates and manages browser sessions on demand using Playwright and exposes Chrome DevTools Protocol (CDP) URLs for remote control of these sessions.

## Features

- Create browser sessions (Chromium, Firefox, WebKit)
- Get CDP URLs for browser control
- Manage session lifecycle
- Configure browser options (headless mode, viewport size, user agent)
- Automatic cleanup of expired sessions

## Setup

### Prerequisites

- Python 3.8+
- pip

### Installation
.
1. Clone the repository
2. Install dependencies:

```bash
cd browser-server
pip install -r requirements.txt
```

3. Install Playwright browsers:

```bash
python -m playwright install
```

### Configuration

Copy the example environment file and adjust settings as needed:

```bash
cp .env-example .env
```

Edit the `.env` file to customize settings.

## Usage

Start the server:

```bash
python main.py
```

The API will be available at http://localhost:8000.

### API Endpoints

#### Create a new browser session

```
POST /sessions
```

Request body:
```json
{
  "browser_type": "chromium",
  "headless": true,
  "viewport_size": {
    "width": 1280,
    "height": 720
  },
  "user_agent": "Mozilla/5.0 ...",
  "timeout": 300
}
```

**Example curl command:**
```bash
curl -X POST http://localhost:8000/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "browser_type": "chromium",
    "headless": true,
    "viewport_size": {
      "width": 1920,
      "height": 1080
    },
    "user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36",
    "timeout": 600
  }'
```

#### List all active sessions

```
GET /sessions
```

**Example curl command:**
```bash
curl -X GET http://localhost:8000/sessions
```

#### Get session details

```
GET /sessions/{session_id}
```

**Example curl command:**
```bash
curl -X GET http://localhost:8000/sessions/550e8400-e29b-41d4-a716-446655440000
```

#### Delete a session

```
DELETE /sessions/{session_id}
```

**Example curl command:**
```bash
curl -X DELETE http://localhost:8000/sessions/550e8400-e29b-41d4-a716-446655440000
```

## Development

### Running in Debug Mode

Set `DEBUG=True` in your `.env` file to enable automatic reloading during development.

### Adding New Features

The modular structure allows for easy extension:

- Add new browser options in `browser_manager.py`
- Extend session capabilities in `session_manager.py`
- Add new API endpoints in `main.py`
