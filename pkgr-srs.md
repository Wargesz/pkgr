# Software Requirements Specification (SRS)

## Project Name: Pkgr

**Version:** 1.4

**Date:** November 1, 2026

## 1. Introduction

### 1.1 Purpose

This document specifies the software requirements for Pkgr, an event-driven build automation and distribution system written in Go. It covers the central Mediator application (SSH key authentication, TUI server, client binary SCP transfer, blueprint parsing, and decoupled log streaming) and the containerized Worker application, with all Worker-to-Mediator communication utilizing gRPC.

### 1.2 Document Conventions

* **REQ-ID:** Unique requirement identifier (e.g., `REQ-MED-1`, `REQ-WRK-1`, `REQ-CLI-1`, `REQ-SEC-1`, `REQ-RPC-1`, `REQ-BLU-1`).

* **Priority:** Categorized as High, Medium, or Low.

### 1.3 Intended Audience and Reading Suggestions

* **Developers & System Architects:** Focus on Sections 2, 3, and 4 to understand system architecture, Blueprint structure, SSH authentication, gRPC protocol design, decoupled build scheduling, SCP delivery, and worker log streaming.

* **DevOps & Infrastructure Engineers:** Focus on Operating Environment (Section 2.4), Communications Interfaces (Section 3.4), and Nonfunctional Requirements (Section 5).

### 1.4 Product Scope

Pkgr automates building software applications directly from source code using declarative Bash-based **Blueprints**. Authenticated users interact via a Terminal User Interface (TUI) over SSH. All communication between Mediator and Worker nodes is standardized on gRPC (bidirectional streaming for logs/heartbeats and RPCs for build jobs). It runs builds asynchronously according to blueprint instructions, streams real-time logs, handles client reconnection transparently, and delivers compiled binaries directly to client local machines via SCP for automatic installation.

### 1.5 References

* IEEE Std 830-1998: IEEE Recommended Practice for Software Requirements Specifications.

* gRPC & Protocol Buffers Specification (v3).

* Docker Documentation & Void Linux Package Build Guidelines.

* OpenSSH `ForceCommand` & `authorized_keys` Environment Specification.

## 2. Overall Description

### 2.1 Product Perspective

Pkgr is a distributed, event-driven build service designed to simplify compiling software from source and delivering binaries through an SSH terminal interface. It consists of:

1. **Mediator Node:** Central hub running on an Oracle Free Tier VM that authenticates clients via SSH public keys, enforces an interactive TUI interface (`ForceCommand`), stores and parses package **Blueprints**, exposes a gRPC server for worker nodes, tracks background build states, streams live logs to persistent files, and transfers finished binaries to client machines via SCP.

2. **Worker Node:** Docker container running a Go application on a Void Linux base image that connects to the Mediator via gRPC, parses dispatched Blueprint parameters, installs specified `xbps` dependencies (`needs`), executes build instructions (`steps`), streams live build logs back to the Mediator over a gRPC stream, and executes build steps independently of client SSH session states.

```
+-----------------------------------------------------------------------------------+  
|                                    Pkgr System                                    |  
|                                                                                   |  
|  +------------+       SSH Key Auth / TUI (`pkgr` user)    +--------------------+  |  
|  | Client SSH | <---------------------------------------> |      Mediator      |  |  
|  | Terminal   |     Tail Logs / SCP Artifact Download     | (Blueprints + VM)  |  |  
|  +------------+                                           +--------------------+  |  
|                                                                     |             |  
|                                                         gRPC Stream | Job Dispatch|  
|                                                         & Heartbeat | & Log Feed  |  
|                                                                     v             |  
|                                                           +--------------------+  |  
|                                                           |    Worker Node     |  |  
|                                                           |    (Void Linux)    |  |  
|                                                           +--------------------+  |  
+-----------------------------------------------------------------------------------+  

```

### 2.2 Product Functions

* Declarative software packaging specification via executable **Blueprint** files.

* SSH Public Key Authentication mapping client identities to package build requests.

* Interactive Terminal User Interface (TUI) with search filtering and package selection over SSH.

* Unified gRPC interface for all Mediator-to-Worker communication (job distribution, status heartbeats, and real-time log streaming).

* Decoupled asynchronous build execution: builds continue uninterrupted if SSH connection breaks.

* Automatic log re-attachment via `tail -f` when a client reconnects during an active build.

* Binary transfer via SCP to client local environment followed by client-side local installation upon completion.

* Real-time build log streaming from worker nodes over gRPC streams to persistent log files on the Mediator.

* Ephemeral processing and automatic deletion of custom flag build artifacts after transfer.

* Isolated containerized execution environment with Void Linux dependency installation.

### 2.3 User Classes and Characteristics

* **End User / Developer:** Authenticates via personal SSH public key (`ssh pkgr@<mediator-ip>`), uses the TUI to search/select packages defined by Blueprints, monitors build logs, and receives compiled binaries directly on their local machine via SCP.

* **System Administrator / Packager:** Authors and updates Blueprint files, manages client SSH public key authorization, configures `sshd` `ForceCommand`, maintains the Mediator VM and gRPC services, and manages worker nodes.

### 2.4 Operating Environment

* **Mediator Host:** Oracle VM (`VM.Standard.E2.1.Micro` — 1 OCPU, 1 GB RAM, ARM/AMD) running `sshd`, Blueprint storage, and Mediator gRPC Server.

* **Mediator Runtime:** Go runtime (Linux environment).

* **Worker Container:** Base image `voidlinux/voidlinux` running the worker Go application with a gRPC client module.

* **Worker Execution Engine:** Docker engine on worker host machines.

### 2.5 Design and Implementation Constraints

* Developed entirely in Go (Golang).

* Blueprints MUST be written as valid Bash files exposing structured metadata variables (`name`, `source`, `needs`, `steps`).

* All internal inter-node communication between Mediator and Workers MUST exclusively use gRPC defined via Protocol Buffers.

* OpenSSH configuration MUST enforce public key authentication and launch the Pkgr TUI via `ForceCommand` for user `pkgr`.

* Build execution state MUST be fully decoupled from active TUI frontend sessions.

* Mediator storage and memory consumption must remain strictly optimized to fit within `VM.Standard.E2.1.Micro` hardware limits.

* Worker environments must strictly use the official Void Linux base image.

### 2.6 User Documentation

* Blueprint authoring specification guide detailing how to write valid Bash blueprint files with source URLs, `xbps` dependency arrays, and build steps.

* Client SSH quickstart guide detailing SSH key registration, navigation hotkeys, and local SCP delivery/install behavior.

* Deployment guide for configuring OpenSSH public key management, Mediator daemon/gRPC server, and Worker nodes.

### 2.7 Assumptions and Dependencies

* Blueprint source URLs point to accessible tarball or ZIP archives.

* Client machine has SSH client and SCP utilities installed.

* Worker nodes have outbound network access to retrieve source code archives specified in Blueprints and Void Linux dependencies.

* Worker nodes can establish outbound gRPC connections (HTTP/2 over TLS) to the Mediator service.

## 3. External Interface Requirements

### 3.1 User Interfaces

* **SSH Key Authenticated TUI:** Upon SSH public key authentication, the server forces execution of the Pkgr interactive terminal UI.

* **TUI Viewports:** Search filter input field, blueprint/package list view with real-time status (Ready / Building / Queued), build log viewer (`tail -f` stream), and local installation completion screen.

### 3.2 Hardware Interfaces

Standard TCP/IP networking interfaces over port 22 (SSH/SCP) for user interactions and gRPC network port (e.g., port 50051) for communication between Mediator and Workers.

### 3.3 Software Interfaces

* **OpenSSH Daemon (`sshd`):** Authenticates client public keys and executes forced command wrapper.

* **Blueprint Engine:** Bash execution and parsing layer on Mediator/Worker to evaluate blueprint fields.

* **gRPC Framework & Protocol Buffers (proto3):** IPC/RPC protocol defining worker registration, job dispatching, heartbeat streams, log transmission services, and blueprint transmission.

* **SCP Command Utility:** Used to deliver completed build output archives directly to client local storage.

* **Go Runtime:** Execution engine for Mediator, TUI handler, and Worker daemons.

* **Docker Engine:** Container runtime environment for executing Void Linux workers.

* **Void Linux Package Manager (xbps):** Invoked inside worker containers for fetching build dependencies listed in a blueprint's `needs` field.

### 3.4 Communications Interfaces

* SSH/SCP (port 22) for client authentication, interactive TUI streaming, build log tailing, and secure binary delivery.

* gRPC over TLS (HTTP/2) exclusively for all Worker-to-Mediator and Mediator-to-Worker communication, including:

  * Worker registration & heartbeat streaming (`rpc RegisterWorker`, `rpc HeartbeatStream`).

  * Job & Blueprint dispatch (`rpc DispatchJob`, `rpc ReportBuildStatus`).

  * Real-time build log streaming (`rpc StreamBuildLogs`).

## 4. System Features

### 4.1 Feature 1: Blueprint Specification & Management

* **4.1.1 Description & Priority:** Blueprints are Bash files stored on the Mediator that store the complete instructions needed to fetch, satisfy dependencies for, and compile a program on a Worker node. **Priority: High**

* **4.1.2 Structure & Fields:**
  Each Blueprint file is a Bash script containing four required declaration fields:

  * `name` (String): The identifier name of the program being built.

  * `source` (String): The target URL of the source code archive (ZIP or tarball).

  * `needs` (Array of Strings): List of `xbps` package names that must be installed on the worker container prior to building.

  * `steps` (Array of Strings): Sequential compilation and build commands (e.g., `./configure`, `make`, `make install DESTDIR=...`).

* **4.1.3 Functional Requirements:**

  * **REQ-BLU-1:** The system SHALL define program build specifications using Bash files termed **Blueprints**.

  * **REQ-BLU-2:** A Blueprint file SHALL contain a `name` string field identifying the target program.

  * **REQ-BLU-3:** A Blueprint file SHALL contain a `source` string field specifying the URL of the source code ZIP or tarball.

  * **REQ-BLU-4:** A Blueprint file SHALL contain a `needs` array of strings specifying required `xbps` packages.

  * **REQ-BLU-5:** A Blueprint file SHALL contain a `steps` array of strings detailing sequential build and compilation commands.

  * **REQ-MED-4:** Mediator SHALL index and parse Blueprint files to serve the package catalog shown in the client TUI.

### 4.2 Feature 2: SSH Public Key Authentication & Forced TUI

* **4.2.1 Description & Priority:** Authenticates clients via SSH public keys and enforces the interactive package management TUI. **Priority: High**

* **4.2.2 Stimulus/Response Sequences:**

  1. Client -> Mediator: Initiates SSH connection (`ssh pkgr@<host>`).

  2. Mediator (`sshd`): Validates client SSH public key against authorized key registry.

  3. Mediator (`sshd`): Invokes forced command launch of Pkgr TUI application upon successful authentication.

  4. Client -> TUI: Searches and selects package blueprint target.

* **4.2.3 Functional Requirements:**

  * **REQ-SEC-1:** System SHALL authenticate clients connecting to the `pkgr` user profile strictly via SSH public key authentication.

  * **REQ-CLI-1:** System SHALL enforce execution of the Pkgr TUI upon successful SSH key authentication using OpenSSH `ForceCommand`.

  * **REQ-CLI-2:** TUI SHALL display a real-time searchable package list derived from available Blueprints with status indicators (e.g., `Available`, `Building`, `Ready`).

### 4.3 Feature 3: Asynchronous Build Management & Reconnect Handling

* **4.3.1 Description & Priority:** Decouples build jobs from active SSH sessions. Builds run independently in the background based on Blueprint instructions, and reconnecting clients automatically re-attach to live logs via `tail -f`. **Priority: High**

* **4.3.2 Stimulus/Response Sequences (Connection Drop & Reconnect):**

  1. Mediator -> Worker: Dispatches build job with Blueprint spec via gRPC; Worker begins compiling and streaming logs over gRPC.

  2. Client: Loses SSH connection or closes terminal while build is active.

  3. Mediator: Continues background build execution, writing incoming gRPC log frames into dedicated log file on VM disk.

  4. Client -> Mediator: Re-establishes SSH session.

  5. TUI -> Mediator: Identifies ongoing build associated with package request.

  6. Mediator -> Client: Automatically executes `tail -f` stream on active log file to resume displaying live build progress.

* **4.3.3 Functional Requirements:**

  * **REQ-MED-1:** System SHALL execute compilation build jobs asynchronously, maintaining build processing regardless of client SSH session state.

  * **REQ-MED-2:** Mediator SHALL write live log frames received over gRPC from workers into dedicated log files on disk.

  * **REQ-MED-3:** Mediator SHALL automatically attach reconnected client sessions to active package build log streams using a `tail -f` log tailing buffer.

### 4.4 Feature 4: Local SCP Artifact Delivery & Client Installation

* **4.4.1 Description & Priority:** Once an artifact build completes (or if pre-built), the Mediator transfers the output archive to the client's local computer via SCP and executes client local installation. **Priority: High**

* **4.4.2 Stimulus/Response Sequences:**

  1. Worker -> Mediator: Completes build and transfers final ZIP artifact archive over gRPC.

  2. Mediator: Signals completion to client session (or detects build finish upon client reconnect).

  3. Mediator -> Client: Triggers SCP binary transfer of artifact archive directly to client local environment.

  4. Client Session: Executes local installation hook on client computer once SCP transfer completes.

* **4.4.3 Functional Requirements:**

  * **REQ-CLI-3:** Mediator SHALL transfer completed build ZIP artifacts directly to the client's local machine via SCP protocol.

  * **REQ-CLI-4:** System SHALL trigger local binary installation routine on client host environment immediately following successful SCP download.

### 4.5 Feature 5: Unified gRPC Communication Architecture

* **4.5.1 Description & Priority:** All communication between Worker nodes and the Mediator is conducted exclusively via gRPC services over TLS. **Priority: High**

* **4.5.2 Functional Requirements:**

  * **REQ-RPC-1:** ALL communication between Mediator and Worker nodes SHALL strictly use gRPC over HTTP/2.

  * **REQ-RPC-2:** Worker nodes SHALL register and maintain connectivity with the Mediator using a gRPC bidirectional heartbeat stream.

  * **REQ-RPC-3:** Worker nodes SHALL stream stdout/stderr log output back to the Mediator in real time using a gRPC client-side/bidirectional streaming RPC (`StreamBuildLogs`).

  * **REQ-RPC-4:** Worker nodes SHALL stream or upload completed build artifacts back to the Mediator using gRPC chunked streaming.

### 4.6 Feature 6: Automated Source Compilation (Worker Node)

* **4.6.1 Description & Priority:** Isolated container execution, dependency installation via `xbps` from blueprint `needs`, executing blueprint `steps`, and streaming logs via gRPC. **Priority: High**

* **4.6.2 Functional Requirements:**

  * **REQ-WRK-1:** Worker node SHALL run inside a container based on `voidlinux/voidlinux`.

  * **REQ-WRK-2:** Worker SHALL download source archives specified in the Blueprint's `source` field.

  * **REQ-WRK-3:** Worker SHALL install required package dependencies listed in the Blueprint's `needs` array using `xbps` prior to compilation.

  * **REQ-WRK-4:** Worker SHALL sequentially execute commands defined in the Blueprint's `steps` array inside the container environment.

  * **REQ-WRK-5:** Worker SHALL stream live execution logs back to Mediator continuously via gRPC throughout compilation steps.

  * **REQ-WRK-6:** Worker SHALL package successful build output into a ZIP archive and return it to Mediator over gRPC.

## 5. Other Nonfunctional Requirements

### 5.1 Performance Requirements

* **Log Tailing & Reconnect Latency:** Live log stream output via `tail -f` must resume rendering within 200 ms of client SSH session re-attachment.

* **gRPC Overhead:** Streaming log chunk overhead over gRPC must add less than 50 ms latency to real-time display.

* **Mediator Resource Management:** Memory consumption on the Oracle `VM.Standard.E2.1.Micro` instance must remain strictly under 256 MB RAM.

### 5.2 Safety Requirements

All compilation commands and Blueprint execution steps execute strictly inside isolated Docker containers on worker nodes to isolate the Mediator host and SSH daemon from arbitrary code execution.

### 5.3 Security Requirements

* OpenSSH daemon MUST enforce public key authentication for user `pkgr`.

* OpenSSH configuration MUST enforce executable TUI execution via `ForceCommand /usr/local/bin/pkgr-tui`.

* Blueprint evaluation MUST sanitize inputs to prevent shell injection vulnerabilities when executing blueprint instructions inside worker containers.

* All gRPC connections between Workers and Mediator MUST be encrypted using TLS with mutual TLS (mTLS) authentication to verify worker node identity.

* Client-side SCP downloads must strictly validate target destination paths to prevent path traversal issues.

### 5.4 Software Quality Attributes

* **Maintainability & Declarative Packaging:** Defining build parameters via standardized Bash Blueprints enables clean, versionable package recipes.

* **Interoperability & Protocol Standardization:** Using gRPC guarantees strongly typed interfaces via Protocol Buffers between Mediator and Worker daemons.

* **Fault Tolerance & Resilience:** Uninterrupted background build processing ensures client disconnects or network instability will not corrupt or abort active compilation tasks.

* **Usability:** Zero-configuration local installation; clients authenticate via SSH keys and seamlessly receive installed software binaries on their local system.

## 6. Appendices

### Appendix A: Glossary

* **Blueprint:** Bash file containing package specifications: program `name`, archive `source` URL, `needs` array of xbps dependencies, and sequential `steps` array of build commands.

* **Mediator:** Central Go server managing SSH key auth, TUI sessions, Blueprint index, asynchronous build states, gRPC server, log file piping, and SCP transfers.

* **gRPC:** High-performance, open-source universal RPC framework used for all Worker-to-Mediator interactions.

* **Protocol Buffers (proto3):** Google's language-neutral mechanism for serializing structured data.

* **ForceCommand:** OpenSSH directive enforcing specific program execution upon user authentication.

* **SCP (Secure Copy Protocol):** Network protocol used for transferring files securely between host systems.

* **Tail -f:** Streamed file reading mechanism used to display ongoing log output in real time.

* **Worker:** Containerized Go execution agent running on Void Linux that evaluates Blueprint parameters and communicates with Mediator via gRPC.