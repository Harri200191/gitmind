# Models Directory

This directory is intended for storing LLM model files (e.g., `.gguf` files for llama.cpp).

## Recommended Models

For best results with commit message generation, consider these quantized models:

- **Meta-Llama-3.1-8B.Q4_K_M.gguf** (~4.7GB) - Good balance of quality and size
- **CodeLlama-7B-Instruct.Q4_K_M.gguf** (~3.8GB) - Code-focused model
- **Mistral-7B-Instruct-v0.3.Q4_K_M.gguf** (~4.1GB) - General purpose, good performance

## Download Sources

You can download these models from:
- [Hugging Face](https://huggingface.co/models?search=gguf)
- [TheBloke's repositories](https://huggingface.co/TheBloke)

## Configuration

After downloading a model, update your `.commitgen.yaml` file:

```yaml
model:
  enabled: true
  provider: llama.cpp
  model_path: ./models/your-model-name.gguf
  # ... other settings
```

## Note

Model files are excluded from git via `.gitignore` due to their large size.