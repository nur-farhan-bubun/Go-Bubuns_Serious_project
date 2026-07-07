```markdown
# 📝 README_AI.md - System Constraints & Architectural Guidelines

> **Attention AI:** You are building a highly scalable, low-latency, and memory-optimized WebSocket Chat & Real-Time Mapping Service in Go. You must adhere strictly to the following step-by-step implementation phases. Do not write lazy, unoptimized, or suboptimal boilerplate code.

---

## ⚡ Phase 1: WebSocket Network & Latency Optimization

To achieve sub-millisecond real-time updates on client maps and chats, apply these network layer optimizations first:

### 1.1 Tuned Upgrader Buffers & Conditional Compression
* Configure explicitly managed read and write buffer sizes on the `websocket.Upgrader`:
  * `ReadBufferSize: 4096`
  * `WriteBufferSize: 4096`
* Avoid default large allocations to drastically minimize memory spikes during massive concurrent user handshakes.
* **Do not enable per-message compression by default.** Compression heavily increases CPU utilization and latency. Only enable it if average payloads exceed several kilobytes and network bandwidth is the proven system bottleneck.

### 1.2 Zero JSON Re-serialization Overhead (Fan-out Rule)
* **Never repeatedly marshal the same payload during a fan-out operation.** 
* Messages may originate from Kafka, a database, or internal system events. Regardless of the source, invoke `json.Marshal()` exactly **once** before entering the broadcast path. 
* Reuse the serialized `[]byte` slice for every recipient in the room. Marshalling once is perfectly fine; marshalling $N$ times for $N$ clients is strictly forbidden.

### 1.3 Strict Network Write Deadlines & Backpressure Policy
* Before pushing any data to a client's socket connection, always set a strict network write deadline using `conn.SetWriteDeadline(time.Now().Add(writeWait))`.
* **The central Hub must never block indefinitely on a slow client.** You must implement an explicit backpressure policy:
  * If a client's outbound transmission queue (`send` channel) is full, the Hub must either **disconnect the client** immediately or **drop the oldest message** to preserve system-wide throughput.

### 1.4 Robust Heartbeat Support (Ping/Pong)
* Implement continuous heartbeat support to detect dead or idle connections:
  * Set a strict `ReadDeadline` on the connection.
  * Register a custom `PongHandler` that extends the read deadline upon receiving a valid pong frame.
  * Run a dedicated background ticker that transmits a `Ping` message to the client every 30 seconds.
  * Automatically disconnect idle or unresponsive clients that violate the read deadline window.

---

## 💾 Phase 2: Zero-Allocation & Memory Management

To keep the service lightweight and protect it from intensive Garbage Collection (GC) pauses:

### 2.1 Shard Room Ownership (Contention Relief)
* **Avoid a single global mutex protecting every room in the application state.** 
* Partition your room maps into independent, localized **shards** (e.g., using a hashed room ID string modulo a fixed bucket size). This reduces lock contention significantly when hundreds of users are actively interacting across separate rooms concurrently.

### 2.2 Pre-allocated Slices and Maps
* When aggregating client lists, area subscribers, or active room sessions, always check if the capacity is predictable.
* If capacity is known, pre-allocate maps and slices using `make([]Type, 0, capacity)` to eliminate dynamic under-the-hood slice resizing and memory copying overhead.

### 2.3 Strict `sync.Pool` Usage
* Use `sync.Pool` **only** for frequently allocated, short-lived temporary objects (like high-frequency inbound chat packet structs or temporary byte buffers).
* **Do not pool:** Long-lived objects, massive data buffers that rarely repeat, or stateful structs. 

### 2.4 Pragmatic Escape Analysis (Pointers vs. Values)
* Prefer **value types** for small, immutable data structures to allow the Go compiler's escape analysis to store them cleanly on the stack instead of throwing them onto the heap.
* Use **pointers** exclusively for mutable shared states, massive structs, and long-lived connection objects. Avoid unnecessary pointer indirections that complicate memory layouts.

---

## 🧵 Phase 3: Safe Goroutine Lifecycle & Channel Management

To guarantee thread safety and eliminate memory leaks across long-lived network loops:

### 3.1 Strict Single Writer Rule (Concurrency Guard)
* Guarantee that **exactly one** dedicated Goroutine handles writing tasks (`conn.WriteMessage`) to any single WebSocket connection instance. Concurrent writes to a Gorilla WebSocket instance will throw a native runtime panic.
* Every connected client must possess an isolated outbound buffered channel (e.g., `send chan []byte 256`) fueled exclusively by its single write loop. The buffer provides a safety cushion against minor network blips.

### 3.2 Channel Ownership & Preventing Ghost Leaks
* **The Central Hub owns channel closure.** Never close application or client channels from multiple disparate goroutines.
* When a client disconnects, clean up resources aggressively. Close the client's dedicated `send` channel from the appropriate owner, invoke deferred teardown functions (`defer conn.Close()`), and immediately purge reference pointers from all room shards to eliminate dangling "ghost" goroutines.

### 3.3 Structured Observability: Runtime Logging & Error Tracing
* Maintain detailed structured logging and distributed error tracing across all lifecycle loops (connection, disconnection, read loops, write loops, and Kafka handling).
* Log critical context values (e.g., `RoomID`, `UserID`, `TraceID`) using structured logging (`log/slog`). Ensure network errors or unexpected disconnect reasons are trapped and traced meticulously for effortless production debugging.

---

## 🏗️ Phase 4: Kafka Distributed Integration & Event Flow

Now that the local memory and connection lifecycle are optimized, interface the memory hub with Kafka using this decoupled, non-blocking asynchronous architecture:

### 4.1 Asynchronous Producing with Batching
* Use an asynchronous Kafka producer using `kafka.Writer` from the `segmentio/kafka-go` library.
* Explicitly configure performance batching using these exact properties:
  * `BatchSize: 100`
  * `BatchTimeout: 10 * time.Millisecond`
* This reduces network system call overhead and maximizes throughput under heavy messaging spikes.

### 4.2 Dynamic Consumer Groups for Horizontal Scaling
* Every running instance of this microservice must generate a unique, random, and short-lived Consumer Group ID at startup (e.g., append a `UUID` to the base group name).
* This enforces a **Fan-out/Broadcast** pattern across all horizontal pods, ensuring every instance receives a copy of the message to push to its locally connected WebSocket clients.

### 4.3 Non-blocking Consumer Loop
* The main Kafka consumption loop (`FetchMessage` or `ReadMessage`) must be entirely non-blocking.
* As soon as a message is received, hand it off immediately to an internal worker pool or a Go channel.
* **Never** execute database writes, heavy validation, or complex business logic directly inside the main consumption loop.

### 4.4 Context-Aware Graceful Shutdown
* Implement robust OS signal interception (specifically handling `os.Interrupt` and `syscall.SIGTERM`).
* When a shutdown signal is caught, gracefully flush and close all active Kafka readers and writers to prevent consumer group rebalancing lags in the cluster.

---

### 🛠️ How to use this Guide with AI:
When initiating a code generation request for the Chat/WebSocket structure, prepend your request with:
*"Read the design rules and constraints specified in `README_AI.md`. First, generate the core optimized, sharded WebSocket connection hub with single-marshal fan-outs and heartbeat logic, and then add Phase 4's asynchronous non-blocking Kafka event integration to power it."*

```