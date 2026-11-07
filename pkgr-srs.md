# Software Requirements Specification (SRS) for Pkgr

**System Name:** Pkgr  
**Document Version:** 3.0  
**Date:** November 7, 2026 (Updated)  

---

## 1. Introduction

### 1.1 Purpose
This document specifies the software requirements for Pkgr, an event-driven build automation and distribution system written in Go. It covers the central Mediator application (gRPC server hosting the `Dispatcher` service for client and worker connections, JSON blueprint parsing, artifact storage, and log streaming) and the containerized Worker application, with all communication utilizing gRPC and Protocol Buffers (`proto3`).

### 1.2 Document Conventions
* **REQ-ID:** Unique requirement identifier (e.g., REQ-MED-1, REQ-WRK-1, REQ-CLI-1, REQ-SEC-1, REQ-RPC-1, REQ-BLU-1).
* **Priority:** Categorized as High, Medium, or Low.

### 1.3 Intended Audience and Reading Suggestions
* **Developers & System Architects:** Focus on Sections 2, 3, and 4 to understand system architecture, entity responsibilities, user stories, JSON Blueprint structure, `Dispatcher` gRPC protocol design (`BuildTask`, `BuildResult`, `ArtifactChunk`), decoupled build scheduling, and worker log streaming.
* **DevOps & Infrastructure Engineers:** Focus on Operating Environment (Section 2.4), Communications Interfaces (Section 3.4), and Nonfunctional Requirements (Section 5).

### 1.4 Product Scope
Pkgr automates building software applications directly from source code using declarative JSON-based **Blueprints**. Clients communicate directly with the Mediator via gRPC to request software artifacts, stream real-time build logs, and receive compiled packages for installation. All communication between Clients, Mediator, and Worker nodes is standardized on gRPC (`proto3` schema `pkgr.Dispatcher`). It runs builds asynchronously according to blueprint instructions, streams real-time logs, handles cached artifact distribution, and delivers compiled binaries back to the client via chunked gRPC streams.

### 1.5 References
* IEEE Std 830-1998: IEEE Recommended Practice for Software Requirements Specifications.
* gRPC & Protocol Buffers Specification (v3).
* Docker Documentation & Void Linux Package Build Guidelines.

---

## 2. Overall Description

### 2.1 Product Perspective & Entity Roles
Pkgr is a distributed, event-driven build service designed to simplify compiling software from source and delivering binaries through a gRPC client interface. The system consists of three primary entities:

1. **Mediator Node:** Central hub running on an Oracle Free Tier VM that:
   * Hosts the `Dispatcher` gRPC service (`package pkgr`).
   * Receives artifact requests (`ArtifactRequest`) from clients.
   * Dispatches build tasks (`BuildTask`) to worker nodes over bidirectional gRPC streams (`ConnectWorker`).
   * Receives build logs (`build_log`) from worker nodes.
   * Receives compiled artifacts (`ArtifactChunk`) from worker nodes via chunked transfer streaming.
   * Forwards build logs and chunked artifacts (`BuildResult`) to clients via server streaming (`RequestArtifact`).
   * Stores and manages pre-built package artifacts.

2. **Client:** User-facing application/CLI that:
   * Issues `ArtifactRequest` messages to the Mediator via `RequestArtifact`.
   * Receives real-time build logs forwarded in `BuildResult` stream messages.
   * Receives binary packages streamed as `ArtifactChunk` messages and installs them locally.

3. **Worker Node:** Isolated execution unit running inside a Docker container on a Void Linux base image that:
   * Connects to the Mediator using `ConnectWorker` and listens for incoming `BuildTask` assignments.
   * Executes builds according to Blueprint specifications inside isolated containers.
   * Streams build logs (`build_log`) back to the Mediator within `BuildResult` messages.
   * Sends completed build artifacts to the Mediator via `ArtifactChunk` stream messages.

```
+-----------------------------------------------------------------------------------+  
|                                    Pkgr System                                    |  
|                                                                                   |  
|  +------------+             gRPC Client API               +--------------------+  |  
|  |   Client   | <---------------------------------------> |      Mediator      |  |  
|  +------------+       rpc RequestArtifact(...)            +--------------------+  |  
|                        (stream BuildResult)                         |             |  
|                                                                     | ConnectWorker  
|                                                                     | (stream)    |  
|                                                                     v             |  
|                                                           +--------------------+  |  
|                                                           |    Worker Node     |  |  
|                                                           |    (Void Linux)    |  |  
|                                                           +--------------------+  |  
+-----------------------------------------------------------------------------------+  
```

### 2.2 User Stories & Core Workflows

#### User Story 1: Cached Artifact Delivery
* **Scenario:** A user wants to install a package, and the Mediator already has a pre-built artifact stored for that package.
* **Workflow:**
  1. The **Client** invokes `RequestArtifact` passing an `ArtifactRequest` with the package name.
  2. The **Mediator** checks its storage and identifies an already built artifact for that package.
  3. The **Mediator** streams the cached artifact back to the **Client** as a series of `BuildResult` messages containing `ArtifactChunk` data, followed by `complete = true`.
  4. The **Client** reassembles the binary chunks and installs the package locally.

#### User Story 2: On-Demand Compilation & Delivery
* **Scenario:** A user wants to install a package, and the Mediator does not have a pre-built artifact available.
* **Workflow:**
  1. The **Client** invokes `RequestArtifact` passing an `ArtifactRequest` with the package name.
  2. The **Mediator** determines it does not have an already built artifact for that package.
  3. The **Mediator** converts the package's JSON Blueprint into a `BuildTask` message and sends it to an active **Worker Node** over its established `ConnectWorker` stream.
  4. The **Worker Node** receives `BuildTask` and starts building the artifact inside its isolated Void Linux container.
  5. The **Worker Node** streams real-time build logs back over `ConnectWorker` using `BuildResult.build_log`.
  6. The **Mediator** forwards the log stream to the **Client** over `RequestArtifact`.
  7. Upon compilation completion, the **Worker Node** streams the compiled package back over `ConnectWorker` using `BuildResult.artifact_chunk`.
  8. The **Mediator** stores the received artifact for future requests.
  9. The **Mediator** streams the artifact chunks to the **Client** over `RequestArtifact` and concludes with `complete = true`.
  10. The **Client** reassembles the package binary and installs it locally.

### 2.3 User Characteristics
* **End User / Developer:** Uses the client application to request package artifacts, inspect live build logs, and receive installed binaries on their local system.
* **System Administrator / Packager:** Authors and updates JSON Blueprint files, maintains the Mediator VM and gRPC services, and manages worker nodes.

### 2.4 Operating Environment
* **Mediator Host:** Oracle VM (VM.Standard.E2.1.Micro — 1 OCPU, 1 GB RAM, ARM/AMD) running the Mediator Go gRPC Server (`pkgr.Dispatcher`), Blueprint registry, and pre-built artifact storage.
* **Mediator Runtime:** Go runtime (Linux environment).
* **Worker Container:** Base image `voidlinux/voidlinux` running the worker Go application with the `pkgr.Dispatcher` gRPC client module.
* **Worker Execution Engine:** Docker engine on worker host machines.

### 2.5 Design and Implementation Constraints
* Developed entirely in Go (Golang) using Protocol Buffers (`proto3`) package `pkgr`.
* Blueprints MUST be written as valid JSON files exposing structured fields (`name`, `source`, `needs`, `binaries`, `steps`) matching the `BuildTask` Protobuf message structure.
* All communication between Clients, Mediator, and Worker nodes MUST exclusively use the gRPC `Dispatcher` service interface.
* All artifact transportation across the system MUST be handled via chunked streaming using `ArtifactChunk` messages to optimize memory footprint on constrained hardware.
* Build execution state MUST be fully decoupled from active client connection states.
* Mediator storage and memory consumption must remain strictly optimized to fit within VM.Standard.E2.1.Micro hardware limits (staying under 256 MB RAM).
* Worker execution environments must strictly use the official Void Linux base image.

### 2.6 User Documentation
* Blueprint authoring specification guide detailing how to write valid JSON blueprint files with source URLs, xbps dependency arrays, target binaries, and build steps.
* Client user guide detailing how to issue artifact requests, monitor log streams, and configure automatic local binary installations.
* Deployment guide for configuring the Mediator gRPC server and Worker nodes.

### 2.7 Assumptions and Dependencies
* Blueprint source URLs point to accessible tarball or ZIP archives.
* Worker nodes have outbound network access to retrieve source code archives specified in Blueprints and Void Linux dependencies (`xbps`).
* Worker nodes and clients can establish outbound gRPC connections (HTTP/2 over TLS) to the Mediator `Dispatcher` service.

---

## 3. External Interface Requirements

### 3.1 User Interfaces
* **gRPC Client Interface / CLI:** Programmatic or command-line client interacting with the Mediator gRPC `Dispatcher` API (`RequestArtifact`) to issue package artifact requests, receive streaming build logs, and download binary artifacts via chunked transfer streaming.

### 3.2 Hardware Interfaces
Standard TCP/IP networking interfaces over configured gRPC network ports (e.g., port 50051) for communication between Clients, Mediator, and Workers.

### 3.3 Software Interfaces
* **Blueprint Engine:** JSON parsing layer on Mediator/Worker that maps blueprint files directly to `BuildTask` Protobuf messages.
* **gRPC Framework & Protocol Buffers (`proto3`):** Strongly typed interface definition file (`pkgr.Dispatcher`) defining:
  * `rpc ConnectWorker (stream BuildResult) returns (stream BuildTask)`
  * `rpc RequestArtifact (ArtifactRequest) returns (stream BuildResult)`
  * Message types: `BuildTask`, `BuildResult` (`oneof content`), `ArtifactChunk`, and `ArtifactRequest`.
* **Go Runtime:** Execution engine for Mediator and Worker daemons (`option go_package = "github.com/wargesz/pkgr/proto;pb"`).
* **Docker Engine:** Container runtime environment for executing Void Linux workers.
* **Void Linux Package Manager (`xbps`):** Invoked inside worker containers for fetching build dependencies listed in `BuildTask.needs`.

### 3.4 Communications Interfaces
gRPC over TLS (HTTP/2) exclusively for all Client-to-Mediator and Worker-to-Mediator communication via the `Dispatcher` service:
* **Worker Connection & Task Stream:** `rpc ConnectWorker (stream BuildResult) returns (stream BuildTask)`
  * Worker opens a stream sending `BuildResult` messages and receiving `BuildTask` assignments.
  * Used for worker registration, real-time log transmission (`build_log`), chunked artifact uploading (`artifact_chunk`), and completion signaling (`complete`).
* **Client Request & Delivery Stream:** `rpc RequestArtifact (ArtifactRequest) returns (stream BuildResult)`
  * Client calls RPC with `ArtifactRequest(name)`.
  * Mediator responds with a stream of `BuildResult` messages multiplexing log messages, chunked binary payloads (`ArtifactChunk`), and final completion status (`complete`).

---

## 4. System Features

### 4.1 Feature 1: Blueprint Specification & Management
* **4.1.1 Description & Priority:** Blueprints are JSON files stored on the Mediator containing complete instructions for fetching, satisfying dependencies, compiling, and packaging an application on a Worker node. **Priority: High**
* **4.1.2 Structure & Fields:** Each Blueprint file is a valid JSON document that maps directly to the `BuildTask` Protobuf message schema, containing five required fields:
  * `name` (String / Protobuf field 1): Name of the package.
  * `source` (String / Protobuf field 2): URL link to the source file ZIP or tarball.
  * `needs` (Array of Strings / Protobuf field 3): Names of Void xbps packages specifying build dependencies.
  * `steps` (Array of Strings / Protobuf field 4): Build commands that need to be run in order to correctly build the package.
  * `binaries` (Array of Strings / Protobuf field 5): List of binaries the package installs.
* **4.1.3 Functional Requirements:**
  * **REQ-BLU-1:** The system SHALL define program build specifications using structured JSON files termed **Blueprints**.
  * **REQ-BLU-2:** A Blueprint file SHALL contain a `name` string field identifying the target package (`BuildTask.name`).
  * **REQ-BLU-3:** A Blueprint file SHALL contain a `source` string field specifying the source code URL (`BuildTask.source`).
  * **REQ-BLU-4:** A Blueprint file SHALL contain a `needs` array of strings specifying required xbps dependency packages (`BuildTask.needs`).
  * **REQ-BLU-5:** A Blueprint file SHALL contain a `steps` array of strings detailing sequential build commands (`BuildTask.steps`).
  * **REQ-BLU-6:** A Blueprint file SHALL contain a `binaries` array of strings listing target installed binaries (`BuildTask.binaries`).
  * **REQ-MED-4:** Mediator SHALL index and parse Blueprint files to generate `BuildTask` messages matching client package requests.

### 4.2 Feature 2: Artifact Request Handling & Delivery (User Story 1 & 2)
* **4.2.1 Description & Priority:** The Mediator processes incoming artifact requests from clients via `RequestArtifact`, serving pre-built cached artifacts immediately or dispatching builds to workers when no cached artifact exists. **Priority: High**
* **4.2.2 Functional Requirements:**
  * **REQ-MED-5:** Upon receiving an `ArtifactRequest` from a client, the Mediator SHALL check if a pre-built artifact is already stored for the requested package.
  * **REQ-MED-6:** If a pre-built artifact exists, the Mediator SHALL directly stream the artifact to the client via `BuildResult.artifact_chunk` messages without triggering a new build (User Story 1).
  * **REQ-MED-7:** If no pre-built artifact exists, the Mediator SHALL dispatch a `BuildTask` message over `ConnectWorker` to an available Worker node (User Story 2).
  * **REQ-MED-8:** The Mediator SHALL receive completed artifacts sent from Worker nodes via `BuildResult.artifact_chunk` streams, store them in Mediator storage for future requests, and forward them to the requesting client via `RequestArtifact`.
  * **REQ-CLI-1:** The Client SHALL send `ArtifactRequest` messages to the Mediator via gRPC and handle receiving, reassembling `ArtifactChunk` data, and installing the returned package binaries.

### 4.3 Feature 3: Live Build Log Streaming (User Story 2)
* **4.3.1 Description & Priority:** During on-demand builds, Worker nodes stream live build logs back to the Mediator within `BuildResult.build_log` payload wrappers, which forwards them directly to the client. **Priority: High**
* **4.3.2 Functional Requirements:**
  * **REQ-WRK-5:** The Worker node SHALL stream `stdout` and `stderr` build execution logs back to the Mediator in real time using `BuildResult.build_log`.
  * **REQ-MED-2:** The Mediator SHALL write incoming build log streams from Worker nodes into log storage and forward the log stream to the active client session over `RequestArtifact`.
  * **REQ-MED-3:** System SHALL decouple background build execution from active client connection states, allowing disconnected clients to re-attach to ongoing log streams.

### 4.4 Feature 4: Unified gRPC Communication Architecture
* **4.4.1 Description & Priority:** All communication between Clients, Worker nodes, and the Mediator is conducted exclusively via the `pkgr.Dispatcher` gRPC service over TLS. **Priority: High**
* **4.4.2 Functional Requirements:**
  * **REQ-RPC-1:** ALL communication between Clients, Mediator, and Worker nodes SHALL strictly use gRPC over HTTP/2 defined via the `pkgr.Dispatcher` Protocol Buffer schema.
  * **REQ-RPC-2:** Worker nodes SHALL register and receive build tasks using the bidirectional gRPC stream `rpc ConnectWorker (stream BuildResult) returns (stream BuildTask)`.
  * **REQ-RPC-3:** Worker nodes SHALL stream log output to the Mediator using `BuildResult.build_log` inside the `ConnectWorker` stream.
  * **REQ-RPC-4:** ALL artifact transportation across the system (Worker -> Mediator and Mediator -> Client) SHALL be handled via chunked file streaming using `ArtifactChunk` wrapped in `BuildResult.artifact_chunk`.
  * **REQ-RPC-5:** Job completion signaling SHALL be explicitly communicated across streams using the `BuildResult.complete` boolean flag.

### 4.5 Feature 5: Automated Containerized Compilation (Worker Node)
* **4.5.1 Description & Priority:** Isolated container execution on Worker nodes using Void Linux. **Priority: High**
* **4.5.2 Functional Requirements:**
  * **REQ-WRK-1:** Worker node SHALL run inside a container based on `voidlinux/voidlinux`.
  * **REQ-WRK-2:** Worker SHALL download source archives specified in `BuildTask.source`.
  * **REQ-WRK-3:** Worker SHALL install required package dependencies listed in `BuildTask.needs` using `xbps` prior to compilation.
  * **REQ-WRK-4:** Worker SHALL sequentially execute commands defined in `BuildTask.steps` inside the container environment.
  * **REQ-WRK-6:** Worker SHALL package successful build output and stream the completed artifact back to the Mediator as a sequence of `ArtifactChunk` messages wrapped in `BuildResult`.

---

## 5. Other Nonfunctional Requirements

### 5.1 Performance Requirements
* **Log Streaming Latency:** Log chunk transmission overhead (`build_log`) from Worker to Mediator to Client must add less than 50 ms latency.
* **Mediator Resource Management:** Memory consumption on the Oracle VM.Standard.E2.1.Micro instance must remain strictly under 256 MB RAM. Chunked transfer streaming via `ArtifactChunk` MUST be utilized to prevent buffer allocation spikes during large artifact transfers.

### 5.2 Safety Requirements
All compilation commands and Blueprint execution steps execute strictly inside isolated Docker containers on worker nodes to protect the Mediator host and client environment from arbitrary code execution.

### 5.3 Security Requirements
* Blueprint evaluation MUST validate JSON structures and sanitize inputs to prevent shell injection vulnerabilities inside worker containers.
* All gRPC connections between Clients, Workers, and Mediator MUST be encrypted using TLS.

### 5.4 Software Quality Attributes
* **Maintainability:** Declarative JSON Blueprints and strongly typed Protobuf messages (`BuildTask`) allow for straightforward maintenance and updates of software build recipes.
* **Interoperability:** Strongly typed Protocol Buffer interfaces (`pkgr.Dispatcher`) ensure seamless communication across all system entities.
* **Fault Tolerance:** Decoupled background builds ensure transient client disconnects do not interrupt active Worker compilation tasks.

---

## 6. Appendices

### Appendix A: Glossary & Schema Reference

* **`Dispatcher` Service:** The central gRPC service interface defining `ConnectWorker` for worker communication and `RequestArtifact` for client communication.
* **`BuildTask`:** Protobuf message sent from Mediator to Worker containing the compiled blueprint instructions (`name`, `source`, `needs`, `steps`, `binaries`).
* **`BuildResult`:** Protobuf message carrying a `oneof` payload (`build_log`, `artifact_chunk`, `complete`) for streaming logs, binary chunks, and job status.
* **`ArtifactChunk`:** Protobuf message representing a segment of binary package data (`filename`, `empty`, `bytes data`) used in chunked transfer streaming.
* **`ArtifactRequest`:** Protobuf message sent by Clients to request package artifacts by `name`.
* **Blueprint:** JSON file containing package build specifications: program name, archive source URL, needs array of xbps dependencies, binaries array of installed binaries, and sequential steps array of build commands.
* **Chunked Transfer:** The technique of streaming binary data in sequential chunks (`ArtifactChunk`) over a gRPC stream rather than loading entire binary files into memory, minimizing RAM usage on resource-constrained nodes.
* **Mediator:** Central Go server managing artifact requests, cached artifact storage, build dispatching to workers, log stream routing, and artifact distribution.
* **Client:** User application/CLI that requests artifacts, displays build log streams, and installs downloaded binaries.
* **Worker Node:** Containerized Go execution agent running on Void Linux that receives build requests with Blueprints, compiles source code, streams logs back to the Mediator, and returns finished artifacts.
