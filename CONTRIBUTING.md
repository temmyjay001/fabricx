# Contributing to FabricX

We welcome contributions to the FabricX Developer Toolkit! By contributing, you help us improve and grow the project. Please take a moment to review this document to understand how to contribute effectively.

## Code of Conduct

This project adheres to the Contributor Covenant Code of Conduct. By participating, you are expected to uphold this code. Please report unacceptable behavior to [your email or contact method].

## How to Contribute

### Bug Reports

If you find a bug, please open an issue on our [GitHub Issues page](https://github.com/temmyjay001/fabricx/issues). When reporting a bug, please include:

*   A clear and concise description of the bug.
*   Steps to reproduce the behavior.
*   Expected behavior.
*   Screenshots or error messages if applicable.
*   Your operating system and FabricX version.

### Feature Requests

We love new ideas! If you have a feature request, please open an issue on our [GitHub Issues page](https://github.com/temmyjay001/fabricx/issues). Describe your idea clearly and explain why it would be beneficial to the project.

### Pull Requests

We welcome pull requests for bug fixes, new features, and improvements. To submit a pull request:

1.  **Fork the repository** and clone it to your local machine.
2.  **Create a new branch** from `main` for your changes: `git checkout -b feature/your-feature-name` or `git checkout -b bugfix/your-bug-fix`.
3.  **Make your changes**, ensuring they adhere to the project's coding style and conventions.
4.  **Write tests** for your changes.
5.  **Run tests** to ensure everything passes: `npm test` (for JS/TS) and `cd core && make test` (for Go).
6.  **Update documentation** if your changes affect how the project is used.
7.  **Commit your changes** using clear and descriptive commit messages (see "Commit Message Guidelines" below).
8.  **Push your branch** to your forked repository.
9.  **Open a pull request** to the `main` branch of the original repository.

## Development Setup

### Prerequisites

*   Node.js (>=18.0.0)
*   Go (>=1.25.3)
*   Docker and Docker Compose
*   Git

### Local Setup

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/temmyjay001/fabricx.git
    cd fabricx
    ```
2.  **Install Node.js dependencies:**
    ```bash
    npm install
    ```
3.  **Install Go dependencies:**
    ```bash
    cd core
    go mod download
    cd ..
    ```
4.  **Build the project:**
    ```bash
    npm run build # Builds CLI and SDK
    cd core
    make build # Builds Go runtime binary
    cd ..
    ```

## Testing

*   **TypeScript (CLI/SDK):**
    ```bash
    npm test # From the root directory
    ```
*   **Go (Core Runtime):**
    ```bash
    cd core
    make test # Runs unit tests
    make test-integration # Runs integration tests
    cd ..
    ```

## Commit Message Guidelines

We follow the Conventional Commits specification. This helps us with automated changelog generation and semantic versioning.

Examples:

*   `feat(scope): Add new feature`
*   `fix(scope): Fix bug in module`
*   `docs(scope): Update documentation`
*   `chore(scope): Update dependencies`

## Licensing

By contributing your code, you agree to license your contributions under the [Apache-2.0 License](LICENSE).
