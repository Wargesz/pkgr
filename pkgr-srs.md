# Software Requirements Specification (SRS)

## Project Name: Pkgr

**Version:** 2.1

**Date:** November 7, 2026 (Updated)

## 1. Introduction

### 1.1 Purpose

This document specifies the software requirements for Pkgr, an event-driven build automation and distribution system written in Go. It covers the central Mediator application (gRPC server for client and worker connections, JSON blueprint parsing, artifact storage, and log streaming) and the containerized Worker application, with all communication utilizing gRPC.

### 1.2 Document Conventions

* **REQ-ID:** Unique requirement identifier (e.g., `REQ-MED-1`, `REQ-WRK-1`, `REQ-CLI-1`, `REQ-SEC-1`, `REQ-RPC-1`, `REQ-BLU-1`).
* **Priority:** Categorized as High, Medium, or Low.

### 1.3 Intended Audience and Reading Suggestions

* **Developers & System Architects:** Focus on Sections 2, 3, and 4 to understand system architecture, entity responsibilities, user stories, JSON Blueprint structure, gRPC protocol design, decoupled build scheduling, and worker log streaming.
* **DevOps & Infrastructure Engineers:** Focus on Operating Environment (Section 2.4), Communications Interfaces (Section 3.4), and Nonfunctional Requirements (Section 5).

### 1.4 Product Scope

Pkgr automates building software applications directly from source code using declarative JSON-based **Blueprints**. Clients communicate directly with the Mediator via gRPC to request software artifacts, stream real-time build logs, and receive compiled packages for installation. All communication between Clients, Mediator, and Worker nodes is standardized on gRPC. It runs builds asynchronously according to blueprint instructions, streams real-time logs, handles cached artifact distribution, and delivers compiled binaries back to the client via gRPC streams.

### 1.5 References

* IEEE Std 830-1998: IEEE Recommended Practice for Software Requirements Specifications.
* gRPC & Protocol Buffers Specification (v3).
* Docker Documentation & Void Linux Package Build Guidelines.

## 2. Overall Description

### 2.1 Product Perspective & Entity Roles

Pkgr is a distributed, event-driven build service designed to simplify compiling software from source and delivering binaries through a gRPC client interface. The system consists of three primary entities:

1. **Mediator Node:** Central hub running on an Oracle Free Tier VM that:
* Receives artifact requests from clients.
* Sends build requests to worker nodes.
* Receives build logs from worker nodes.
* Receives compiled artifacts from worker nodes.
* Sends build logs to clients.
* Stores and sends artifacts to clients.


2. **Client:** User-facing application/CLI that:
* Sends artifact requests to the Mediator.
* Receives build logs from the Mediator.
* Receives build artifacts from the Mediator and installs them locally.


3. **Worker Node:** Isolated execution unit running inside a Docker container on a Void Linux base image that:
* Receives build requests from the Mediator (including Blueprint specs).
* Executes builds and streams build logs back to the Mediator.
* Sends completed build artifacts to the Mediator.



```
+-----------------------------------------------------------------------------------+  
|                                    Pkgr System                                    |  
|                                                                                   |  
|  +------------+             gRPC Client API               +--------------------+  |  
|  |   Client   | <---------------------------------------> |      Mediator      |  |  
|  +------------+   Artifact Request / Build Logs Feed /   +--------------------+  |  
|                             Artifact Download                       |             |  
|                                                                     | gRPC Stream |  
|                                                                     | & Heartbeat |  
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
1. The **Client** sends an artifact request for a package to the **Mediator**.
2. The **Mediator** checks its storage and identifies an already built artifact for that package.
3. The **Mediator** sends the cached artifact back to the **Client**.
4. The **Client** receives and installs the package locally.



#### User Story 2: On-Demand Compilation & Delivery

* **Scenario:** A user wants to install a package, and the Mediator does not have a pre-built artifact available.
* **Workflow:**
1. The **Client** sends an artifact request for a package to the **Mediator**.
2. The **Mediator** determines it does not have an already built artifact for that package.
3. The **Mediator** sends a build request to an available **Worker Node**, passing the JSON Blueprint responsible for that package.
4. The **Worker Node** starts building the artifact inside its isolated Void Linux container according to the Blueprint.
5. The **Worker Node** streams real-time build logs back to the **Mediator**.
6. The **Mediator** forwards the build logs it receives from the Worker Node to the **Client**.
7. Upon compilation completion, the **Worker Node** sends the built artifact to the **Mediator**.
8. The **Mediator** stores the received artifact for future requests.
9. The **Mediator** sends the artifact to the **Client**.
10. The **Client** receives and installs the package locally.



### 2.3 User Characteristics

* **End User / Developer:** Uses the client application to request package artifacts, inspect live build logs, and receive installed binaries on their local system.
* **System Administrator / Packager:** Authors and updates JSON Blueprint files, maintains the Mediator VM and gRPC services, and manages worker nodes.

### 2.4 Operating Environment

* **Mediator Host:** Oracle VM (`VM.Standard.E2.1.Micro` — 1 OCPU, 1 GB RAM, ARM/AMD) running the Mediator gRPC Server, Blueprint registry, and pre-built artifact storage.
* **Mediator Runtime:** Go runtime (Linux environment).
* **Worker Container:** Base image `voidlinux/voidlinux` running the worker Go application with a gRPC client module.
* **Worker Execution Engine:** Docker engine on worker host machines.

### 2.5 Design and Implementation Constraints

* Developed entirely in Go (Golang).
* Blueprints MUST be written as valid JSON files exposing structured fields (`name`, `source`, `needs`, `binaries`, `steps`).
* All communication between Clients, Mediator, and Worker nodes MUST exclusively use gRPC defined via Protocol Buffers.
* Build execution state MUST be fully decoupled from active client connection states.
* Mediator storage and memory consumption must remain strictly optimized to fit within `VM.Standard.E2.1.Micro` hardware limits.
* Worker execution environments must strictly use the official Void Linux base image.

### 2.6 User Documentation

* Blueprint authoring specification guide detailing how to write valid JSON blueprint files with source URLs, `xbps` dependency arrays, target binaries, and build steps.
* Client user guide detailing how to issue artifact requests, monitor log streams, and configure automatic local binary installations.
* Deployment guide for configuring the Mediator gRPC server and Worker nodes.

### 2.7 Assumptions and Dependencies

* Blueprint source URLs point to accessible tarball or ZIP archives.
* Worker nodes have outbound network access to retrieve source code archives specified in Blueprints and Void Linux dependencies (`xbps`).
* Worker nodes and clients can establish outbound gRPC connections (HTTP/2 over TLS) to the Mediator service.

## 3. External Interface Requirements

### 3.1 User Interfaces

* **gRPC Client Interface / CLI:** Programmatic or command-line client interacting with the Mediator gRPC API to issue package artifact requests, receive streaming logs, and receive artifact downloads.

### 3.2 Hardware Interfaces

Standard TCP/IP networking interfaces over configured gRPC network ports (e.g., port 50051) for communication between Clients, Mediator, and Workers.

### 3.3 Software Interfaces

* **Blueprint Engine:** JSON parsing layer on Mediator/Worker to parse blueprint fields.
* **gRPC Framework & Protocol Buffers (proto3):** IPC/RPC protocol defining client requests, worker registration, job dispatching, heartbeat streams, log transmission services, and artifact transfers.
* **Go Runtime:** Execution engine for Mediator and Worker daemons.
* **Docker Engine:** Container runtime environment for executing Void Linux workers.
* **Void Linux Package Manager (xbps):** Invoked inside worker containers for fetching build dependencies listed in a blueprint's `needs` field.

### 3.4 Communications Interfaces

* gRPC over TLS (HTTP/2) exclusively for all Client-to-Mediator and Worker-to-Mediator communication, including:
* Client artifact requests & download streaming (`rpc RequestArtifact`, `rpc DownloadArtifact`).
* Worker registration & heartbeat streaming (`rpc RegisterWorker`, `rpc HeartbeatStream`).
* Job & Blueprint dispatch (`rpc DispatchJob`, `rpc ReportBuildStatus`).
* Real-time build log streaming from Worker -> Mediator -> Client (`rpc StreamBuildLogs`).
* Artifact upload from Worker -> Mediator (`rpc UploadArtifact`).



## 4. System Features

### 4.1 Feature 1: Blueprint Specification & Management

* **4.1.1 Description & Priority:** Blueprints are JSON files stored on the Mediator containing complete instructions for fetching, satisfying dependencies, compiling, and packaging an application on a Worker node. **Priority: High**
* **4.1.2 Structure & Fields:**
Each Blueprint file is a valid JSON document containing five required fields:
* `name` (String): Name of the package.
* `source` (String): URL link to the source file ZIP or tarball.
* `needs` (Array of Strings): Names of Void `xbps` packages specifying build dependencies.
* `binaries` (Array of Strings): List of binaries the package installs.
* `steps` (Array of Strings): Build commands that need to be run in order to correctly build the package.


* **4.1.3 Functional Requirements:**
* **REQ-BLU-1:** The system SHALL define program build specifications using structured JSON files termed **Blueprints**.
* **REQ-BLU-2:** A Blueprint file SHALL contain a `name` string field identifying the target package.
* **REQ-BLU-3:** A Blueprint file SHALL contain a `source` string field specifying the source code URL.
* **REQ-BLU-4:** A Blueprint file SHALL contain a `needs` array of strings specifying required `xbps` dependency packages.
* **REQ-BLU-5:** A Blueprint file SHALL contain a `binaries` array of strings listing target installed binaries.
* **REQ-BLU-6:** A Blueprint file SHALL contain a `steps` array of strings detailing sequential build commands.
* **REQ-MED-4:** Mediator SHALL index and parse Blueprint files to match client package requests to specific build specifications.



### 4.2 Feature 2: Artifact Request Handling & Delivery (User Story 1 & 2)

* **4.2.1 Description & Priority:** The Mediator processes incoming artifact requests from clients, serving pre-built cached artifacts immediately or dispatching builds to workers when no cached artifact exists. **Priority: High**
* **4.2.2 Functional Requirements:**
* **REQ-MED-5:** Upon receiving an artifact request from a client, the Mediator SHALL check if a pre-built artifact is already stored for the requested package.
* **REQ-MED-6:** If a pre-built artifact exists, the Mediator SHALL directly send the artifact to the client via gRPC streaming without triggering a new build (User Story 1).
* **REQ-MED-7:** If no pre-built artifact exists, the Mediator SHALL dispatch a build request containing the corresponding package Blueprint to an available Worker node (User Story 2).
* **REQ-MED-8:** The Mediator SHALL receive completed artifacts sent from Worker nodes, store them in Mediator storage for future requests, and forward them to the requesting client.
* **REQ-CLI-1:** The Client SHALL send artifact requests to the Mediator via gRPC and handle receiving and installing the returned package binaries.



### 4.3 Feature 3: Live Build Log Streaming (User Story 2)

* **4.3.1 Description & Priority:** During on-demand builds, Worker nodes stream live build logs back to the Mediator, which forwards them directly to the client. **Priority: High**
* **4.3.2 Functional Requirements:**
* **REQ-WRK-5:** The Worker node SHALL stream stdout and stderr build execution logs back to the Mediator in real time throughout compilation.
* **REQ-MED-2:** The Mediator SHALL write incoming build log streams from Worker nodes into log storage and forward the log stream to the active client session.
* **REQ-MED-3:** System SHALL decouple background build execution from active client connection states, allowing disconnected clients to re-attach to ongoing log streams.



### 4.4 Feature 4: Unified gRPC Communication Architecture

* **4.4.1 Description & Priority:** All communication between Clients, Worker nodes, and the Mediator is conducted exclusively via gRPC services over TLS. **Priority: High**
* **4.4.2 Functional Requirements:**
* **REQ-RPC-1:** ALL communication between Clients, Mediator, and Worker nodes SHALL strictly use gRPC over HTTP/2.
* **REQ-RPC-2:** Worker nodes SHALL register and maintain connectivity with the Mediator using a gRPC bidirectional heartbeat stream.
* **REQ-RPC-3:** Worker nodes SHALL stream log output to the Mediator using gRPC streaming RPCs.
* **REQ-RPC-4:** Worker nodes and Mediator SHALL transfer build artifacts using gRPC chunked file streaming.



### 4.5 Feature 5: Automated Containerized Compilation (Worker Node)

* **4.5.1 Description & Priority:** Isolated container execution on Worker nodes using Void Linux. **Priority: High**
* **4.5.2 Functional Requirements:**
* **REQ-WRK-1:** Worker node SHALL run inside a container based on `voidlinux/voidlinux`.
* **REQ-WRK-2:** Worker SHALL download source archives specified in the Blueprint's `source` field.
* **REQ-WRK-3:** Worker SHALL install required package dependencies listed in the Blueprint's `needs` array using `xbps` prior to compilation.
* **REQ-WRK-4:** Worker SHALL sequentially execute commands defined in the Blueprint's `steps` array inside the container environment.
* **REQ-WRK-6:** Worker SHALL package successful build output and send the completed artifact to the Mediator over gRPC.



## 5. Other Nonfunctional Requirements

### 5.1 Performance Requirements

* **Log Streaming Latency:** Log chunk transmission overhead from Worker to Mediator to Client must add less than 50 ms latency.
* **Mediator Resource Management:** Memory consumption on the Oracle `VM.Standard.E2.1.Micro` instance must remain strictly under 256 MB RAM.

### 5.2 Safety Requirements

All compilation commands and Blueprint execution steps execute strictly inside isolated Docker containers on worker nodes to protect the Mediator host and client environment from arbitrary code execution.

### 5.3 Security Requirements

* Blueprint evaluation MUST validate JSON structures and sanitize inputs to prevent shell injection vulnerabilities inside worker containers.
* All gRPC connections between Clients, Workers, and Mediator MUST be encrypted using TLS.

### 5.4 Software Quality Attributes

* **Maintainability:** Declarative JSON Blueprints allow for straightforward maintenance and updates of software build recipes.
* **Interoperability:** Strongly typed Protocol Buffer interfaces ensure seamless communication across all system entities.
* **Fault Tolerance:** Decoupled background builds ensure transient client disconnects do not interrupt active Worker compilation tasks.

## 6. Appendices

### Appendix A: Glossary

* **Blueprint:** JSON file containing package build specifications: program `name`, archive `source` URL, `needs` array of xbps dependencies, `binaries` array of installed binaries, and sequential `steps` array of build commands.
* **Mediator:** Central Go server managing artifact requests, cached artifact storage, build dispatching to workers, log stream routing, and artifact distribution.
* **Client:** User application/CLI that requests artifacts, displays build log streams, and installs downloaded binaries.
* **Worker Node:** Containerized Go execution agent running on Void Linux that receives build requests with Blueprints, compiles source code, streams logs back to the Mediator, and returns finished artifacts.
