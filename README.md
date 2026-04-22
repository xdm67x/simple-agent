# Simple Agent

Simple Agent is an interactive general-purpose AI agent that runs locally on your computer, capable of executing software engineering tasks by taking direct action on your filesystem.

## Features

- **Tool Use**: The agent can interact with your system using a set of tools:
  - `bash`: Execute shell commands to explore the project, read files, run tests, and manage the codebase.
  - `ask_user`: Request clarification or input from the user when needed.
- **Local LLM Integration**: Powered by Ollama, allowing you to use various models (default: `gemma4:31b-cloud`).
- **Interactive Loop**: A real-time CLI interface with a "thinking" indicator and tool-call logging.
- **System Prompting**: Uses a dedicated system prompt to ensure the agent observes the environment before making assumptions.

## Project Structure

- `main.go`: Entry point of the application, initializes the agent and registers tools.
- `agent/`: Core logic for managing the LLM conversation, tool registration, and execution loops.
- `tools/`: Implementations of the available tools (`bash` and `ask_user`).
- `SYSTEM_PROMPT.md`: The system instructions that define the agent's behavior and goals.

## Getting Started

### Prerequisites

- [Ollama](https://ollama.ai/) installed and running.
- Go (Golang) installed.

### Running the Agent

1. Set the desired Ollama model (optional):
   ```bash
   export OLLAMA_MODEL=gemma4:31b-cloud
   ```

2. Run the agent:
   ```bash
   go run main.go
   ```

## How it Works

The agent follows an **Observe $\rightarrow$ Act $\rightarrow$ Review** cycle:
1. **Observe**: When asked about the project, it uses the `bash` tool to explore files and code.
2. **Act**: It performs the requested task (e.g., editing code, creating files).
3. **Review**: It verifies its changes by running build commands or tests before finalizing the response.
