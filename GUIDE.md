# raspberry pi deployment guide

from zero to a working AI endpoint on your pi.

---

## before you start

**what you need:**
- raspberry pi 4 (4GB or 8GB) or raspberry pi 5
- 64-bit raspberry pi OS (bookworm recommended)
- microSD card (32GB+) or USB SSD
- ethernet cable or wifi connected

**important: model size vs RAM**

| your pi RAM | max model | recommendation |
|---|---|---|
| 4GB | ~2B params | qwen2.5-1.5b (Q4, ~1GB) |
| 8GB | ~7B params | qwen2.5-3b (Q4, ~2GB) or qwen2.5-7b (Q4, ~4.5GB, tight) |

the 7B model on 8GB pi will work but it'll be slow and use almost all your RAM. the 3B model is the sweet spot.

---

## step 1: install docker

run these commands one by one:

```bash
# 1. download docker install script
curl -fsSL https://get.docker.com -o get-docker.sh

# 2. run it
sudo sh get-docker.sh

# 3. add your user to the docker group (so you don't need sudo)
sudo usermod -aG docker $USER

# 4. log out and back in for the group change to take effect
exit
```

log out and back in (or just run `newgrp docker`).

verify docker works:

```bash
docker --version
# should print something like: Docker version 27.x.x
```

---

## step 2: install git (if not already)

```bash
sudo apt-get update && sudo apt-get install -y git
```

---

## step 3: clone the repo

```bash
git clone https://github.com/ThomasVuNguyen/ai-gateway.git
cd ai-gateway
```

(replace the URL with wherever you push this repo)

---

## step 4: get a tailscale auth key

1. go to https://login.tailscale.com/admin/settings/keys on your mac
2. click **"Generate auth key..."**
3. check **"Reusable"**
4. check **"Ephemeral"** (optional, cleans up if the pi goes offline)
5. click **"Generate key"**
6. copy the key (starts with `tskey-auth-...`)

also, make sure funnel is enabled:

1. go to https://login.tailscale.com/admin/dns
2. scroll to **"HTTPS Certificates"** → enable it
3. go to https://login.tailscale.com/admin/acls/file
4. add this to your ACL policy (inside the `nodeAttrs` section, or create one):

```json
"nodeAttrs": [
    {
        "target": ["*"],
        "attr": ["funnel"]
    }
]
```

---

## step 5: run setup

```bash
./setup.sh
```

it will ask you:

1. **API key** → press Enter to auto-generate one (it'll print it, save it somewhere)
2. **Tailscale auth key** → paste the `tskey-auth-...` key from step 4

then it downloads the model (~1-4.5GB depending on your pi) and starts everything.

**⚠️ if you have a 4GB pi**, edit `.env` BEFORE running setup to use a smaller model:

```bash
# edit .env first
nano .env
```

change these two lines:

```
MODEL_FILE=qwen2.5-1.5b-instruct-q4_k_m.gguf
MODEL_ALIAS=qwen2.5-1.5b
```

then also change the download URL in setup.sh line ~76 to:

```
MODEL_URL="https://huggingface.co/Qwen/Qwen2.5-1.5B-Instruct-GGUF/resolve/main/qwen2.5-1.5b-instruct-q4_k_m.gguf"
```

or just manually download the model:

```bash
curl -L -o models/qwen2.5-1.5b-instruct-q4_k_m.gguf \
  https://huggingface.co/Qwen/Qwen2.5-1.5B-Instruct-GGUF/resolve/main/qwen2.5-1.5b-instruct-q4_k_m.gguf
```

---

## step 6: verify it's working

```bash
# check all containers are running
docker compose ps
```

you should see 3 containers: `ai-gateway-tailscale`, `ai-gateway`, `ai-gateway-llama`

```bash
# check health
curl http://localhost:8080/health
```

should return something like:

```json
{"status":"ok","gateway":"running","llama_server":{"status":"ok","url":"http://llama-server:8081"}}
```

```bash
# check hardware
curl -H "Authorization: Bearer YOUR_API_KEY" http://localhost:8080/v1/hardware
```

should show your pi's CPU (like "Cortex-A76" for pi 5).

```bash
# test inference
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"qwen2.5","messages":[{"role":"user","content":"Say hello in 5 words"}]}'
```

**⏱️ first response will be slow** (~30-60 seconds on pi) because the model needs to load into memory. after that, responses are faster (~5-15 tokens/sec on pi 5, ~2-5 tokens/sec on pi 4).

---

## step 7: find your public URL

```bash
docker logs ai-gateway-tailscale 2>&1 | grep "Funnel"
```

or just check tailscale admin console — your pi will show up as `ai-gateway` and the funnel URL will be something like:

```
https://ai-gateway.your-tailnet.ts.net
```

test it from your phone or mac (any network):

```bash
curl https://ai-gateway.your-tailnet.ts.net/health
```

---

## step 8: use it from anywhere

from any openai-compatible app, set:

- **base URL**: `https://ai-gateway.your-tailnet.ts.net/v1`
- **API key**: the key from step 5
- **model**: `qwen2.5` (or whatever you set as MODEL_ALIAS)

python example:

```python
from openai import OpenAI

client = OpenAI(
    base_url="https://ai-gateway.your-tailnet.ts.net/v1",
    api_key="your-api-key"
)

response = client.chat.completions.create(
    model="qwen2.5",
    messages=[{"role": "user", "content": "Hello from my pi!"}]
)
print(response.choices[0].message.content)
```

---

## troubleshooting

**"model file not found" error in llama-server logs:**

```bash
docker logs ai-gateway-llama
```

check that MODEL_FILE in `.env` matches the actual filename in `models/`:

```bash
ls models/
```

**containers keep restarting:**

```bash
docker compose logs --tail 50
```

look for errors. most common: wrong model path, not enough RAM.

**pi runs out of memory:**

use a smaller model. or add swap:

```bash
sudo fallocate -l 4G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
# make it permanent:
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

**tailscale funnel not working:**

- make sure funnel is enabled in ACLs (step 4)
- make sure HTTPS certificates are enabled in tailscale DNS settings
- check logs: `docker logs ai-gateway-tailscale`

---

## keeping it running

the containers are set to `restart: unless-stopped`, so they'll survive reboots.

to update:

```bash
cd ai-gateway
git pull
docker compose up -d --build
```

to stop:

```bash
docker compose down
```

to check logs:

```bash
docker compose logs -f         # all logs, live
docker compose logs gateway    # just the gateway
docker compose logs llama-server  # just llama
```
