#!/bin/bash

# wait-for-ci.sh - Wait for GitHub Actions CI run to complete and report results
# Usage: ./wait-for-ci.sh <run-id>
#   or: ./wait-for-ci.sh (uses most recent run)

set -e

# Check if gh CLI is installed
if ! command -v gh &> /dev/null; then
    echo "Error: GitHub CLI (gh) is not installed."
    echo "Install it from: https://cli.github.com/"
    exit 1
fi

# Get run ID from argument or fetch latest
if [ $# -eq 0 ]; then
    echo "No run ID provided, fetching most recent workflow run..."
    RUN_ID=$(gh run list --limit 1 --json databaseId --jq '.[0].databaseId')
    if [ -z "$RUN_ID" ]; then
        echo "Error: Could not fetch recent workflow runs"
        exit 1
    fi
    echo "Using run ID: $RUN_ID"
else
    RUN_ID=$1
    echo "Monitoring run ID: $RUN_ID"
fi

# Function to get run status
get_run_status() {
    gh run view "$RUN_ID" --json status,conclusion,name --jq '.status'
}

# Function to get job details
get_job_details() {
    gh run view "$RUN_ID" --json jobs --jq '.jobs[] | "\(.name):\(.conclusion)"'
}

# Function to print colored status
print_status() {
    local job_name=$1
    local status=$2

    case $status in
        success)
            echo -e "  ✅ ${job_name}: PASS"
            ;;
        failure)
            echo -e "  ❌ ${job_name}: FAIL"
            ;;
        cancelled)
            echo -e "  ⚠️  ${job_name}: CANCELLED"
            ;;
        skipped)
            echo -e "  ⏭️  ${job_name}: SKIPPED"
            ;;
        *)
            echo -e "  ❓ ${job_name}: ${status^^}"
            ;;
    esac
}

# Get initial run info
RUN_INFO=$(gh run view "$RUN_ID" --json name,headBranch,event,createdAt 2>/dev/null || true)
if [ -z "$RUN_INFO" ]; then
    echo "Error: Could not fetch run information for ID $RUN_ID"
    exit 1
fi

WORKFLOW_NAME=$(echo "$RUN_INFO" | jq -r '.name')
BRANCH=$(echo "$RUN_INFO" | jq -r '.headBranch')
EVENT=$(echo "$RUN_INFO" | jq -r '.event')
CREATED=$(echo "$RUN_INFO" | jq -r '.createdAt')

echo "=========================================="
echo "Workflow: $WORKFLOW_NAME"
echo "Branch: $BRANCH"
echo "Event: $EVENT"
echo "Started: $CREATED"
echo "=========================================="
echo ""

# Wait for run to complete
echo "Waiting for CI run to complete..."
DOTS=""
while true; do
    STATUS=$(get_run_status)

    if [ "$STATUS" == "completed" ]; then
        echo -e "\n✓ Run completed!"
        break
    fi

    # Show progress indicator
    DOTS="${DOTS}."
    if [ ${#DOTS} -gt 3 ]; then
        DOTS="."
    fi
    echo -ne "\rStatus: $STATUS$DOTS    \r"

    sleep 5
done

echo ""
echo "=========================================="
echo "Results:"
echo "=========================================="

# Get final run details
FINAL_INFO=$(gh run view "$RUN_ID" --json conclusion,jobs)
CONCLUSION=$(echo "$FINAL_INFO" | jq -r '.conclusion')

# Print job results
echo "$FINAL_INFO" | jq -r '.jobs[] | "\(.name):\(.conclusion)"' | while IFS=: read -r job_name status; do
    print_status "$job_name" "$status"
done

echo "=========================================="

# Overall result
case $CONCLUSION in
    success)
        echo -e "Overall: ✅ SUCCESS"
        exit 0
        ;;
    failure)
        echo -e "Overall: ❌ FAILURE"
        echo ""
        echo "To view failed job logs:"
        echo "  gh run view $RUN_ID --log-failed"
        exit 1
        ;;
    cancelled)
        echo -e "Overall: ⚠️  CANCELLED"
        exit 2
        ;;
    *)
        echo -e "Overall: ❓ $CONCLUSION"
        exit 3
        ;;
esac