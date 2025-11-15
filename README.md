# FabricX Developer Toolkit

FabricX is a developer toolkit designed to simplify building on Hyperledger Fabric. It provides a CLI, an SDK, and chaincode templates to abstract away the complexity of setting up, deploying to, and interacting with a Fabric network.

## Project Overview

The FabricX project aims to streamline Hyperledger Fabric development by offering a suite of tools that abstract away much of the underlying complexity. It enables developers to focus on chaincode logic and application development rather than intricate network configurations and low-level interactions.

The project consists of three main components:

1.  **A CLI (`fabricx`)**: Written in TypeScript, providing high-level commands for common Fabric operations.
2.  **An SDK (`@fabricx/sdk`)**: Also in TypeScript, offering a programmatic interface to interact with Fabric networks.
3.  **A Go Runtime Core (`core`)**: Handles the low-level interactions with Hyperledger Fabric, communicating with the TypeScript components via gRPC.

## Published Packages

The FabricX CLI and SDK are available on npm:

*   **CLI**: [`@fabricx/cli`](https://www.npmjs.com/package/@fabricx/cli)
*   **SDK**: [`@fabricx/sdk`](https://www.npmjs.com/package/@fabricx/sdk)

## Features

*   **Simplified Network Management**: Easily initialize, deploy, and tear down local Fabric networks.
*   **Chaincode Deployment**: Deploy chaincode templates with simple commands.
*   **Intuitive Interaction**: Invoke and query chaincode functions with clear, concise commands or programmatic calls.
*   **Monorepo Structure**: Organized into distinct packages for clear separation of concerns and easier development.
*   **gRPC Communication**: Efficient and robust communication between TypeScript and Go components.

## Getting Started

### Prerequisites

*   Node.js (>=18.0.0)
*   Go (>=1.25.3)
*   Docker and Docker Compose

### Installation

To get started with FabricX, clone the repository and install dependencies:

```bash
git clone https://github.com/temmyjay001/fabricx.git
cd fabricx
npm install # Installs dependencies for CLI and SDK
cd core
go mod download # Installs Go dependencies for the runtime
cd ..
```

### Building the Project

To build all components of the FabricX toolkit:

```bash
npm run build # Builds CLI and SDK
cd core
make build # Builds the Go runtime binary
cd ..
```

Alternatively, you can build the Docker image for the Go runtime:

```bash
cd core
make docker-build
cd ..
```

### Key Commands (CLI)

Once the CLI is built and linked (e.g., via `npm link` in the `cli` directory or by using `npx`), you can use the following commands:

*   **Initialize a local Fabric network:**
    ```bash
    npx fabricx init
    ```

*   **Deploy a chaincode template:**
    ```bash
    npx fabricx deploy <template_name>
    ```
    *Example:* `npx fabricx deploy settlement`

*   **Invoke a chaincode function:**
    ```bash
    npx fabricx invoke <chaincode> <function> --args '<args_json_array>'
    ```
    *Example:* `npx fabricx invoke settlement createTx --args '["TX001","BANK_A","BANK_B",1000]'`

*   **Query a chaincode function:**
    ```bash
    npx fabricx query <chaincode> <function> --args '<args_json_array>'
    ```
    *Example:* `npx fabricx query settlement getTx --args '["TX001"]'`

*   **Tear down the local network:**
    ```bash
    npx fabricx stop
    ```

## Development Conventions

*   **Tech Stack:**
    *   **CLI & SDK:** TypeScript, Node.js
    *   **Core Runtime:** Go
    *   **Containerization:** Docker, Docker Compose
    *   **Testing:** Jest (TypeScript) and Go's built-in testing suite.
*   **Communication:** The TypeScript and Go layers communicate via a gRPC bridge.
*   **Project Structure:**
    ```
    /
    ├── cli/        # TypeScript CLI
    ├── sdk/        # TypeScript SDK
    ├── core/       # Go Runtime Core
    └── templates/  # Chaincode templates
    ```

## Contributing

We welcome contributions to the FabricX Developer Toolkit! Please see our [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

This project is licensed under the Apache-2.0 License - see the [LICENSE](LICENSE) file for details.
