# Failover Routing

Route `deepseek-chat` requests to Deepseek (75%) and local Ollama (25%) by weight.

## How it works

`routes.yaml` defines two targets for the same model with a `weight` strategy:

```yaml
routes:
  - id: deepseek-chat
    match:
      model: deepseek-chat
    strategy: weight
    targets:
      - provider: deepseek-main
        model: deepseek-chat
        weight: 3
      - provider: ollama-local
        model: qwen2.5:0.5b
        weight: 1
```

Requests are distributed randomly: 3 out of 4 go to Deepseek, 1 out of 4 to Ollama. If Deepseek is unavailable, all requests fall through to Ollama.

## Try it

1. Set up providers in `configs/providers.yaml`
2. Copy `routes.yaml` to `configs/routes.yaml`
3. Run Strait and send requests to `/v1/chat/completions` with `model: deepseek-chat`
