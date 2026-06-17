# what the hell is this?

a deployable gateway that can be deployed to a mac mini, raspberry pi, or any linux-based server. openai schema and all that stuff.

computationally running on the hardware (mac mini with macos, raspberry pi, x86 linux server, etc.), assume zero dependencies. 

llm inference runs on:
- mac metal
- cpu
- nvidia gpu
- apple neural engine

(why are we not having amd gpu u ask? well i dont got none)

the models should be downloaded locally and run locally.

it should be accessible thru any device on the internet. 

# tech decisions

web exposure: tailscale funnel (stable url, zero config, no domain needed, free)

llm runtime: llama.cpp (supports metal, cpu, nvidia. runs llama-server which already speaks openai schema)

programming language: go (single binary, zero runtime deps, matches the zero-fuss vibe)

deployment method: docker compose (one command to spin everything up)

# future enhancements

- mlx for mac (apple metal + neural engine support, runs natively outside docker. skipping for now to keep things simple)
- cloudflare tunnel as alternative to tailscale (custom domain, ddos protection, production-grade. more setup but nicer if sharing the endpoint with others)

(anecdote: i did a proj of a local rag chat application, distribution requires people to install nvidia drivers. that was windows but pain in the ass, never again. urghhh)


# rule for ai

this is a document written by thomas, and for thomas. not a typical readme. so u not overwrite it.

for areas with [tbd] or [idk], those are design decisions, if anything u can like uhhh fill those out (only with thomas explicit permissions).