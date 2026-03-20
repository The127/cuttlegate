# ADR 0004: AI development tooling evaluation — Graphiti and Axon

**Date:** 2026-03-20
**Status:** Accepted
**Issue:** #78

## Context

The foundation sprint is the right time to evaluate AI development tooling before domain knowledge accumulates. Two tools were identified: Graphiti (Zep) for AI agent persistent memory via temporal knowledge graphs, and Axon for graph-powered code intelligence exposed via MCP tools to AI agents.

The project uses Claude Code as its primary AI agent. It already has a working context system: ADRs in `docs/adr/`.

## Decision

**Graphiti: Defer.**
**Axon: Defer** (blocked by missing Go support).

Neither tool is adopted in Sprint 1.

## Rationale

**Axon:**

Axon indexes codebases into a structural knowledge graph (KuzuDB, embedded, local) and exposes call graphs, impact analysis, dead code detection, and change coupling via MCP tools. The MCP integration is the right shape for Claude Code — precomputed structural context rather than grep-based flat text search. However, Axon's tree-sitter parsers support Python, TypeScript, and JavaScript only. Cuttlegate is a Go codebase. There is nothing to index. This is a hard blocker with no workaround.

Revisit when Go parser support lands upstream.

**Graphiti:**

Graphiti provides temporally-aware knowledge graph memory with bi-temporal fact tracking, hybrid retrieval (BM25 + semantic + graph traversal), and multiple backend options (Neo4j, FalkorDB, Kuzu, Neptune). The system requires a graph database, a Python environment, and LLM calls per ingestion to extract structured facts.

The existing context system is working — Sprint 1 completed six issues across multiple sessions with no context failures attributable to the memory system. Graphiti's value proposition (temporal queries across a large, frequently-changing knowledge base) applies to a later stage of the project. The infrastructure overhead is not justified now.

If Graphiti is adopted later, it must remain entirely external to the application — no production code may depend on it.

## Consequences

- No new tooling infrastructure is introduced in Sprint 1
- The existing context system remains the agent context mechanism
- Axon should be re-evaluated when Go parser support is confirmed upstream
- Graphiti should be re-evaluated in Sprint 4–5 when the knowledge base complexity warrants a graph query layer
- This ADR supersedes the "evaluate in foundation sprint" rationale from #78 — deferral is the foundation decision
