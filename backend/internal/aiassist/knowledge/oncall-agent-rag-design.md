# OnCall Agent RAG And Multi-Agent Design

## Scope

This change fills the gap between the resume feature list and the current codebase without introducing Google A2A as a hard dependency. The first implementation keeps agents in the same Go service and passes task context, tool results, and evidence through typed structs. The agent interfaces are kept narrow so an A2A adapter can be added later if agents need to run as independent services.

## Stages

1. Multi-agent evidence orchestration
   - Add alert analysis, log retrieval, knowledge retrieval, and decision agents.
   - Use a shared `TaskContext` and `Evidence` model so every tool result can be traced by source, confidence, and related alert.
   - Route the AIOps endpoint through the orchestrator and keep the old plan-execute-replan path as a fallback.

2. RAG scheduled sync
   - Add a local document syncer that scans configured directories on a fixed schedule.
   - Detect document additions and changes by mtime, size, and SHA256 hash.
   - Reuse the existing knowledge indexing graph for parsing, splitting, embedding, and Milvus writes.
   - Persist document sync metadata in MySQL.

3. RAG evaluation and retrieval tuning
   - Add retrieval metrics for hit rate, TopK recall, citation coverage, no-answer rate, and low-similarity rejection rate.
   - Add retriever configuration for final TopK, coarse TopK multiplier, similarity threshold, metadata filter, and context assembly.
   - Keep defaults compatible with the existing behavior.

4. Verification
   - Add focused unit tests around pure orchestration, sync planning, metrics calculation, and retriever option handling.
   - Run `go test ./...` and review Go changes before the final commit.

## A2A Decision

Google A2A is not required for the first version because all roles run inside one service and share the same tools and persistence layer. A2A becomes useful when alert, log, knowledge, and decision agents are deployed as separate services or need cross-framework interoperability. The local agent interface is the extension point for a future `a2a_adapter`.
