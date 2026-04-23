"""
go-mirofish backend entry point.
"""

import os
import sys

# Ensure UTF-8 output on Windows consoles before other imports.
if sys.platform == 'win32':
    # Ensure Python uses UTF-8.
    os.environ.setdefault('PYTHONIOENCODING', 'utf-8')
    # Reconfigure stdio streams for UTF-8 output.
    if hasattr(sys.stdout, 'reconfigure'):
        sys.stdout.reconfigure(encoding='utf-8', errors='replace')
    if hasattr(sys.stderr, 'reconfigure'):
        sys.stderr.reconfigure(encoding='utf-8', errors='replace')

# Add the project root to the module search path.
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from app import create_app
from app.config import Config


def main():
    """Run the backend process."""
    # Validate configuration.
    errors = Config.validate()
    if errors:
        print("Configuration error:")
        for err in errors:
            print(f"  - {err}")
        print("\nCheck the values in your .env file.")
        sys.exit(1)
    
    # Create the app.
    app = create_app()
    
    # Resolve runtime settings.
    host = os.environ.get('FLASK_HOST', '0.0.0.0')
    port = int(os.environ.get('FLASK_PORT', 5001))
    debug = Config.DEBUG
    
    # Start the service.
    app.run(host=host, port=port, debug=debug, threaded=True)


if __name__ == '__main__':
    main()

