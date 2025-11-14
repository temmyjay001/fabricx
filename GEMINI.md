# GEMINI.md: FabricX Developer Toolkit

This file provides an overview of the FabricX Developer Toolkit project, its goals, and how to interact with it.

## Project Overview

FabricX is a developer toolkit designed to simplify building on Hyperledger Fabric. It provides a CLI, an SDK, and chaincode templates to abstract away the complexity of setting up, deploying to, and interacting with a Fabric network.

The project consists of three main components:
1.  **A CLI (`fabricx`)** written in TypeScript for high-level commands.
2.  **An SDK (`@fabricx/sdk`)** also in TypeScript, which provides a programmatic interface to the toolkit.
3.  **A Go Runtime Core (`core`)** that handles the low-level interactions with Hyperledger Fabric.

The TypeScript and Go components communicate via gRPC.

## Building and Running

The project is not yet initialized. Based on the technical documentation, the following commands will be used for building and running the project.

### Key Commands

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
*   **Project Structure:** The project will be organized into separate packages for the CLI, SDK, and Go runtime.
    ```
    /
    ├── cli/
    ├── sdk/
    ├── core/
    └── templates/
    ```
