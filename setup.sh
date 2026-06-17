#!/usr/bin/env bash
set -euo pipefail

# ─── Colors ───
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}"
echo "╔════════════════════════════════════╗"
echo "║       AI Gateway Setup            ║"
echo "║   llama.cpp + Go + Tailscale      ║"
echo "╚════════════════════════════════════╝"
echo -e "${NC}"

# ─── Check Docker ───
if ! command -v docker &> /dev/null; then
    echo -e "${RED}✗ Docker is not installed.${NC}"
    echo "  Install it from: https://docs.docker.com/get-docker/"
    exit 1
fi
echo -e "${GREEN}✓ Docker found${NC}"

if ! docker compose version &> /dev/null; then
    echo -e "${RED}✗ Docker Compose is not available.${NC}"
    echo "  Make sure you have Docker Compose V2 (comes with Docker Desktop)."
    exit 1
fi
echo -e "${GREEN}✓ Docker Compose found${NC}"

# ─── Setup .env ───
if [ ! -f .env ]; then
    cp .env.example .env
    echo -e "${GREEN}✓ Created .env from .env.example${NC}"
else
    echo -e "${YELLOW}⚠ .env already exists, skipping${NC}"
fi

# ─── Prompt for API Key ───
echo ""
read -p "Enter your API key (or press Enter to auto-generate): " user_api_key
if [ -z "$user_api_key" ]; then
    user_api_key=$(openssl rand -hex 32)
    echo -e "${GREEN}✓ Generated API key: ${user_api_key}${NC}"
fi
# Update .env
if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' "s/^API_KEY=.*/API_KEY=${user_api_key}/" .env
else
    sed -i "s/^API_KEY=.*/API_KEY=${user_api_key}/" .env
fi

# ─── Prompt for Tailscale Auth Key ───
echo ""
echo -e "${CYAN}Get your Tailscale auth key from: https://login.tailscale.com/admin/settings/keys${NC}"
echo -e "${CYAN}Make sure to create a 'Reusable' key and enable 'Funnel' in your tailnet.${NC}"
read -p "Enter your Tailscale auth key (tskey-auth-...): " ts_key
if [ -n "$ts_key" ]; then
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s/^TS_AUTHKEY=.*/TS_AUTHKEY=${ts_key}/" .env
    else
        sed -i "s/^TS_AUTHKEY=.*/TS_AUTHKEY=${ts_key}/" .env
    fi
    echo -e "${GREEN}✓ Tailscale auth key saved${NC}"
else
    echo -e "${YELLOW}⚠ No Tailscale key provided. You can add it later in .env${NC}"
fi

# ─── Download Default Model ───
MODEL_URL="https://huggingface.co/Qwen/Qwen2.5-7B-Instruct-GGUF/resolve/main/qwen2.5-7b-instruct-q4_k_m.gguf"
MODEL_FILE="models/qwen2.5-7b-instruct-q4_k_m.gguf"

echo ""
if [ -f "$MODEL_FILE" ]; then
    echo -e "${YELLOW}⚠ Model already downloaded: ${MODEL_FILE}${NC}"
else
    echo -e "${CYAN}Downloading Qwen 2.5 7B (Q4_K_M, ~4.5GB)...${NC}"
    echo -e "${CYAN}This will take a few minutes depending on your internet speed.${NC}"
    echo ""

    if command -v curl &> /dev/null; then
        curl -L --progress-bar -o "$MODEL_FILE" "$MODEL_URL"
    elif command -v wget &> /dev/null; then
        wget --show-progress -O "$MODEL_FILE" "$MODEL_URL"
    else
        echo -e "${RED}✗ Neither curl nor wget found. Please download the model manually:${NC}"
        echo "  URL: $MODEL_URL"
        echo "  Save to: $MODEL_FILE"
        exit 1
    fi
    echo -e "${GREEN}✓ Model downloaded${NC}"
fi

# ─── Detect GPU ───
echo ""
if command -v nvidia-smi &> /dev/null; then
    echo -e "${GREEN}✓ NVIDIA GPU detected${NC}"
    read -p "Use GPU for inference? (Y/n): " use_gpu
    if [[ "$use_gpu" != "n" && "$use_gpu" != "N" ]]; then
        echo -e "${CYAN}Starting with GPU support...${NC}"
        docker compose -f docker-compose.yml -f docker-compose.gpu.yml up -d --build
    else
        echo -e "${CYAN}Starting in CPU-only mode...${NC}"
        docker compose up -d --build
    fi
else
    echo -e "${YELLOW}ℹ No NVIDIA GPU detected, running in CPU-only mode${NC}"
    docker compose up -d --build
fi

# ─── Wait for Tailscale to connect and get the public URL ───
echo ""
echo -e "${CYAN}Waiting for Tailscale to connect...${NC}"

PUBLIC_URL=""
for i in $(seq 1 30); do
    # Try to get the funnel status
    FUNNEL_OUTPUT=$(docker exec ai-gateway-tailscale tailscale funnel status 2>/dev/null || true)
    
    # Extract the https URL from funnel output
    URL=$(echo "$FUNNEL_OUTPUT" | grep -oE 'https://[a-zA-Z0-9._-]+\.ts\.net' | head -1 || true)
    
    if [ -n "$URL" ]; then
        PUBLIC_URL="$URL"
        break
    fi
    
    # Also try tailscale status to get the machine name
    if [ -z "$PUBLIC_URL" ]; then
        TS_STATUS=$(docker exec ai-gateway-tailscale tailscale status --self 2>/dev/null || true)
        TS_NAME=$(echo "$TS_STATUS" | awk '{print $2}' | head -1 || true)
        if [ -n "$TS_NAME" ] && [[ "$TS_NAME" == *"."* ]]; then
            PUBLIC_URL="https://${TS_NAME}"
            break
        fi
    fi

    echo -n "."
    sleep 2
done
echo ""

# ─── Done ───
echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║            AI Gateway is running! 🚀                ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════════╝${NC}"
echo ""

if [ -n "$PUBLIC_URL" ]; then
    echo -e "┌──────────────────────────────────────────────────────┐"
    echo -e "│                                                      │"
    echo -e "│  ${GREEN}YOUR PUBLIC ENDPOINT:${NC}                                │"
    echo -e "│                                                      │"
    echo -e "│  ${CYAN}${PUBLIC_URL}${NC}"
    echo -e "│                                                      │"
    echo -e "└──────────────────────────────────────────────────────┘"
    echo ""
    echo -e "  API Base URL:  ${CYAN}${PUBLIC_URL}/v1${NC}"
    echo -e "  Health Check:  ${CYAN}${PUBLIC_URL}/health${NC}"
    echo -e "  Hardware:      ${CYAN}${PUBLIC_URL}/v1/hardware${NC}"
else
    echo -e "${YELLOW}⚠ Tailscale is still connecting. Run this to get your URL:${NC}"
    echo -e "  ${CYAN}docker exec ai-gateway-tailscale tailscale funnel status${NC}"
    echo ""
    echo -e "  Local:     ${CYAN}http://localhost:8080${NC}"
    echo -e "  Health:    ${CYAN}http://localhost:8080/health${NC}"
    echo -e "  Hardware:  ${CYAN}http://localhost:8080/v1/hardware${NC}"
fi

echo ""
echo -e "  Your API key:  ${YELLOW}${user_api_key}${NC}"
echo ""
echo -e "  ${GREEN}Test it:${NC}"

if [ -n "$PUBLIC_URL" ]; then
    echo -e "  curl ${PUBLIC_URL}/v1/chat/completions \\"
else
    echo -e "  curl http://localhost:8080/v1/chat/completions \\"
fi
echo -e "    -H 'Authorization: Bearer ${user_api_key}' \\"
echo -e "    -H 'Content-Type: application/json' \\"
echo -e "    -d '{\"model\":\"qwen2.5\",\"messages\":[{\"role\":\"user\",\"content\":\"Hello!\"}]}'"
echo ""

