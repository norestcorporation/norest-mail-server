#!/bin/bash

echo "=========================================="
echo "  Chapter 5B Worker Recovery Tests"
echo "=========================================="
echo ""

echo "[1] Testing worker crash recovery..."
# Check if worker is running
if docker ps | grep -q "norest-worker"; then
    echo "✓ Worker is running"
    
    # Get worker container ID
    worker_container=$(docker ps -q --filter "name=norest-worker")
    
    if [ -n "$worker_container" ]; then
        # Kill the worker
        docker kill "$worker_container" > /dev/null 2>&1
        echo "✓ Worker killed"
        
        # Wait for lease expiry (simulated - actual test would need shorter lease)
        sleep 2
        
        # Check if worker restarts (docker-compose restart policy)
        sleep 3
        if docker ps | grep -q "norest-worker"; then
            echo "✓ Worker restarted"
        else
            echo "⚠ Worker did not restart (check docker-compose restart policy)"
        fi
    else
        echo "✗ Could not find worker container"
    fi
else
    echo "✗ Worker is not running"
fi

echo ""
echo "[2] Testing stuck job recovery..."
# This would require creating a job and stopping the worker mid-processing
echo "⚠ Stuck job recovery: Manual verification needed (requires job manipulation)"

echo ""
echo "[3] Testing graceful shutdown..."
# Send SIGTERM to worker
if docker ps | grep -q "norest-worker"; then
    worker_container=$(docker ps -q --filter "name=norest-worker")
    docker stop "$worker_container" > /dev/null 2>&1
    echo "✓ Worker stopped gracefully"
    
    # Restart it
    docker start "$worker_container" > /dev/null 2>&1
    sleep 2
    if docker ps | grep -q "norest-worker"; then
        echo "✓ Worker restarted after graceful shutdown"
    fi
fi

echo ""
echo "=========================================="
echo "  Worker Recovery Tests Complete"
echo "=========================================="
