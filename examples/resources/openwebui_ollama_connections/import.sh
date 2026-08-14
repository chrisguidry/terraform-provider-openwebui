# The Ollama connection list is one configuration block, so this resource is a
# singleton. Import it under its fixed id, and every connection comes with it.
# The API may return the connection keys masked, so confirm them after the
# import.
terraform import openwebui_ollama_connections.example ollama
