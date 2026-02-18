#!/bin/bash
# Record all demo GIFs for LogPilot README.
# Usage: ./docs/demos/record-all.sh
# Requires: asciinema, agg (https://github.com/asciinema/agg)

set -e
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
DEMO="$REPO_DIR/demo"
GIF_DIR="$SCRIPT_DIR"

# Build demo tool
echo "Building demo tool..."
cd "$REPO_DIR"
go build -o demo ./cmd/demo/

# Colors for output
GREEN='\033[0;32m'
NC='\033[0m'

record_demo() {
    local name="$1"
    local script="$2"
    local cast_file="$GIF_DIR/${name}.cast"
    local gif_file="$GIF_DIR/${name}.gif"

    echo -e "${GREEN}Recording: ${name}${NC}"
    
    asciinema rec "$cast_file" \
        --cols 120 \
        --rows 30 \
        --command "$script" \
        --overwrite \
        --quiet

    echo -e "${GREEN}Converting to GIF: ${name}${NC}"
    "$REPO_DIR/agg" --cols 120 --rows 30 --speed 1.5 "$cast_file" "$gif_file" 2>/dev/null || \
    agg --cols 120 --rows 30 --speed 1.5 "$cast_file" "$gif_file"

    # Clean up cast file
    rm -f "$cast_file"
    echo "  → $gif_file"
}

# 1. Hero demo: JSON logs with color-coded output
record_demo "demo-json" "$DEMO $SCRIPT_DIR/sample-json.log"

# 2. Logfmt format demo
record_demo "demo-logfmt" "$DEMO $SCRIPT_DIR/sample-logfmt.log"

# 3. Plain text format demo  
record_demo "demo-plain" "$DEMO $SCRIPT_DIR/sample-plain.log"

# 4. Pipe demo: simulated streaming
record_demo "demo-pipe" "bash -c 'echo \"Streaming logs via pipe...\"; echo; cat $SCRIPT_DIR/sample-json.log | $DEMO /dev/stdin'"

# 5. Format detection demo: show all 3 formats
record_demo "demo-formats" "bash -c '
echo \"=== JSON Logs ===\"
$DEMO $SCRIPT_DIR/sample-json.log
echo
echo \"=== Logfmt Logs ===\"
$DEMO $SCRIPT_DIR/sample-logfmt.log
echo
echo \"=== Plain Text Logs ===\"
$DEMO $SCRIPT_DIR/sample-plain.log
'"

echo ""
echo -e "${GREEN}All demos recorded!${NC}"
echo "GIFs saved to: $GIF_DIR/"
ls -la "$GIF_DIR"/*.gif 2>/dev/null
