#!/bin/bash

# StreamLite Setup Script
# This script helps you set up StreamLite for the first time

set -e

echo "=============================="
echo "StreamLite Setup"
echo "=============================="
echo ""

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "Error: Docker is not installed. Please install Docker first."
    exit 1
fi

# Check if Docker Compose is available
if ! docker compose version &> /dev/null; then
    echo "Error: Docker Compose is not available. Please install Docker Compose."
    exit 1
fi

# Create necessary directories
echo "Creating directories..."
mkdir -p videos config

# Create .env file for frontend if it doesn't exist
if [ ! -f frontend/.env ]; then
    echo "Creating frontend/.env file..."
    cp frontend/.env.example frontend/.env
fi

echo ""
echo "=============================="
echo "Setup Complete!"
echo "=============================="
echo ""
echo "Next steps:"
echo "1. Place your video files in the 'videos' directory"
echo "2. Run 'docker compose up -d' to start all services"
echo "3. Access the application at http://localhost:3000"
echo ""
echo "For more information, see README.md"
