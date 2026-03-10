# Artifactory Automation 🚀

**A Enterprise-Grade Demonstration of Artifact Management for Multi-Language Microservices**

---

[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)](#)
[![Artifacts](https://img.shields.io/badge/artifacts-artifactory-orange.svg)](#)
[![Stack](https://img.shields.io/badge/stack-Java%20|%20Go%20|%20Docker-blue.svg)](#)

This repository provides a comprehensive blueprint for **publishing, managing, and consuming artifacts** within modern CI/CD pipelines. Using JFrog Artifactory as the central hub, it demonstrates a robust workflow for **Java**, **Go**, and **Docker-based microservices**, simulating a complex fintech ecosystem.

---

## 📖 Table of Contents

1. [Introduction](#-introduction)
2. [The Value of Artifact Management](#-the-value-of-artifact-management)
3. [Project Architecture](#-project-architecture)
4. [Getting Started](#-getting-started)
5. [Usage Guide](#-usage-guide)
6. [CI/CD Workflow](#-cicd-workflow)
7. [Jenkins Configuration](#-jenkins-configuration)
8. [Technology Stack](#-technology-stack)

---

## 🌟 Introduction

In high-scale DevOps environments, teams must share libraries and services seamlessly. Relying on local builds or manual transfers leads to version drift, security vulnerabilities, and "it works on my machine" inconsistencies.

This project leverages **Artifactory** to achieve:
- **Build Once, Deploy Anywhere**: Immutable artifacts that remain consistent across environments.
- **Centralized Governance**: A single source of truth for all binary dependencies.
- **Traceability**: Clear versioning and audit trails for every release.

**Core Workflow:**  
`Source Control → CI Build → Artifactory Publish → Downstream Consumption → Secure Deployment`

---

## 💡 The Value of Artifact Management

### Challenges Without Centralization
*   **Redundancy**: Teams manually rebuilding shared libraries, wasting time and resources.
*   **Version Fragmentation**: Inconsistent versions causing production outages.
*   **Security Risks**: No automated scanning or validation of 3rd party dependencies.

### Solutions With Artifactory
*   **Immutable Releases**: Once a version is published, it is never overwritten, ensuring reliability.
*   **Dependency Caching**: Faster build times by proxying and caching remote repositories.
*   **Multi-Format Support**: Native support for Java (Maven/Gradle), Go, Docker, Helm, and more.

---

## 🏗️ Project Architecture

```text
artifact-publishing-demo/
├── transaction-core/         # Java Shared Library (Gradle)
├── payment-service/          # Java Microservice (Consumes transaction-core)
├── user-auth-service/        # Java Microservice (Consumes transaction-core)
├── transaction-core-go/      # Go Shared Library
├── docker/                   # Containerization Scripts & Dockerfiles
├── scripts/                  # Automation & CI/CD Utility Scripts
│   ├── publish-artifact.sh   # Logic for pushing to Artifactory
│   └── fetch-artifact.sh     # Logic for pulling dependencies
└── demo-data/                # Mock datasets for simulation
```

---

## 🛠️ Getting Started

### Prerequisites
Ensure your environment meets the following requirements:
*   **Java 17+**: Required for Java-based modules.
*   **Go 1.20+**: Required for Go-based modules.
*   **Docker**: For building and running containerized images.
*   *(Optional)* **Jenkins/GitLab CI**: To explore automated pipeline integrations.

### Installation
```bash
# Clone the repository
git clone https://github.com/your-username/artifact-publishing-demo.git

# Navigate to the project root
cd artifact-publishing-demo
```

---

## 🚀 Usage Guide

### Java Ecosystem
1. **Build & Publish Shared Library:**
   ```bash
   cd transaction-core
   ./gradlew build
   ../scripts/publish-artifact.sh
   ```
2. **Consume in Microservices:**
   Add the following to your `build.gradle`:
   ```gradle
   dependencies {
       implementation 'global.citytech:transaction-core:1.0.0-demo'
   }
   ```

### Go Ecosystem
1. **Testing the Module:**
   ```bash
   cd transaction-core-go
   go test ./...
   ```
2. **Integration:**
   ```go
   import "github.com/demo/transaction-core-go"
   // Usage
   transactioncore.ProcessTransaction("TX123")
   ```

### Containerization
1. **Build the Payment Service Image:**
   ```bash
   cd docker
   ./build.sh
   ```
2. **Execute Deployment:**
   ```bash
   docker run -it --rm payment-service:demo
   ```

---

## 🔄 CI/CD Workflow

1.  **Commit**: Developers push code to the repository.
2.  **Continuous Integration**: Jenkins/GitLab CI triggers an automated build.
3.  **Artifact Generation**: The build produces a versioned binary (e.g., `1.0.0-demo`).
4.  **Publish**: The `publish-artifact.sh` script uploads the binary to the Artifactory repository.
5.  **Consumption**: Downstream services pull the precisely versioned dependency.
6.  **Deployment**: Docker builds the final production-ready image using the verified artifact.

---

## ⚙️ Jenkins Configuration

Below is a conceptual snippet of a `Jenkinsfile` used for this automation:

```groovy
pipeline {
    agent any
    stages {
        stage('Compile & Test') {
            steps {
                sh './gradlew build'
            }
        }
        stage('Publish to Artifactory') {
            steps {
                sh './scripts/publish-artifact.sh'
            }
        }
        stage('Cleanup') {
            steps {
                // Post-build cleanup to maintain agent health
                cleanWs()
            }
        }
    }
}
```

> [!NOTE]
> The `cleanWs()` step is critical in high-frequency CI environments to prevent disk saturation by removing transient build artifacts.

---

## 💻 Technology Stack

*   **Language Runtimes**: Java 17 (Gradle), Go 1.20
*   **Containerization**: Docker
*   **Artifact Management**: JFrog Artifactory (Simulation)
*   **Orchestration**: Jenkins / GitLab CI
*   **Data Serialization**: JSON

---

**Build Once • Publish Anywhere • Consume Securely**
