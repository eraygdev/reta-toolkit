# AGENTS.md

## Context & Workflow Rules

* **Code Output Format:** Use the `"Bul:"` and `"Değiştir:"` format for targeted edits, strictly preserving TAB indentations so changes can be applied via `Ctrl+Shift+F`. Ensure the `"Bul:"` block is long and specific enough to match uniquely. Provide the complete file content only when performing extensive refactoring or creating new files.
* **Markdown & README Formatting:** When generating `README.md` or any Markdown file containing nested code blocks, always wrap the outer output in **four backticks** (` ````markdown `) instead of three so inner triple-backtick (` ``` `) blocks do not break rendering or copy-paste structure.
* **Resource Consumption:** Always state the estimated Disk (GB/MB) and RAM (MB/GB) usage before recommending or installing any external tool, runtime, or dependency.
* **Unseen Files:** Never assume or comment on the contents of a file not explicitly provided in the context; directly state `"bunu görmem lazım"`.
* **Conflict Resolution:** If a conflict arises between these rules, exported files, or existing code, do not decide independently—ask for clarification first.
* **Architectural Alignment:** Adhere strictly to the project's existing architecture. If proposing a new design pattern or structural change, explicitly explain **WHY** it is necessary.
* **Handling Uncertainty:** If unsure about an API, library behavior, or edge case, explicitly state `"emin değilim"`. Never guess or fabricate information.
* **Comments & Language Policy:** All inline code comments and docstrings must be professional, natural (human-written style), and strictly in **English**. Never include AI-generated markers (e.g., `// Added by AI`, `// <- ADD THIS`, `// Optimized`). All conversational explanations outside code blocks must be in **Turkish**.
* **Root Cause Debugging:** When an error log or stack trace is shared, briefly explain the **root cause** first before providing the fix.

## Code Quality & Maintainability Standards

* **Intention-Revealing Naming:** Avoid vague abbreviations like `data`, `tmp`, or `res`; use descriptive names (e.g., `parsedConfigLines`, `activeConnections`). Follow **Boolean Naming Conventions** by prefixing boolean variables and functions with state or question verbs (e.g., `isValid`, `hasLeak`, `shouldSkip`).
* **No Magic Numbers or Magic Strings:** Never hardcode unexplained numbers or string literals inside logic. Define them as descriptive **Named Constants** at the top of the file or module (e.g., `MAX_RETRY_COUNT = 3`).
* **Single Responsibility & Parameter Object Pattern:** Follow the **Single Responsibility Principle (SRP)** so every function performs only one specific task. If a function requires more than 3 parameters, group them into a dedicated data structure, object, or model (**Parameter Object**).
* **Low Cognitive Complexity & Guard Clauses:** Avoid deeply nested `if-else` blocks. Validate errors, edge cases, and empty inputs at the start of the function using **Early Return (Guard Clauses)** to keep the main execution path flat and left-aligned.
* **Separation of Concerns (SoC):** Strictly separate **Business Logic** from **I/O operations**. Keep console printing or UI rendering out of data-processing functions; return structured data and let the presentation/CLI layer handle output.
* **Explicit Error Handling & Context Wrapping:** Never swallow or ignore errors silently. Propagate errors upward wrapped with clear operational context (**Error Wrapping / Exception Chaining**) indicating what failed and on which input.
* **Self-Documenting Code:** Avoid redundant comments that merely repeat what the code does. Write concise single-sentence docstrings above functions and use **English** comments strictly to explain **why** non-obvious technical decisions or OS-specific workarounds exist (**"Why" over "What"**).
* **Testability & Documentation Sync:** Design functions to be loosely coupled (**Loosely Coupled**) so independent **Unit Tests** can be written easily.

## Operational Boundaries

* **Always:** Follow the project's native formatter/linter standards, keep dependencies minimal, and write unit tests for new core logic.
* **Ask First:** Before adding any new third-party package/dependency, changing folder structures, or altering public CLI flags/API signatures.
* **Never:** Read, expose, or hardcode `.env` secrets/API keys; delete existing passing tests without replacement; or leave unused variables/dead code behind.

## Project Commands & Definition of Done

* **Test / Lint / Build Commands:** *(Optional: Add project-specific commands here, e.g., `npm test`, `pytest`, `go test ./...`)*
* **Done Means:** The code compiles/runs cleanly without linter warnings, existing and new tests pass, edge cases are handled, and `README.md` is updated if user-facing behavior changed.
