#!/bin/bash
set -e

# Check if HOST is provided
if [ -z "$1" ]; then
    echo "Usage: $0 <host> [user] [deploy-path]"
    echo "Example: $0 192.168.1.50 myuser /opt/hubstack"
    exit 1
fi

HOST=$1
USER=${2:-$USER}
DEPLOY_PATH=${3:-/opt/hubstack}

echo "==> Deploying to $USER@$HOST:$DEPLOY_PATH"

# Create remote directory and set ownership
echo "==> Creating remote directory..."
ssh -t "$USER@$HOST" "sudo mkdir -p $DEPLOY_PATH && sudo chown -R $USER:$USER $DEPLOY_PATH"

# Copy files to remote host
echo "==> Copying files to remote host..."
rsync -avz --delete \
    --exclude='.git' \
    --exclude='bin/hubstack' \
    --exclude='*.md' \
    --exclude='go.mod' \
    --exclude='go.sum' \
    --exclude='cmd/' \
    --exclude='internal/' \
    ./bin \
    ./templates \
    ./docker-compose.yml \
    ./.dockerignore \
    ./config.yml \
    "$USER@$HOST:$DEPLOY_PATH/"

# Set proper permissions for Docker container access
echo "==> Setting permissions..."
ssh "$USER@$HOST" "chmod 666 $DEPLOY_PATH/config.yml"

# Deploy with docker-compose on remote host
echo "==> Building and starting Docker container on remote host..."
ssh -t "$USER@$HOST" "cd $DEPLOY_PATH && sudo docker compose up -d --build"

echo "==> Deployment complete!"
echo "==> View logs with: make remote-logs HOST=$HOST"
echo "==> Check status with: make remote-status HOST=$HOST"
echo "==> Restart with: make remote-restart HOST=$HOST"
echo "==> Stop with: make remote-stop HOST=$HOST"
